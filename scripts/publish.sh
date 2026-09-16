#!/usr/bin/env bash
# tv-opinion-atelier 发布脚本：建仓 + 推送 + 生成分享连接
#
# 用法（令牌只走环境变量，不落盘、不进 shell 历史）：
#   export GH_USER=<github用户名>
#   export GH_TOKEN=<personal access token，需 repo 权限>
#   bash scripts/publish.sh [仓库名=tv-opinion-atelier] [private|public=private]
#
# 幂等：仓库已存在则直接复用并推送。

set -euo pipefail

GH_USER="${GH_USER:?请先 export GH_USER=<github用户名>}"
GH_TOKEN="${GH_TOKEN:?请先 export GH_TOKEN=<personal access token>}"
REPO="${1:-tv-opinion-atelier}"
VIS="${2:-private}"

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

API="https://api.github.com"

# ---- 1. 建仓（已存在则跳过） ----
echo "==> 检查/创建仓库 $GH_USER/$REPO ($VIS)"
CODE=$(curl -s -o /tmp/.gh-repo-resp.json -w "%{http_code}" \
  -X POST "$API/user/repos" \
  -H "Authorization: token $GH_TOKEN" \
  -H "Accept: application/vnd.github+json" \
  -d "{\"name\":\"$REPO\",\"private\":$( [ "$VIS" = private ] && echo true || echo false ),\
\"description\":\"穹言·电视舆情运营官 —— OKF 知识库驱动的电视行业舆情运营专家（含四平台二进制与自更新通道）\"}")
case "$CODE" in
  201) echo "    仓库已创建" ;;
  422) echo "    仓库已存在，复用" ;;
  401) echo "x 令牌无效或过期"; exit 1 ;;
  *)   echo "x 建仓失败 HTTP $CODE"; cat /tmp/.gh-repo-resp.json; exit 1 ;;
esac
rm -f /tmp/.gh-repo-resp.json

# ---- 2. 配置远端并推送（令牌仅出现在本次 URL 中） ----
REMOTE_URL="https://$GH_USER:$GH_TOKEN@github.com/$GH_USER/$REPO.git"
if git remote get-url origin >/dev/null 2>&1; then
  git remote set-url origin "$REMOTE_URL"
else
  git remote add origin "$REMOTE_URL"
fi
BRANCH="$(git rev-parse --abbrev-ref HEAD)"
git push -u origin "$BRANCH"
# 推完立刻把令牌从 remote URL 里抹掉，只留干净地址
git remote set-url origin "https://github.com/$GH_USER/$REPO.git"

# ---- 3. 生成分享连接 ----
echo
echo "==> 发布完成。分享连接（发给接收方）："
echo
echo "  git clone https://github.com/$GH_USER/$REPO.git tv-opinion-atelier"
echo "  cd tv-opinion-atelier && bash scripts/bootstrap.sh --cn"
echo
echo "  （私有仓库：接收方需先在 GitHub 被邀请为协作者，或改用 deploy key / fine-grained token）"
echo
echo "==> 本地验证：tvop repo status 应显示与远端同步"
git remote get-url origin
