#!/usr/bin/env python3
"""Internal OpenVINO GenAI backend for shai."""

from __future__ import annotations

import argparse
import json
import os
import sys
from pathlib import Path


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

    try:
        import openvino_genai as ov_genai

        pipe = ov_genai.LLMPipeline(str(model_path), device)
        config = ov_genai.GenerationConfig()
        config.max_new_tokens = max_tokens

        history = build_chat_history(payload)
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

        emit({"type": "done", "text": assistant})
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
