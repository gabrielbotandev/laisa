#!/usr/bin/env bash
set -euo pipefail

PREFIX="${HOME}/.local/bin"

usage() {
  cat <<EOF
Usage: ./scripts/uninstall.sh [--prefix PATH]

Default prefix: ~/.local/bin
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --prefix)
      PREFIX="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

OS="$(uname -s)"
if [[ "$OS" == "Darwin" ]]; then
  DATA_DIR="${HOME}/Library/Application Support/shai"
  CONFIG_DIR="$DATA_DIR"
  CACHE_DIR="${HOME}/Library/Caches/shai"
else
  DATA_DIR="${XDG_DATA_HOME:-${HOME}/.local/share}/shai"
  CONFIG_DIR="${XDG_CONFIG_HOME:-${HOME}/.config}/shai"
  CACHE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/shai"
fi

BIN="${PREFIX}/shai"
if [[ -f "$BIN" ]]; then
  rm -f "$BIN"
  echo "Removed $BIN"
else
  echo "Binary not found at $BIN"
fi

read -r -p "Remove config at ${CONFIG_DIR}? [y/N] " ans
if [[ "${ans,,}" == "y" ]]; then
  rm -f "${CONFIG_DIR}/config.yaml"
  echo "Removed config"
fi

read -r -p "Remove cache at ${CACHE_DIR}? [y/N] " ans
if [[ "${ans,,}" == "y" ]]; then
  rm -rf "$CACHE_DIR"
  echo "Removed cache"
fi

read -r -p "Remove ALL downloaded models at ${DATA_DIR}/models? [y/N] " ans
if [[ "${ans,,}" == "y" ]]; then
  rm -rf "${DATA_DIR}/models"
  mkdir -p "${DATA_DIR}/models"
  echo "Removed models"
fi

read -r -p "Remove Python venv and backend at ${DATA_DIR}? [y/N] " ans
if [[ "${ans,,}" == "y" ]]; then
  rm -rf "${DATA_DIR}/.venv" "${DATA_DIR}/backend"
  echo "Removed backend and venv"
fi

echo "Uninstall complete."
