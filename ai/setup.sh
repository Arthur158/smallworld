#!/usr/bin/env bash
set -euo pipefail
ai_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
mode=auto
torch_index=""
while (($#)); do
  case "$1" in
    --cpu) mode=cpu; shift ;;
    --cuda) mode=cuda; shift ;;
    --torch-index) torch_index="${2:?Provide a PyTorch wheel index URL}"; shift 2 ;;
    *) echo "Usage: $0 [--cpu|--cuda] [--torch-index URL]" >&2; exit 2 ;;
  esac
done
smallworld_python="${SMALLWORLD_PYTHON:-python3}"
"$smallworld_python" -c 'import sys; assert (3,10) <= sys.version_info < (3,14), "Use Python 3.10 through 3.13 for the pinned PyTorch wheel"'
command -v "${SMALLWORLD_NODE:-node}" >/dev/null || { echo 'Install Node.js 18+ for generated maps.' >&2; exit 1; }
"${SMALLWORLD_NODE:-node}" -e 'if (Number(process.versions.node.split(".")[0]) < 18) throw new Error("Use Node.js 18+")'
if [[ ! -f "$ai_dir/../smallworld-map-generator/src/generation/mapGenerator.js" ]]; then
  (cd "$ai_dir/.." && git submodule update --init --recursive)
fi
command -v go >/dev/null || { echo 'Install Go 1.24 or newer first.' >&2; exit 1; }
if [[ "$mode" == auto ]]; then
  if command -v nvidia-smi >/dev/null && nvidia-smi -L >/dev/null 2>&1; then mode=cuda; else mode=cpu; fi
fi
if [[ -z "$torch_index" ]]; then
  if [[ "$mode" == cuda ]]; then torch_index=https://download.pytorch.org/whl/cu128; else torch_index=https://download.pytorch.org/whl/cpu; fi
fi
if ! "$smallworld_python" -m venv "$ai_dir/.venv"; then
  echo 'Python venv support is required (Ubuntu/Debian: install python3-venv).' >&2
  exit 1
fi
"$ai_dir/.venv/bin/python" -m pip install 'torch==2.8.0' --index-url "$torch_index"
"$ai_dir/.venv/bin/python" -m pip install -e "$ai_dir"
(cd "$ai_dir/../backend2" && go build -o "$ai_dir/bin/botenv" ./cmd/botenv)
"$ai_dir/.venv/bin/smallworld-doctor" --device "$mode"
echo "Ready. Train with: $ai_dir/.venv/bin/smallworld-train --device $mode"
