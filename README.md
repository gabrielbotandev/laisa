# Laisa

**Laisa** is a terminal-first local AI assistant for Linux and macOS. It lets you run Hugging Face models locally with OpenVINO GenAI, using either an interactive Bubble Tea TUI or fast one-shot CLI prompts.

The user-facing command is only:

```bash
laisa
```

Python, model runners, and downloads are managed internally.

## Requirements

- Linux or macOS
- Go 1.22+ (to build)
- Python 3.10+
- Enough RAM/disk for your chosen model

## Installation

```bash
git clone https://github.com/gabrielbotandev/laisa.git
cd laisa
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
laisa --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
laisa "Say hello in one sentence"
laisa
```

## Downloading a model

Interactive:

```bash
laisa --download
```

Non-interactive:

```bash
laisa --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
laisa --download OpenVINO/Mistral-7B-Instruct-v0.3-int4-cw-ov --name Mistral-7B --force
```

## Running prompts

```bash
laisa "Explain Docker in two sentences"
laisa --model Phi-3.5-mini "What is OpenVINO?"
laisa --device CPU --max-tokens 500 "Short answer only"
cat main.go | laisa "Explain this code"
```

List models, show config:

```bash
laisa --list-models
laisa --config
laisa --version
laisa --help
```

## TUI usage

```bash
laisa
```

Open the interactive UI: scrollable transcript on top, composer below it, and a footer with cwd/git branch, session token totals, and context usage (used % vs model limit).

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
| `/tokens <n>` | Set max output tokens |
| `/clear` | Clear chat |
| `/config` | Show config |
| `/quit` | Exit |

Chat history is kept in memory for the current session only.

## Model compatibility

`laisa` can download any public Hugging Face repository, but it can only **run** models packaged for OpenVINO GenAI.

Recommended sources:

- [OpenVINO organization on Hugging Face](https://huggingface.co/OpenVINO)
- Suffixes such as `-ov`, `-int4-ov`, `-int4-cw-ov`

Standard Transformers checkpoints (for example `mistralai/Mistral-7B-Instruct-v0.3`) must be converted to OpenVINO before use.

## Storage paths

| Purpose | Linux | macOS |
|---------|-------|-------|
| Data | `$XDG_DATA_HOME/laisa` or `~/.local/share/laisa` | `~/Library/Application Support/laisa` |
| Models | `<data>/models` | `<data>/models` |
| Config | `$XDG_CONFIG_HOME/laisa` or `~/.config/laisa` | `~/Library/Application Support/laisa` |
| Cache | `$XDG_CACHE_HOME/laisa` or `~/.cache/laisa` | `~/Library/Caches/laisa` |
| Python venv | `<data>/.venv` | `<data>/.venv` |

## NPU notes

`--device NPU` is experimental and depends on your hardware and OpenVINO build. If inference fails, try CPU:

```bash
laisa --device CPU "Say hello"
```

## Troubleshooting

**Python backend is not installed**

```bash
./scripts/install.sh
```

**No model found**

```bash
laisa --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
```

**Gated Hugging Face models**

Export `HF_TOKEN` with a valid Hugging Face token before downloading.

**Command not found**

Ensure `~/.local/bin` is in your `PATH`.

**After upgrading laisa (Python dependency bump)**

Recreate the venv so `openvino-genai` 2026.1.x and `huggingface_hub` 1.16.x install cleanly:

```bash
rm -rf ~/.local/share/laisa/.venv
./scripts/install.sh
```

On macOS, use `~/Library/Application Support/laisa/.venv` instead of `~/.local/share/laisa/.venv`.

**OpenVINO library errors (`libopenvino.so` not found)**

Reinstall the venv (do not pin `openvino` separately — `openvino-genai` pulls matching packages):

```bash
rm -rf ~/.local/share/laisa/.venv
./scripts/install.sh
```

**Python 3.14**

`openvino-genai` wheels on PyPI may lag behind the newest Python. If `pip install` fails, create the venv with Python 3.12 or 3.13 (`python3.13 -m venv ...`).

## Uninstall

```bash
./scripts/uninstall.sh
```

Models are only removed if you explicitly confirm.

## Build from source

```bash
go mod tidy
go build -o laisa .
./scripts/install.sh
laisa --download OpenVINO/Phi-3.5-mini-instruct-int4-cw-ov --name Phi-3.5-mini
laisa "Say hello in one sentence"
laisa
```

## License

Licensed under the MIT License; see [LICENSE](LICENSE).
