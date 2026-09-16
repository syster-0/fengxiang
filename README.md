# 穹言 · 电视舆情运营官（tv-opinion-atelier）

把贴吧、B站、抖音的高浓度评论采进来，经仓颉 RIA-TV++ 蒸馏成**带权重与人味分**的
OKF 知识库，再从库里长出像真人写的电视行业评论。

一句话：**不是写文案的，是经营一个会生长的评论语料库的。**

## 类型

Agent 型（单个 AI 专家）· 三模块体系

| 模块 | 职责 | 实体 |
|---|---|---|
| M1 架构层 | 运行时、CLI、HTTP、MCP | `cmd/tvop` + `internal/*`（Go） |
| M2 蒸馏层 | 去 AI 味打分 + MediaCrawler 适配 | `internal/taste` · `internal/crawl` |
| M3 知识库 | OKF Wiki（概念 / 权重 / 门禁） | `wiki/` |

## 功能

### 1. 按声部定向采集
不按关键词泛采，按**语域形态**采：贴吧捞长贴与追楼，抖音捞回复区短句，
B站捞测评与拆解。单次 ≤ 200 条，来源先分级再入库。

### 2. 仓颉 RIA-TV++ 蒸馏
Adler 整体理解 → 五提取器 → 三重验证 → 晋级门 → RIA++ 能力卡
→ Zettelkasten 链接 → 压力测试 → 确定性编译入库。
**过不了三重验证的不进库**，登记为知识缺口。

### 3. 人味准入（不是加分项）
`humanness` 进排序分的**乘数位**：
`rank = weight × (0.5 + 0.5·humanness) × (1 + 0.4·resonance) × (1 + 0.2·tension) × penalty`
—— AI 味重的内容权重再高也被乘下去。11 条人味算子 + 15 项非判别特征 + 7 项增强项。

### 4. 五因子权重治理
`weight = α·use + β·recency + γ·link + δ·explicit + ε·quality`，
配 audit / promote / decay / recall 四类治理 job 与缺口检测，形成「采 → 归 → 蒸 → 校 → 回」闭环。

### 5. 四个核心声部

| 声部 | 平台 | 目标张力 | 目标长度 | 立身之本 |
|---|---|---|---|---|
| tieba-long | 贴吧 | 0.55 | 120 字 | 多余的细节与具体经历 |
| hot-consensus | 全平台 | 0.30 | 40 字 | 可执行动作 + 时间承诺 |
| bili-geek | B站 | 0.45 | 90 字 | 参数精度 + **必带短板** |
| douyin-conflict | 抖音 | 0.85 | 22 字 | 判决前置、零缓冲 |

### 6. 合规红线（4 条，生成前预检）
绝对化用语 / 无据贬损竞品 / 编造统计 / 价格与国补承诺。
扫描器对**引注与元讨论**免疫 —— 引用他人违规词不等于自己主张。

### 7. 主攻领域知识（随包分发）
TCL SQD-Mini LED 全系 7 系列参数梯度、三大核心技术、与 RGB-Mini LED 的路线之争；
2026-09-09~09-16 全网舆情窗口的 6 大叙事、4 个处置优先级；
安装/贴墙落差的真实事实内核（默认挂架不支持后插拔线）。

## 使用示例

- 「以『电视 选型』为关键词，扫一遍贴吧、B站和抖音的评论，蒸馏入库，然后告诉我这批语料的人味分和可用的声部。」
- 「为『65 寸 Mini LED 和 OLED 怎么选』写 20 条像真人的评论区回复，分声部输出，每条注明命中了哪条人味规则。」
- 「跑一轮权重治理：审计全库、晋升高权重概念、归档零使用的，并报告人味分分布的变化。」
- 「T7M Pro 装完贴不上墙，用户在小红书骂了 —— 帮我看这属于什么问题，该怎么回。」

## 库内现状（冷启动基线 · 2026-09-16）

- 概念 **48** · verified **33** · 可用于生成 **33** · 均权重 **0.659** · 均人味 **0.847** · 均风险 **0.000**
- 校验：`tvop okf validate` 全绿（0 错误 / 0 告警）
- 概念分布：Voice Persona 4 · Comment Pattern 5 · Tension Axis 6 · Model Spec 6 ·
  Market Claim 4 · Risk Rule 4 · Signal 4 · Audience Segment 4 · Source 3 ·
  Knowledge Gap 3 · Glossary 2 · Playbook 2 · Decision 1
- 实弹验证：四个核心声部各生成一条评论，人味分 0.919–1.000，合规风险 0.00

## 前置条件

| 依赖 | 说明 |
|---|---|
| Go 1.24+ | 仅 `go run` 回退时需要；`bin/tvop` 已编译 |
| MediaCrawler | **已内嵌**于 `third_party/MediaCrawler`（浅克隆，NON-COMMERCIAL LEARNING LICENSE 1.1）；外部副本可用 `MEDIACRAWLER_ROOT` 指定 |
| Python 环境 | `third_party/MediaCrawler/.venv`（Python 3.11，依赖已装齐）；jieba 需 `setuptools<70` 构建约束 |
| 浏览器内核 | `~/.cache/ms-playwright`（Chromium 153 + headless shell，已装） |
| 各平台凭证 | 由 MediaCrawler 侧要求（qrcode 扫码 / cookie），首次采集需人工登录 |

采集链路实测（2026-09-16）：plan 生成 ✓ · 付费门禁拦截 ✓ · 200 条上限 ✓ ·
jsonl 归一化（平台自动推断）✓ · 声部选样 ✓ · 对立轴配对 ✓ · feed 投喂 ✓ ·
上游 CLI 启动 ✓ · Playwright 无头启动 ✓。**真实抓取停在登录边界**——扫码/cookie 需人工完成。

自检：`bin/tvop doctor`

## 快速开始

```bash
cd <包根>
./bin/tvop doctor                 # 环境自检
./bin/tvop govern report          # 库状态
./bin/tvop search 分区 虚标        # 召回
./bin/tvop taste rules            # 人味规则表
./bin/tvop serve --addr :8080     # HTTP
./bin/tvop mcp                    # MCP（stdio，8 工具）
```

## 目录结构

```
├── agents/            专家人设（穹言）
├── skills/            tvop-ops / tvop-crawl / tvop-distill / tvop-okf / tvop-taste
├── bin/tvop           M1 架构层产物（Go CLI）
├── cmd/ internal/     Go 源码（可重新编译）
├── wiki/              M3 知识库
│   ├── concepts/      概念（路径即 ID，16 类封闭词表）
│   ├── source-map/    来源双向索引
│   ├── tests/         压力测试集（triggers / decoys / negatives）
│   ├── index.md       编译生成的索引（勿手改）
│   └── log.md         操作日志（append-only）
├── sources/tcl/       随包分发的来源语料
├── third_party/       内嵌 MediaCrawler（浅克隆 + .venv 运行环境）
└── avatars/           头像（512×512 PNG）
```

## 头像

`avatars/expert.png`（512×512，57KB，程序化生成）。
如需替换：PNG / 512×512 / ≤ 500KB。

## 安装与注册

```bash
python3 scripts/register_expert.py <expert-dir>
```

## 打包分享

```bash
zip -r tv-opinion-atelier.zip tv-opinion-atelier/ -x '*/\.git/*'
```

## git 分发与接收方还原

本仓库（包根即 git 仓库根）携带**全部可带走数据**：Go 源码、四平台预编译二进制、
wiki 知识库（48 概念 + source-map + tests + log）、TCL 语料、Agent/Skill、头像、
内嵌 MediaCrawler 源码快照（见 `third_party/MediaCrawler/VENDORED.md`）。

**分发通道**（当前）：

| 项 | 值 |
|---|---|
| 仓库 | `https://github.com/syster-0/fengxiang.git` |
| 默认分支 | `main`（本地开发分支 `master`，推送映射 `master:main`） |
| 性质 | 临时公共仓，用于一次性分发；**分发完成后可能删仓**。删仓后改用离线包 `tv-opinion-atelier.zip`，或另建新仓并 `git remote set-url origin <新地址>`（`tvop repo` 会自动跟随新远端） |
| 完整分发记录 | 见包内 `DISPATCH.md` |

**发送方**：

```bash
# 方式一：一键建仓推送（令牌只走环境变量，推完自动从 remote URL 抹除）
GH_USER=<用户名> GH_TOKEN=<token> bash scripts/publish.sh

# 方式二：已有远端，直接推
git add -A && git commit -m "..." 
git push origin master:main            # 本地 master → 远端 main
```

推送凭据要求：fine-grained PAT 需 **Contents: Read and write**；或走 GitHub 设备流
（scope=repo，无需令牌）。注意 WorkBuddy 的 GitHub 连接器是 Copilot 只读授权，
**不能**用于推送。

**自感知更新（专家自己会意识到需要拉取）**：

包内置 `tvop repo` 子命令，专家会话开始跑 `tvop doctor` 即可见分发远端；发送方
推送增量后，接收方侧：

```bash
tvop repo status   # 联网比对：报告落后/领先几个提交，落后时提示 pull
tvop repo pull     # 安全拉取：工作区脏则拒绝；仅 fast-forward 不产生合并
                   # 拉取后提示 tvop okf reindex && okf validate && govern recompute
```

知识库增量（wiki/、sources/）随 git 一起走 —— 发送方蒸馏归档后 `git push`，
接收方专家运行 `tvop repo status` 就能发现并拉取，实现「持续生长」的跨机同步。

**接收方（任意系统）**：

```bash
# Linux / macOS
git clone https://github.com/syster-0/fengxiang.git tv-opinion-atelier && cd tv-opinion-atelier
bash scripts/bootstrap.sh --cn       # --cn 走国内镜像（gh-proxy/goproxy/清华PyPI/npmmirror）

# Windows (PowerShell)
git clone https://github.com/syster-0/fengxiang.git tv-opinion-atelier; cd tv-opinion-atelier
powershell -ExecutionPolicy Bypass -File scripts\bootstrap.ps1 -CN
```

（仓库若已被删除，改为解压 `tv-opinion-atelier.zip` 后执行同样的 bootstrap 命令。）

引导脚本幂等，五步自动完成：① 按平台挑选 `bin/tvop-<os>-<arch>` 预编译二进制
（无匹配时回退 `go build`）→ ② 缺失时克隆 MediaCrawler → ③ 建 Python venv
（含 jieba 所需 `setuptools<70` 构建约束）→ ④ 安装 Playwright Chromium →
⑤ `tvop doctor` + `okf validate` 自检。

**跨平台适配说明**：
- 二进制矩阵：`linux/amd64 · darwin/amd64 · darwin/arm64(Apple Silicon) · windows/amd64`，
  锚点基于 `os.Executable()`，从任意工作目录调用都能定位 wiki。
- 代码层路径统一用 `filepath.Join`，概念 ID 逻辑分隔符恒为 `/`，Windows 反斜杠已适配。
- 账号登录态（`accounts/`、`browser_data/`）、venv、采集数据属机器本地运行态，不入 git；
  接收方引导后重新生成。中国大陆网络用 `--cn`/`-CN` 开关走镜像。

## 边界与免责

- **内嵌 MediaCrawler 采用 NON-COMMERCIAL LEARNING LICENSE 1.1**：仅限非商业的学习与研究用途；
  商业舆情场景必须停用该引擎，改接有授权的数据源或平台官方 API。
- 采集永远：单次 ≤200 条、控频、不并发轰炸、遵守平台条款与 robots.txt。
- 采集的是公开评论，用途限于语域分析与内部策略；对外引用需脱敏，不点名真人账号。
- 生成的评论用于语域演练与内部审阅，不得批量投放平台冒充真实用户。
- 参数类事实以 `sources/tcl/sqd-product-corpus.md` 为准，全部来自**厂商宣传口径**，
  实验室数据与实际使用可能存在偏差；标「待确认」项不可作为断言引用。
- 本专家不代替企业做最终公关决策；涉及监管口径与安装投诉的事项，
  须经内部核实后方可对外定性。
