# 分发记录（DISPATCH）

> 本文件随包走：记录「这个包是怎么分发出去的、对方怎么还原、以后怎么更新」。
> 接收方 clone 后可以直接读这一篇上手。

## 1. 分发通道

| 项 | 值 |
|---|---|
| 仓库地址 | `https://github.com/syster-0/fengxiang.git` |
| 默认分支 | `main` |
| 发起日期 | 2026-09-16 |
| 分发时 commit | `07ec361`（9 个提交，503 个文件） |
| 本地对应分支 | `master`（已设上游 `origin/main`） |
| 仓库性质 | 临时公共仓，一次性分发用；**分发完成后可能删仓** |

## 2. 接收方三步还原

```bash
# ① 拉包
git clone https://github.com/syster-0/fengxiang.git tv-opinion-atelier
cd tv-opinion-atelier

# ② 引导（幂等，五步：挑二进制 → 补 MediaCrawler → 建 venv → 装 Chromium → 自检）
bash scripts/bootstrap.sh --cn            # Linux / macOS，--cn 走国内镜像
powershell -ExecutionPolicy Bypass -File scripts\bootstrap.ps1 -CN   # Windows

# ③ 确认环境
./bin/tvop doctor          # 全绿才算就绪
./bin/tvop okf validate    # 期望：概念 48 · 错误 0 · 告警 0
```

仓库若已被删除：解压 `tv-opinion-atelier.zip` 后执行同样的 bootstrap 命令。

## 3. 随包带走的全部资产

- `cmd/` + `internal/`：Go 源码（`tvop` CLI，含 `repo` 分发自感知子命令）
- `bin/`：四平台预编译二进制 —— `tvop`(linux/amd64)、`tvop-linux-amd64`、
  `tvop-darwin-amd64`、`tvop-darwin-arm64`、`tvop-windows-amd64.exe`
- `wiki/`：OKF 知识库（48 概念 + source-map + tests + log，含四声部定向训练语料）
- `sources/tcl/`：TCL 来源语料；`agents/` + `skills/`：专家与技能定义；`avatars/`：头像
- `third_party/MediaCrawler/`：采集引擎源码快照（锚点 commit 见 `VENDORED.md`）

**不入包的机器本地运行态**（接收方引导后自行生成）：`accounts/`、`browser_data/`、
Python venv、采集原始数据。

## 4. 持续生长：接收方如何拿到增量

包内置 `tvop repo`，专家会话开头就跑 `tvop doctor`，doctor 会报出分发远端；
之后专家按「**先同步、后干活**」自动执行：

```bash
tvop repo status   # 联网比对，报告落后/领先几个提交
tvop repo pull     # 安全拉取：工作区脏则拒绝；仅 fast-forward，不产生合并
                   # 拉取完成后提示：tvop okf reindex && okf validate && govern recompute
```

发送方蒸馏归档 → `git push` → 接收方下次跑专家时自动发现并拉取。
`wiki/`、`sources/` 的增量随 git 一起走，这就是「跨机持续生长」的实现方式。

## 5. 发送方推送（再次分发时照做）

```bash
# 方式一：一键建仓推送（令牌只走环境变量，推完自动从 remote URL 抹除，幂等）
GH_USER=<用户名> GH_TOKEN=<token> bash scripts/publish.sh

# 方式二：远端已存在
git add -A && git commit -m "..." && git push origin master:main
```

**凭据要求**（实测结论，避免踩坑）：

- fine-grained PAT：必须勾 **Contents: Read and write**，且仓库在授权范围内。
  注意 `GET /repos/...` 返回的 `permissions.push: true` 是账号角色位，
  **不代表**令牌有写权限——用写探针实测才准。
- 或者走 GitHub 设备流（scope=`repo`，对方点一下授权即可，无需令牌）。
- WorkBuddy 的 GitHub 连接器走 Copilot MCP（`ghu_` 型令牌，scope 为空），
  **只读、不能推送**，`create_or_update_file` / `push_files` 一律 403。

## 6. 换仓 / 删仓怎么办

```bash
git remote set-url origin <新仓库地址>
git push --force origin master:main
```

`tvop doctor` 与 `tvop repo status` 都从 `git remote get-url origin` 实时读取，
换远端后无需改任何代码或配置，自感知自动跟随新地址。
