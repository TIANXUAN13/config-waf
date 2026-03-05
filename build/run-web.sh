#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BIN_PATH="${SCRIPT_DIR}/safeline-darwin-web"

if ! command -v go >/dev/null 2>&1; then
  echo "[run-web] go not found, please install Go 1.20+ first."
  echo "[run-web] download: https://go.dev/dl/"
  exit 1
fi

if [[ -f "${BIN_PATH}" ]]; then
  echo "[run-web] found existing binary: ${BIN_PATH}"
else
  echo "[run-web] binary not found, building: ${BIN_PATH}"
  cd "${ROOT_DIR}"
  export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
  go build -o "${BIN_PATH}" ./cmd
fi

echo "[run-web] starting web console..."
exec "${BIN_PATH}" web "$@"
