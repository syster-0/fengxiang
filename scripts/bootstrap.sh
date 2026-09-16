#!/usr/bin/env bash
# tv-opinion-atelier 一键引导（Linux / macOS）
#
# 用法：
#   bash scripts/bootstrap.sh          # 全量引导（二进制 + MediaCrawler + Python 环境 + 浏览器内核）
#   bash scripts/bootstrap.sh --cn     # 使用国内镜像（GOPROXY/pip/playwright/GitHub 代理）
#
# 幂等：每一步检测到已完成即跳过，可重复执行。

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

CN_MIRROR=0
[ "${1:-}" = "--cn" ] || [ "${CN:-0}" = "1" ] && CN_MIRROR=1

say() { printf '\n\033[1;36m==> %s\033[0m\n' "$*"; }

GH_PREFIX=""
if [ "$CN_MIRROR" = 1 ]; then
  GH_PREFIX="https://gh-proxy.com/"
  export GOPROXY="https://goproxy.cn,direct"
  export PIP_INDEX_URL="https://pypi.tuna.tsinghua.edu.cn/simple"
  export PLAYWRIGHT_DOWNLOAD_HOST="https://npmmirror.com/mirrors/playwright/"
  say "已启用国内镜像（gh-proxy / goproxy.cn / 清华 PyPI / npmmirror）"
fi

# ---- 0. /tmp 过小防护（部分机器 /tmp 是 10MB tmpfs，pip/go 下载会爆） ----
if [ -d /tmp ] && [ "$(df -k /tmp | awk 'NR==2{print $4}')" -lt 2097152 ]; then
  BIG_TMP="$ROOT/.tmp"; mkdir -p "$BIG_TMP"
  export TMPDIR="$BIG_TMP" GOTMPDIR="$BIG_TMP"
  say "检测到 /tmp 空间过小，临时目录改为 $BIG_TMP"
fi

# ---- 1. tvop 可执行：按平台挑预编译二进制，缺则 go build ----
say "步骤 1/5：tvop 可执行"
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"; ARCH="$(uname -m)"
case "$ARCH" in x86_64|amd64) ARCH=amd64 ;; aarch64|arm64) ARCH=arm64 ;; esac
BIN="bin/tvop-$OS-$ARCH"
# Linux 矩阵名与默认名同体：bin/tvop 本身即 linux/amd64 精简版。
if [ "$OS" = linux ] && [ "$ARCH" = amd64 ] && [ -x bin/tvop ] && [ ! -x "$BIN" ]; then
  cp -f bin/tvop "$BIN"
fi
if [ -x "$BIN" ]; then
  cp -f "$BIN" bin/tvop && chmod +x bin/tvop
  say "  已启用预编译二进制 $BIN"
elif command -v go >/dev/null 2>&1; then
  go build -o bin/tvop ./cmd/tvop
  say "  无预编译匹配，已用本地 Go 编译"
else
  echo "  ✗ 既无匹配二进制（$BIN）也无 Go 工具链。请安装 Go 1.24+ 后重跑。" >&2
  exit 1
fi
./bin/tvop doctor >/dev/null 2>&1 || true

# ---- 2. MediaCrawler 源码 ----
say "步骤 2/5：MediaCrawler 嵌入"
MC="$ROOT/third_party/MediaCrawler"
if [ -f "$MC/main.py" ]; then
  say "  已存在，跳过"
else
  mkdir -p third_party
  git clone --depth 1 "${GH_PREFIX}https://github.com/NanmiCoder/MediaCrawler" "$MC"
  say "  已克隆（上游锚点 commit 见 third_party/MediaCrawler/VENDORED.md）"
fi

# ---- 3. Python 虚拟环境 ----
say "步骤 3/5：Python 环境（jieba 需要 setuptools<70 构建约束）"
cd "$MC"
# 健康判据：venv 存在且关键依赖可导入 —— 半成品 venv（中断残留）自动进入修复分支。
venv_healthy() {
  [ -x ".venv/bin/python" ] || [ -x ".venv/Scripts/python.exe" ] || return 1
  .venv/bin/python -c "import typer, fastapi, jieba, playwright, httpx" >/dev/null 2>&1 \
    || .venv/Scripts/python.exe -c "import typer, fastapi, jieba, playwright, httpx" >/dev/null 2>&1
}
install_deps() {
  PIP=".venv/bin/pip"; [ "$OS" = windows ] && PIP=".venv/Scripts/pip.exe"
  printf 'setuptools<70\nwheel<0.46\n' > .build-constraints.txt
  if [ "$OS" = windows ]; then
    env PIP_CONSTRAINT=.build-constraints.txt "$PIP" install -r requirements.txt
  else
    env PIP_CONSTRAINT="$MC/.build-constraints.txt" "$PIP" install -r requirements.txt
  fi
}
if ! venv_healthy; then
  if [ ! -x ".venv/bin/python" ] && [ ! -x ".venv/Scripts/python.exe" ]; then
    if command -v uv >/dev/null 2>&1; then
      uv venv --seed .venv
    else
      python3 -m venv .venv
      ".venv/bin/pip" install -U "pip<24.1" "setuptools<70" "wheel<0.46"
    fi
  fi
  install_deps
  say "  venv 建好 / 已修复"
else
  say "  已存在且健康，跳过"
fi

# ---- 4. 浏览器内核 ----
say "步骤 4/5：Playwright Chromium"
PW=".venv/bin/playwright"; [ "$OS" = windows ] && PW=".venv/Scripts/playwright.exe"
if "$PW" install chromium >/dev/null 2>&1 || "$PW" install chromium; then
  say "  Chromium 就位"
fi

# ---- 5. 自检 ----
say "步骤 5/5：自检"
cd "$ROOT"
./bin/tvop doctor
./bin/tvop okf validate | tail -2
say "引导完成。生成一条试试：./bin/tvop search 分区 虚标"
