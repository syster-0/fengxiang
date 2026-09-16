---
name: tvop-crawl
description: 按声部定向采集 TV 行业选型评论。用户要求扫贴吧/B站/抖音/小红书/快手/微博/知乎的电视相关评论、批量拉取某个关键词下的高赞评论、把评论投喂进知识库、采集竞品对骂或安装吐槽时使用。负责 MediaCrawler CLI 适配、来源分级、按声部选样与人味初筛。
---

# tvop-crawl · 定向采集

## 何时使用

- 「扫一遍贴吧的电视选型讨论」「把抖音的评论区拉下来」。
- 需要新增语料投喂知识库。
- 需要针对某个具体争议（安装吐槽、分区之争）定点采集。

蒸馏归类走 `$tvop-distill`；落库校验走 `$tvop-okf`。本 skill 只管**采与初筛**。

## 采集原则（三条，违反即重采）

1. **按语域形态采，不按关键词泛采。**
   同样是「电视」，贴吧要捞长贴与追楼，抖音要捞回复区短句，B站要捞测评与拆解。
   泛采回来的是混合语料，蒸馏阶段会分不出声部。

2. **单次 ≤ 200 条。** 宁少勿滥。采多了既不礼貌，蒸馏质量也会崩。

3. **来源分级，先于入库。**
   以下三类不进库（进了会污染整条链路）：
   - 厂商稿 / 官方宣传文案（口径与官方一致，无独立语料价值）；
   - AI 批量生产内容（同模板相似版本数异常高的，如某篇 `similarity_num=188`）；
   - 与主体无关的噪声（样本实测噪声率可达 38%）。

## 平台与声部映射

| 平台 | 声部 | 采集重点 |
|---|---|---|
| tieba | tieba-long | 长贴、追楼、自我引用、具体经历 |
| bili | bili-geek | 测评、拆解、参数精度、补限定句 |
| dy | douyin-conflict | 回复区、冲突起手、短句节奏 |
| xhs | xhs-scene | 居家场景、体感描述、安装落差 |
| ks | （需显式指定 --voice） | 同 dy，但语域有差异 |
| wb | wb-spread | 话题标签、传播句式 |
| zhihu | zhihu-rational | 长论证、限定词、反例处理 |

## 常用命令

```bash
tvop crawl platforms                    # 列出平台与声部映射
tvop crawl plan --platform tieba --keywords "电视,选型" --max-notes 200
tvop crawl run   --platform tieba --keywords "电视,选型" --max-notes 200 --yes
tvop crawl ingest  <jsonl 或目录>        # 归一化 MediaCrawler 输出
tvop crawl select  --voice tieba-long <jsonl 或目录>   # 按声部选样
tvop crawl feed    --voice tieba-long --batch 20260916 # 投喂成 Transcript 概念
tvop crawl tension <jsonl 或目录>        # 从语料里提对立轴候选
```

MediaCrawler 定位：`MEDIACRAWLER_ROOT` 环境变量，或克隆到 `<包根>/third_party/MediaCrawler`（包内已内置一份浅克隆）。
`tvop doctor` 会报是否找到。上游依赖装不上时（jieba 在新版 setuptools 下构建失败），
用 `PIP_CONSTRAINT=<约束文件> .venv/bin/pip install -r requirements.txt`，
约束文件内容写 `setuptools<70`，并设 `TMPDIR` 到大容量目录（`/tmp` 常是 10MB tmpfs）。

## 许可与合规边界（嵌入 MediaCrawler 的前置约束）

内嵌的 MediaCrawler 源码采用 **NON-COMMERCIAL LEARNING LICENSE 1.1**，这划死了采集引擎的边界：

1. **仅限非商业的学习与研究用途。** 本专家用它做语域分析和蒸馏研究是合规的；
   若用于商业舆情项目，必须改接有授权的数据源或平台官方 API，不得再用该引擎抓取。
2. **禁止大规模爬取与干扰平台运营。** 单次 ≤200 条、控制频率，既是礼貌也是许可条款。
3. **遵守目标平台使用条款与 robots.txt。**
4. 上游声明原文见 `third_party/MediaCrawler/LICENSE` 与各源码文件头部，不得移除。

## 付费与礼貌边界

- **付费动作永不自动重试。** 需要确认时 CLI 会拦下并要求显式确认，不要绕过去。
- 遵守平台频率限制；单次不超过 200 条；不并发轰炸。
- 采集的是**公开评论**，用途限于语域分析与内部策略；对外引用要脱敏，不点名真人账号。

## 采集后立刻做三件事

1. **来源分级**：按上面三类剔掉不可用样本，并把保留理由记进 Source Map 的 note。
2. **人味初筛**：`tvop taste score --file <样本>`。
   AI 味极重的样本（比如明显的营销矩阵稿）单独标记，不进声部语料。
3. **按声部选样**：`tvop crawl select --voice <声部>`。
   选样标准不是「点赞最高」，而是**语域纯度**：这条像不像真人写的。

## 输出规范

- 报清楚：平台、关键词、时间窗、**实际入库条数**（不是请求条数）。
- 报清楚剔除原因与条数（噪声 / 厂商稿 / AI 批产各多少）。
- 给出下一步建议：这批语料够蒸哪些概念，还缺什么声部。
- 若某声部样本不足（比如抖音短评采不到 20 条），**直说不足**，不要用别的声部凑数。
