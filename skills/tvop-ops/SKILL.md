---
name: tvop-ops
description: 电视舆情运营主入口。用户要求开始一轮采集、查看知识库状态、跑权重治理（审计/晋升/衰减/归档）、检查环境是否就绪、查看词表与规范，或问「现在库里有啥、能用几条」时使用。负责环境自检与收尾报告，并在开工前把四要素（平台/关键词/声部/时间窗）问清。
---

# tvop-ops · 运营主入口

## 何时使用

- 开工第一步的环境自检与库状态汇报。
- 需要跑治理：审计、晋升、衰减、归档、五因子重算。
- 需要查词表：概念类型、声部、人味算子、合规红线、平台映射。
- 用户问「库里现在怎么样」「有多少条能用」。

采集走 `$tvop-crawl`；蒸馏走 `$tvop-distill`；落库校验走 `$tvop-okf`；打分走 `$tvop-taste`。
本 skill 只管**入口、自检与治理闭环**。

## 启动检查（不许跳）

0. 确定 CLI 与包根：
   - 首选 `<包根>/bin/tvop`；
   - 不可执行时退回 `cd <包根> && go run ./cmd/tvop`，并在摘要里声明降级。
1. `tvop doctor` —— 检查 wiki 是否就绪、MediaCrawler 是否找到、词表是否完整。
   **doctor 不过，不进入采集。** 缺 wiki 就先 `tvop init`。
2. `tvop govern report` —— 报概念数 / 可用数 / 均权重 / 均人味 / 均风险 / tier 分布。
3. 与用户确认四要素：**平台、关键词、目标声部、时间窗**。
   缺一项就问；**不要静默按「全平台 + AI + 最近 7 天」处理**。
4. 报一句本轮的预期产出与边界（单次 ≤ 200 条；付费动作需确认）。

## 常用命令

```bash
tvop doctor                      # 环境自检
tvop init                        # 初始化 wiki 骨架（幂等）
tvop govern report               # 库状态总览
tvop govern audit                # 审计：结构 / 门禁 / 链接 / 缺口
tvop govern recompute --dry      # 五因子权重重算（先演练）
tvop govern recompute            # 实算并写回
tvop govern promote / decay / recall   # 晋升 / 衰减 / 归档
tvop gaps <要素...>              # 知识缺口检测
tvop spec                        # 词表总览
tvop spec types|voices|weights|rules|platforms
tvop weight explain <概念 ID>     # 单概念的权重因子拆解
tvop search <关键词...>           # 召回（命中 × weight × tier × 人味）
```

## 五因子权重

```
weight = α·use + β·recency + γ·link + δ·explicit + ε·quality
```

| 因子 | 来源 | 饱和度 |
|---|---|---|
| use | 读取记账次数（创建即 1） | 4 次满分 |
| recency | 距最近写入的天数 | 45 天半衰 |
| link | 出入链总数 | 6 条满分 |
| explicit | 人工确认 / 发布后真实互动回流 | — |
| quality | 蒸馏阶段压力测试得分 | — |

排序分（召回用）：
```
rank = weight × (0.5 + 0.5·humanness) × (1 + 0.4·resonance) × (1 + 0.2·tension) × penalty
```

**注意 `humanness` 是乘数。** 这是体系的关键设计：人味不达标的内容，
权重再高也被 0.5+0.5·humanness 这一项压下去。

> 实测基线（2026-09-16 冷启动）：48 概念 · 33 可用 · 均权重 0.659 · 均人味 0.807 · 均风险 0.000。

## 治理闭环（第五段「回」）

这是「持续生长」的实现层，不是可选项：

1. **记账**：`tvop okf get <id> --account` —— 每次真正用到某概念就记一次。
2. **重算**：`tvop govern recompute`。先 `--dry` 看一眼变化再落。
3. **治理**：
   - `promote` —— 高权重、高共鸣的概念升 tier，优先召回。
   - `decay` —— 长期未用按新近度衰减。
   - `recall` —— 零使用、低权重的归档，别让噪音占召回位。
4. **找缺口**：`tvop gaps <要素...>`。返回「某要素无可用概念支撑」时，
   生成投喂清单，回到采集段。
5. **外部回流**：发布后的点赞/回复/转载比回填 `resonance`。
   只在库内循环会让权重自我强化、逐步失真。

## 输出规范

- 先给结论（能不能开工 / 库健不健康），再列证据。
- 数字带日期，因为权重与人味分随时间变动。
- 治理前后给**对比**（`--dry` 演练结果 vs 实算结果），让用户看清变化量。
- 缺什么就明说缺什么，不美化分布。
