# tv-opinion-atelier 一键引导（Windows PowerShell 5.1+）
#
# 用法：
#   powershell -ExecutionPolicy Bypass -File scripts\bootstrap.ps1 [-CN]
#
# -CN 开关启用国内镜像（gh-proxy / goproxy.cn / 清华 PyPI / npmmirror）。
# 幂等：每一步检测到已完成即跳过。

param([switch]$CN)

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

function Say($m) { Write-Host "`n==> $m" -ForegroundColor Cyan }

# ---- 镜像 ----
$GH = ""
if ($CN -or $env:CN -eq "1") {
  $GH = "https://gh-proxy.com/"
  $env:GOPROXY = "https://goproxy.cn,direct"
  $env:PIP_INDEX_URL = "https://pypi.tuna.tsinghua.edu.cn/simple"
  $env:PLAYWRIGHT_DOWNLOAD_HOST = "https://npmmirror.com/mirrors/playwright/"
  Say "已启用国内镜像"
}

# ---- 1. tvop 可执行 ----
Say "步骤 1/5：tvop 可执行"
$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "x86" }
$bin = "bin\tvop-windows-$arch.exe"
if (-not ($arch -eq "amd64")) { $bin = "" }
if ($bin -and (Test-Path $bin)) {
  Copy-Item -Force $bin "bin\tvop.exe"
  Say "  已启用预编译二进制 $bin"
}
elseif (Get-Command go -ErrorAction SilentlyContinue) {
  go build -o bin\tvop.exe ./cmd/tvop
  Say "  无预编译匹配，已用本地 Go 编译"
}
else {
  Write-Host "  x 既无匹配二进制也无 Go 工具链。请安装 Go 1.24+ 后重跑。" -ForegroundColor Red
  exit 1
}

# ---- 2. MediaCrawler 源码 ----
Say "步骤 2/5：MediaCrawler 嵌入"
$MC = Join-Path $Root "third_party\MediaCrawler"
if (Test-Path (Join-Path $MC "main.py")) {
  Say "  已存在，跳过"
}
else {
  New-Item -ItemType Directory -Force -Path third_party | Out-Null
  git clone --depth 1 "${GH}https://github.com/NanmiCoder/MediaCrawler" $MC
  Say "  已克隆"
}

# ---- 3. Python 虚拟环境 ----
Say "步骤 3/5：Python 环境（jieba 需要 setuptools<70 构建约束）"
Set-Location $MC
# 健康判据：venv 存在且关键依赖可导入 —— 半成品 venv（中断残留）自动进入修复分支。
function Test-VenvHealthy {
  if (-not (Test-Path ".venv\Scripts\python.exe")) { return $false }
  & .venv\Scripts\python.exe -c "import typer, fastapi, jieba, playwright, httpx" 2>$null
  return ($LASTEXITCODE -eq 0)
}
if (-not (Test-VenvHealthy)) {
  if (-not (Test-Path ".venv\Scripts\python.exe")) {
    if (Get-Command uv -ErrorAction SilentlyContinue) {
      uv venv --seed .venv
      "setuptools<70`nwheel<0.46" | Out-File -Encoding ascii .build-constraints.txt
      $env:PIP_CONSTRAINT = "$MC\.build-constraints.txt"
      .venv\Scripts\pip.exe install -r requirements.txt
    }
    else {
      python -m venv .venv
      .venv\Scripts\pip.exe install -U "pip<24.1" "setuptools<70" "wheel<0.46"
      .venv\Scripts\pip.exe install -r requirements.txt
    }
  }
  else {
    "setuptools<70`nwheel<0.46" | Out-File -Encoding ascii .build-constraints.txt
    $env:PIP_CONSTRAINT = "$MC\.build-constraints.txt"
    .venv\Scripts\pip.exe install -r requirements.txt
  }
  Say "  venv 建好 / 已修复"
}
else { Say "  已存在且健康，跳过" }

# ---- 4. 浏览器内核 ----
Say "步骤 4/5：Playwright Chromium"
.venv\Scripts\playwright.exe install chromium
Say "  Chromium 就位"

# ---- 5. 自检 ----
Say "步骤 5/5：自检"
Set-Location $Root
.\bin\tvop.exe doctor
.\bin\tvop.exe okf validate | Select-Object -Last 2
Say "引导完成。生成一条试试：.\bin\tvop.exe search 分区 虚标"
