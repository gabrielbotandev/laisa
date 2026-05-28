#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PREFIX="${HOME}/.local/bin"
WITH_MODEL=""
WITH_NAME=""
GO_LDFLAGS="-X github.com/shai/shai/internal/version.Version=0.1.0"

usage() {
  cat <<EOF
Usage: ./scripts/install.sh [options]

Options:
  --prefix PATH       Install shai binary to PATH (default: ~/.local/bin)
  --with-model REPO   Download a Hugging Face model after install
  --name NAME         Local model name (used with --with-model)
  -h, --help          Show this help
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)
      PREFIX="$2"
      shift 2
      ;;
    --with-model)
      WITH_MODEL="$2"
      shift 2
      ;;
    --name)
      WITH_NAME="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      usage
      exit 1
      ;;
  esac
done

OS="$(uname -s)"
case "$OS" in
  Linux|Darwin) ;;
  *)
    echo "Unsupported OS: $OS (Linux and macOS only)" >&2
    exit 1
    ;;
esac

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required but not installed." >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required but not installed." >&2
  exit 1
fi

PY_VER="$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')"
PY_MAJOR="$(echo "$PY_VER" | cut -d. -f1)"
PY_MINOR="$(echo "$PY_VER" | cut -d. -f2)"
if [[ "$PY_MAJOR" -lt 3 ]] || [[ "$PY_MAJOR" -eq 3 && "$PY_MINOR" -lt 10 ]]; then
  echo "Python 3.10+ is required (found $PY_VER)." >&2
  exit 1
fi

mkdir -p "$PREFIX"

echo "Building shai..."
(cd "$ROOT" && go build -ldflags "$GO_LDFLAGS" -o "$PREFIX/shai" .)

# Resolve data dir (same logic as Go app)
if [[ "$OS" == "Darwin" ]]; then
  DATA_DIR="${HOME}/Library/Application Support/shai"
  CONFIG_DIR="$DATA_DIR"
  CACHE_DIR="${HOME}/Library/Caches/shai"
else
  DATA_DIR="${XDG_DATA_HOME:-${HOME}/.local/share}/shai"
  CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/shai"
  CACHE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/shai"
fi

MODELS_DIR="${DATA_DIR}/models"
BACKEND_DIR="${DATA_DIR}/backend"
VENV_DIR="${DATA_DIR}/.venv"

mkdir -p "$DATA_DIR" "$CONFIG_DIR" "$CACHE_DIR" "$MODELS_DIR" "$BACKEND_DIR"

echo "Installing backend files..."
cp "$ROOT/internal/backend/runner.py" "$BACKEND_DIR/runner.py"
chmod 755 "$BACKEND_DIR/runner.py"
cp "$ROOT/internal/backend/requirements.txt" "$BACKEND_DIR/requirements.txt"
echo "0.1.0" > "$BACKEND_DIR/backend.version"

echo "Creating Python virtual environment..."
python3 -m venv "$VENV_DIR"
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
pip install --upgrade pip
pip install -r "$BACKEND_DIR/requirements.txt"
deactivate

if [[ ! -f "${CONFIG_DIR}/config.yaml" ]]; then
  mkdir -p "$CONFIG_DIR"
  cat > "${CONFIG_DIR}/config.yaml" <<'YAML'
default_model: ""
default_device: CPU
max_tokens: 1000
system_prompt: |
  You are a helpful local AI assistant running on the user's laptop.
  Be concise, practical, and direct.
  For code questions, give useful explanations and runnable examples when appropriate.
  You do not have live internet access.
YAML
fi

if [[ -n "$WITH_MODEL" ]]; then
  NAME_ARGS=()
  if [[ -n "$WITH_NAME" ]]; then
    NAME_ARGS=(--name "$WITH_NAME")
  fi
  echo "Downloading model $WITH_MODEL..."
  "$PREFIX/shai" --download "$WITH_MODEL" "${NAME_ARGS[@]}"
fi

case ":$PATH:" in
  *":${PREFIX}:"*) ;;
  *)
    echo ""
    echo "Add $PREFIX to your PATH:"
    echo "  export PATH=\"${PREFIX}:\$PATH\""
    ;;
esac

echo ""
echo "Installed shai to $PREFIX/shai"
echo "Data directory: $DATA_DIR"
