# shai

**shai** is a terminal-first local AI assistant for Linux and macOS. It runs OpenVINO GenAI models on your machine with an interactive Bubble Tea TUI or one-shot CLI prompts.

The user-facing command is only:

```bash
shai
```

Python, model runners, and downloads are managed internally.

## Requirements

- Linux or macOS
- Go 1.22+ (to build)
- Python 3.10+
- Enough RAM/disk for your chosen model

## Installation

```bash
git clone <repo-url> shai
cd shai
./scripts/install.sh
```

Optional flags:

```bash
./scripts/install.sh --prefix /usr/local/bin
./scripts/install.sh --with-model OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
```

If `~/.local/bin` is not in your `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

## First run

1. Install (above).
2. Download a compatible OpenVINO model.
3. Set `default_model` in config or pass `--model`.

```bash
shai --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
shai "Say hello in one sentence"
shai
```

## Downloading a model

Interactive:

```bash
shai --download
```

Non-interactive:

```bash
shai --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
shai --download OpenVINO/Mistral-7B-Instruct-v0.3-int4-cw-ov --name Mistral-7B --force
```

## Running prompts

```bash
shai "Explain Docker in two sentences"
shai --model Phi-3.5-mini "What is OpenVINO?"
shai --device CPU --max-tokens 500 "Short answer only"
cat main.go | shai "Explain this code"
```

List models, show config:

```bash
shai --list-models
shai --config
shai --version
shai --help
```

## TUI usage

```bash
shai
```

Open the interactive UI: header (model/device), scrollable chat, input area, status bar.

### Keyboard shortcuts

| Key | Action |
|-----|--------|
| Enter | Send prompt |
| Ctrl+C | Quit |
| Esc | Cancel input / back |
| Ctrl+L | Clear conversation |
| Ctrl+H | Help |
| Ctrl+M | Model picker |
| Ctrl+D | Download model |
| Ctrl+S | Settings |

### Slash commands

| Command | Description |
|---------|-------------|
| `/help` | Show help |
| `/models` | Model picker |
| `/model <name>` | Switch model |
| `/download <repo>` | Download model |
| `/device CPU\|NPU\|AUTO` | Set device |
| `/tokens <n>` | Set max tokens |
| `/clear` | Clear chat |
| `/config` | Show config |
| `/quit` | Exit |

Chat history is kept in memory for the current session only.

## Model compatibility

`shai` can download any public Hugging Face repository, but it can only **run** models packaged for OpenVINO GenAI.

Recommended sources:

- [OpenVINO organization on Hugging Face](https://huggingface.co/OpenVINO)
- Suffixes such as `-ov`, `-int4-ov`, `-int4-cw-ov`

Standard Transformers checkpoints (for example `mistralai/Mistral-7B-Instruct-v0.3`) must be converted to OpenVINO before use.

## Storage paths

| Purpose | Linux | macOS |
|---------|-------|-------|
| Data | `$XDG_DATA_HOME/shai` or `~/.local/share/shai` | `~/Library/Application Support/shai` |
| Models | `<data>/models` | `<data>/models` |
| Config | `$XDG_CONFIG_HOME/shai` or `~/.config/shai` | `~/Library/Application Support/shai` |
| Cache | `$XDG_CACHE_HOME/shai` or `~/.cache/shai` | `~/Library/Caches/shai` |
| Python venv | `<data>/.venv` | `<data>/.venv` |

## NPU notes

`--device NPU` is experimental and depends on your hardware and OpenVINO build. If inference fails, try CPU:

```bash
shai --device CPU "Say hello"
```

## Troubleshooting

**Python backend is not installed**

```bash
./scripts/install.sh
```

**No model found**

```bash
shai --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
```

**Gated Hugging Face models**

Export `HF_TOKEN` with a valid Hugging Face token before downloading.

**Command not found**

Ensure `~/.local/bin` is in your `PATH`.

**OpenVINO library errors (`libopenvino.so` not found)**

Reinstall so `openvino`, `openvino-genai`, and `openvino-tokenizers` versions match:

```bash
./scripts/install.sh
```

If you upgraded packages manually, align them in `~/.local/share/shai/backend/requirements.txt` and run `pip install -r` inside the venv.

## Uninstall

```bash
./scripts/uninstall.sh
```

Models are only removed if you explicitly confirm.

## Build from source

```bash
go mod tidy
go build -o shai .
./scripts/install.sh
shai --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
shai "Say hello in one sentence"
shai
```

## License

Apache-2.0 (see project license file if provided).
