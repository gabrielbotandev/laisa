#!/usr/bin/env python3
"""Internal OpenVINO GenAI backend for shai."""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path

DEFAULT_CONTEXT_LIMIT = 32768


def emit(obj: dict) -> None:
    print(json.dumps(obj, ensure_ascii=False), flush=True)


def emit_error(message: str, code: int = 1) -> None:
    emit({"type": "error", "message": message})
    print(message, file=sys.stderr)
    sys.exit(code)


def dir_nonempty(path: Path) -> bool:
    if not path.is_dir():
        return False
    for entry in path.iterdir():
        if entry.name.startswith("."):
            continue
        return True
    return False


def read_context_limit(model_path: str) -> int:
    config_path = Path(model_path) / "config.json"
    if not config_path.is_file():
        return DEFAULT_CONTEXT_LIMIT
    try:
        with config_path.open(encoding="utf-8") as f:
            cfg = json.load(f)
    except (OSError, json.JSONDecodeError):
        return DEFAULT_CONTEXT_LIMIT

    value = cfg.get("max_position_embeddings")
    if isinstance(value, int) and value > 0:
        return value

    text_cfg = cfg.get("text_config")
    if isinstance(text_cfg, dict):
        value = text_cfg.get("max_position_embeddings")
        if isinstance(value, int) and value > 0:
            return value

    return DEFAULT_CONTEXT_LIMIT


def token_count(tokenizer, text: str) -> int:
    if not text:
        return 0
    try:
        encoded = tokenizer.encode(text)
        ids = encoded.input_ids
        if hasattr(ids, "shape"):
            shape = ids.shape
            if len(shape) == 0:
                return 0
            return int(shape[-1])
        if hasattr(ids, "__len__"):
            return len(ids)
    except Exception:  # noqa: BLE001
        pass
    return max(1, len(text) // 4)


def count_history_tokens(pipe, history) -> int:
    tokenizer = pipe.get_tokenizer()

    if hasattr(history, "get_prompt"):
        try:
            prompt = history.get_prompt()
            if prompt:
                return token_count(tokenizer, str(prompt))
        except Exception:  # noqa: BLE001
            pass

    # Fallback: sum tokens per message content.
    total = 0
    try:
        if hasattr(history, "messages"):
            messages = history.messages
        elif hasattr(history, "get_messages"):
            messages = history.get_messages()
        else:
            messages = list(history)
    except Exception:  # noqa: BLE001
        messages = []

    for msg in messages:
        if isinstance(msg, dict):
            content = msg.get("content", "")
        else:
            content = getattr(msg, "content", str(msg))
        if content:
            total += token_count(tokenizer, str(content))
    return total


def build_chat_history(payload: dict):
    import openvino_genai as ov_genai

    history = ov_genai.ChatHistory()
    system_prompt = (payload.get("system_prompt") or "").strip()
    has_system = False

    messages = payload.get("messages") or []
    for msg in messages:
        role = msg.get("role", "user")
        content = msg.get("content", "")
        if not content:
            continue
        if role == "system":
            has_system = True
        if role in ("user", "assistant", "system"):
            history.append({"role": role, "content": content})

    if system_prompt and not has_system:
        history.append({"role": "system", "content": system_prompt})

    prompt = (payload.get("prompt") or "").strip()
    stdin_context = (payload.get("stdin_context") or "").strip()

    if prompt or stdin_context:
        user_text = prompt
        if stdin_context:
            if user_text:
                user_text = f"Context from stdin:\n{stdin_context}\n\n{prompt}"
            else:
                user_text = f"Context from stdin:\n{stdin_context}"
        if not messages:
            history.append({"role": "user", "content": user_text.strip()})

    return history


def append_assistant(history, text: str):
    if text:
        history.append({"role": "assistant", "content": text})


def cmd_generate(_args: argparse.Namespace) -> int:
    try:
        raw = sys.stdin.read()
        payload = json.loads(raw) if raw.strip() else {}
    except json.JSONDecodeError as exc:
        emit_error(f"Invalid JSON input: {exc}")
        return 1

    model_path = payload.get("model_path")
    if not model_path:
        emit_error("model_path is required")
        return 1

    device = (payload.get("device") or "CPU").upper()
    max_tokens = int(payload.get("max_tokens") or 1000)
    context_limit = read_context_limit(str(model_path))

    try:
        import openvino_genai as ov_genai

        pipe = ov_genai.LLMPipeline(str(model_path), device)
        emit({"type": "ready", "context_limit": context_limit})

        config = ov_genai.GenerationConfig()
        config.max_new_tokens = max_tokens

        history = build_chat_history(payload)
        prompt_tokens = count_history_tokens(pipe, history)
        full_text: list[str] = []

        def streamer(subword: str):
            text = str(subword)
            full_text.append(text)
            emit({"type": "token", "text": text})
            return ov_genai.StreamingStatus.RUNNING

        result = pipe.generate(history, config, streamer)
        assistant = ""
        if hasattr(result, "texts") and result.texts:
            assistant = result.texts[0]
        elif isinstance(result, str):
            assistant = result
        else:
            assistant = "".join(full_text)

        history_after = build_chat_history(payload)
        append_assistant(history_after, assistant)
        context_tokens = count_history_tokens(pipe, history_after)
        completion_tokens = max(0, context_tokens - prompt_tokens)

        usage = {
            "prompt_tokens": prompt_tokens,
            "completion_tokens": completion_tokens,
            "context_tokens": context_tokens,
            "context_limit": context_limit,
        }
        emit({"type": "usage", "usage": usage})
        emit({"type": "done", "text": assistant, "usage": usage})
        return 0
    except Exception as exc:  # noqa: BLE001
        emit_error(str(exc))
        return 1


def cmd_download(args: argparse.Namespace) -> int:
    repo = args.repo
    local_dir = Path(args.local_dir)
    revision = args.revision or None
    force = args.force

    if local_dir.exists() and dir_nonempty(local_dir) and not force:
        emit_error(
            f"Destination already exists and is not empty: {local_dir}\n"
            "Use --force to overwrite."
        )
        return 1

    local_dir.mkdir(parents=True, exist_ok=True)

    try:
        from huggingface_hub import snapshot_download

        path = snapshot_download(
            repo_id=repo,
            local_dir=str(local_dir),
            revision=revision,
            force_download=force,
        )
        emit({"type": "done", "path": path})
        return 0
    except Exception as exc:  # noqa: BLE001
        emit_error(str(exc))
        return 1


def main() -> int:
    parser = argparse.ArgumentParser(description="shai internal backend")
    sub = parser.add_subparsers(dest="command", required=True)

    gen = sub.add_parser("generate", help="Run text generation (JSON on stdin)")
    gen.set_defaults(func=cmd_generate)

    dl = sub.add_parser("download", help="Download a Hugging Face model")
    dl.add_argument("--repo", required=True)
    dl.add_argument("--local-dir", required=True)
    dl.add_argument("--revision", default="")
    dl.add_argument("--force", action="store_true")
    dl.set_defaults(func=cmd_download)

    args = parser.parse_args()
    return int(args.func(args))


if __name__ == "__main__":
    os.environ.setdefault("PYTHONUNBUFFERED", "1")
    raise SystemExit(main())
