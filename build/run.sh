#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_PKG="./cmd"
BIN_PREFIX="safeline"

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
DEFAULT_CONFIG_FILE="${SCRIPT_DIR}/token.csv"
DEFAULT_WEB_LISTEN="0.0.0.0:28000"

ensure_go() {
  if command -v go >/dev/null 2>&1; then
    return
  fi
  echo "[error] 未检测到 Go 环境（go 命令不存在）"
  echo "请先安装 Go 1.20+，再重新执行脚本。"
  echo "安装参考: https://go.dev/dl/"
  echo "macOS(示例): brew install go"
  echo "Ubuntu(示例): sudo apt-get update && sudo apt-get install -y golang-go"
  exit 1
}

host_goos() {
  local s
  s="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "${s}" in
    darwin*) echo "darwin" ;;
    linux*) echo "linux" ;;
    msys*|mingw*|cygwin*) echo "windows" ;;
    *) echo "unknown" ;;
  esac
}

host_goarch() {
  local m
  m="$(uname -m)"
  case "${m}" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "${m}" ;;
  esac
}

host_bin_name() {
  local goos arch
  goos="$(host_goos)"
  arch="$(host_goarch)"
  if [[ "${goos}" == "windows" ]]; then
    echo "${BIN_PREFIX}-${goos}-${arch}.exe"
  else
    echo "${BIN_PREFIX}-${goos}-${arch}"
  fi
}

build_one() {
  ensure_go
  local goos="$1"
  local goarch="$2"
  local out
  if [[ "${goos}" == "windows" ]]; then
    out="${SCRIPT_DIR}/${BIN_PREFIX}-${goos}-${goarch}.exe"
  else
    out="${SCRIPT_DIR}/${BIN_PREFIX}-${goos}-${goarch}"
  fi
  echo "[build] ${goos}/${goarch} -> ${out}"
  (
    cd "${ROOT_DIR}"
    GOOS="${goos}" GOARCH="${goarch}" go build -o "${out}" "${APP_PKG}"
  )
}

build_all() {
  build_one darwin amd64
  build_one darwin arm64
  build_one linux amd64
  build_one linux arm64
  build_one windows amd64
  build_one windows arm64
  echo "[build] done"
}

build_host() {
  build_one "$(host_goos)" "$(host_goarch)"
}

build_target() {
  local goos="${1:-}"
  local goarch="${2:-}"
  if [[ -z "${goos}" || -z "${goarch}" ]]; then
    echo "[build] 用法: --build-target <goos> <goarch>"
    exit 1
  fi
  case "${goos}" in
    mac|macos|osx) goos="darwin" ;;
    win) goos="windows" ;;
  esac
  build_one "${goos}" "${goarch}"
}

pick_target_to_build() {
  local targets=(
    "macOS amd64|darwin|amd64"
    "macOS arm64|darwin|arm64"
    "Linux amd64|linux|amd64"
    "Linux arm64|linux|arm64"
    "Windows amd64|windows|amd64"
    "Windows arm64|windows|arm64"
  )
  local idx pick line label goos goarch
  echo "选择要编译的平台与架构："
  for idx in "${!targets[@]}"; do
    line="${targets[idx]}"
    label="${line%%|*}"
    printf "  %d) %s\n" "$((idx + 1))" "${label}"
  done
  printf "输入编号: "
  read -r pick
  if ! [[ "${pick}" =~ ^[0-9]+$ ]] || (( pick < 1 || pick > ${#targets[@]} )); then
    echo "无效编号"
    exit 1
  fi
  line="${targets[pick-1]}"
  goos="$(echo "${line}" | cut -d'|' -f2)"
  goarch="$(echo "${line}" | cut -d'|' -f3)"
  build_one "${goos}" "${goarch}"
}

run_source() {
  ensure_go
  if should_inject_default_config "$@"; then
    set -- --config "${DEFAULT_CONFIG_FILE}" "$@"
  fi
  (
    cd "${ROOT_DIR}"
    go run "${APP_PKG}" "$@"
  )
}

run_host_binary() {
  local bin
  if should_inject_default_config "$@"; then
    set -- --config "${DEFAULT_CONFIG_FILE}" "$@"
  fi
  bin="$(host_bin_name)"
  if [[ ! -f "${SCRIPT_DIR}/${bin}" ]]; then
    echo "[run] host binary not found, building first..."
    build_host
  fi
  echo "[run] ${SCRIPT_DIR}/${bin} $*"
  exec "${SCRIPT_DIR}/${bin}" "$@"
}

is_windows_name() {
  [[ "$1" == *.exe ]]
}

pick_and_run_binary() {
  local bins=()
  local f idx pick
  if should_inject_default_config "$@"; then
    set -- --config "${DEFAULT_CONFIG_FILE}" "$@"
  fi
  while IFS= read -r -d '' f; do
    bins+=("${f}")
  done < <(find "${SCRIPT_DIR}" -maxdepth 1 -type f -name "${BIN_PREFIX}-*" -print0 | sort -z)

  if [[ ${#bins[@]} -eq 0 ]]; then
    echo "[run] no binary found in ${SCRIPT_DIR}, run build first."
    exit 1
  fi

  echo "选择要运行的可执行文件："
  for idx in "${!bins[@]}"; do
    printf "  %d) %s\n" "$((idx + 1))" "$(basename "${bins[idx]}")"
  done
  printf "输入编号: "
  read -r pick
  if ! [[ "${pick}" =~ ^[0-9]+$ ]] || (( pick < 1 || pick > ${#bins[@]} )); then
    echo "无效编号"
    exit 1
  fi
  f="${bins[pick-1]}"

  if is_windows_name "${f}" && [[ "$(host_goos)" != "windows" ]]; then
    echo "[run] 当前系统无法直接运行 windows 可执行文件: $(basename "${f}")"
    exit 1
  fi
  if ! is_windows_name "${f}" && [[ "$(host_goos)" == "windows" ]]; then
    echo "[run] 当前系统请运行 .exe 文件"
    exit 1
  fi

  echo "[run] ${f} $*"
  exec "${f}" "$@"
}

should_inject_default_config() {
  if [[ ! -f "${DEFAULT_CONFIG_FILE}" ]]; then
    return 1
  fi
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --config|-c|--config=*)
        return 1
        ;;
    esac
    shift
  done
  return 0
}

interactive_menu() {
  echo "Safeline 运行菜单"
  echo "  1) 直接运行源码 (go run ./cmd)"
  echo "  2) 编译本机平台并运行"
  echo "  3) 编译全部平台"
  echo "  4) 选择已编译二进制并运行"
  echo "  5) 选择平台架构并编译"
  printf "输入编号: "
  local pick
  read -r pick
  case "${pick}" in
    1)
      echo "[run] 未指定子命令，默认启动 Web 控制台: ${DEFAULT_WEB_LISTEN}"
      run_source web --listen "${DEFAULT_WEB_LISTEN}"
      ;;
    2)
      echo "[run] 未指定子命令，默认启动 Web 控制台: ${DEFAULT_WEB_LISTEN}"
      run_host_binary web --listen "${DEFAULT_WEB_LISTEN}"
      ;;
    3) build_all ;;
    4) pick_and_run_binary ;;
    5) pick_target_to_build ;;
    *) echo "无效编号"; exit 1 ;;
  esac
}

usage() {
  cat <<'EOF'
Usage:
  ./build/run.sh                      # 交互菜单
  ./build/run.sh --build-all          # 编译多平台
  ./build/run.sh --build-host         # 编译当前平台
  ./build/run.sh --build-target linux arm64
  ./build/run.sh --build-target mac arm64
  ./build/run.sh --run-source [args]  # go run ./cmd
  ./build/run.sh --run-host [args]    # 运行本机二进制（不存在则先编译）
  ./build/run.sh --pick-run [args]    # 编号选择已有二进制并运行
EOF
}

main() {
  if [[ $# -eq 0 ]]; then
    interactive_menu
    return
  fi

  case "$1" in
    -h|--help)
      usage
      ;;
    --build-all)
      build_all
      ;;
    --build-host)
      build_host
      ;;
    --build-target)
      shift
      build_target "${1:-}" "${2:-}"
      ;;
    --run-source)
      shift
      run_source "$@"
      ;;
    --run-host)
      shift
      run_host_binary "$@"
      ;;
    --pick-run)
      shift
      pick_and_run_binary "$@"
      ;;
    *)
      echo "unknown option: $1"
      usage
      exit 1
      ;;
  esac
}

main "$@"
