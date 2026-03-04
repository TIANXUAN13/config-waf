#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
APP_PKG="./cmd"
BIN_PREFIX="safeline"

export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

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

run_source() {
  (
    cd "${ROOT_DIR}"
    go run "${APP_PKG}" "$@"
  )
}

run_host_binary() {
  local bin
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

interactive_menu() {
  echo "Safeline 运行菜单"
  echo "  1) 直接运行源码 (go run ./cmd)"
  echo "  2) 编译本机平台并运行"
  echo "  3) 编译全部平台"
  echo "  4) 选择已编译二进制并运行"
  printf "输入编号: "
  local pick
  read -r pick
  case "${pick}" in
    1) run_source ;;
    2) run_host_binary ;;
    3) build_all ;;
    4) pick_and_run_binary ;;
    *) echo "无效编号"; exit 1 ;;
  esac
}

usage() {
  cat <<'EOF'
Usage:
  ./build/run.sh                      # 交互菜单
  ./build/run.sh --build-all          # 编译多平台
  ./build/run.sh --build-host         # 编译当前平台
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
