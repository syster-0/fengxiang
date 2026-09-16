---type: Playbook
title: 流程：评论蒸馏与人味权重闭环
voice: tieba-long
atelier:
  weight: 0.7
  tier: long
  verdict: verified
  use_count: 1
  source_ref: sources/tcl/platform-live-corpus.md
  humanness: 0.877
  resonance: 0.6
  tension: 0.45
  risk: 0
  updated: 2026-09-16
  links:
    - voice/tieba-long
    - voice/bili-geek
    - pattern/geek-admit-then-refute
    - signal/spec-backfire
    - decision/not-rebut-industry-decline
---

## R 触发场景

需要**持续生长**而不是一次性产出评论时。
具体触发：新一批平台语料到位、某条评论发布后互动数据回流、
某个概念被反复追问（说明缺口）、某条内容被判定 AI 味过重需回炉。

## I 框架

**五步闭环：采 → 归 → 蒸 → 校 → 回**

```
采  crawl     按声部定向采语料（不是按关键词，是按语域形态）
  ↓
归  bucket    按「声部可提取性」分区（A 参数拆解 / B 换算之争 /
              C 对立冲突 / D 共识金句 / E 体验落差 / F 反例短板）
  ↓
蒸  distill   RIA-TV++：Adler 整体理解 → 五提取器 → 三重验证
              → 晋级门 → RIA++ 能力卡 → Zettelkasten 链接
              → 压力测试 → 确定性编译入库
  ↓
校  validate  tvop okf validate 全绿 + 人味门 + 合规门
  ↓
回  feed      发布后互动数据回流 → 重算五因子权重 → 找缺口 → 回到「采」
```

## A 步骤

**1. 采（定向，不泛采）**

| 声部 | 采集对象 | 采集目标 |
|---|---|---|
| tieba-long | 贴吧长贴、追楼 | 语域与黑话、论证推进方式、自我引用 |
| hot-consensus | 高赞短评 | 共识框架、金句骨架、为什么能高赞 |
| bili-geek | 数码测评、拆解 | 论据结构、参数精度、梗的用法与尺度 |
| douyin-conflict | 抖音回复区 | 张力结构、冲突起手、短句节奏 |

硬约束：单次扫描 ≤ 200 条（`spec.MaxCrawlPerRun`）；不同声部的语料不可混采。

**2. 归（按形态分区，不按来源分区）**

同一来源可能同时贡献多个声部的料 —— 例如一份电商晒单同时提供
「SQD 偏红」的负样本（→ bili-geek）与「自购挂架避坑」的场景样本（→ xhs-scene）。
分区的意义是让声部蒸馏有**明确的取材入口**。

**3. 蒸（RIA-TV++ 七阶段）**

- **Adler 整体理解**：先写 `source/` 概念（结构 → 解释 → 批判 → 应用四步），不清洗矛盾。
- **五提取器**：从语料抽「论点 / 论据 / 限定条件 / 让步 / 收口」五种成分。
- **三重验证**：V1 语料是否支持 → V2 是否与已有概念冲突 → V3 是否可被追问而不崩。
- **晋级门**：过不了三重验证的**不进库**（未入库不使用）。
- **RIA++ 能力卡**：R 原文样本 / I 骨架抽象 / A 触发与迁移 / E 可直接用句式 / B 禁用边界。
- **Zettelkasten 链接**：每个概念至少双向出链 2 条（写进 `atelier.links`）。
- **压力测试**：对每个概念写 triggers / decoys / negatives 三组测试集。
- **编译入库**：`tvop okf new` → `tvop okf sourcemap` → `tvop okf test`。

**4. 校（两道硬门 + 结构校验）**

- **人味门 ≥ 0.60**：`tvop taste score` 打分。
  超标的按 11 条算子回炉改写（翻案腔 / 顿号罗列 / 相邻句同款 / 破折号 /
  冒号 / 序数词标题 / 拟人化喻体 / 概括盖数据 / 禁用起手式 / 翻译腔 / 段首零回指）。
- **合规门 ≤ 0.40**：`spec.RiskScan` 扫 4 条红线。
- **结构校验**：`tvop okf validate` 全绿（frontmatter / Source Map 双向闭合 /
  verified 须有测试集 / index 一致 / log 无断档）。

**5. 回（数字权重闭环 —— 这一层是本体系「持续生长」的机制）**

- 写入即记账：每次读取 `use_count++`（`tvop okf get <id> --account`）。
- 五因子重算：`tvop govern recompute`
  （`weight = α·use + β·recency + γ·link + δ·explicit + ε·quality`）。
- 人味维度并入排序分：
  `rank = weight × (0.5 + 0.5·humanness) × (1 + 0.4·resonance) × (1 + 0.2·tension) × penalty`。
- 治理四类 job：`tvop govern audit | promote | decay | recall`。
- 缺口检测：`tvop gaps <要素>` → 生成投喂清单 → 回到「采」。
- 发布后真实互动（点赞/回复/转载比）回填 `resonance`，构成**外部真实信号**，
  避免只在库内自我循环。

**关键设计**：`humanness` 与 `resonance` **都是乘数项** ——
AI 味重的内容即使权重高，也会被 0.5+0.5·humanness 项压下去。
这让「人味」不是加分项而是**准入因子**。

## B 失效边界

1. **语料污染**：把厂商稿、AI 批量生产内容（`similarity_num` 高的）当语料采进来 →
   整条链路被污染。**采阶段必须做来源分级**（见 `platform-live-corpus.md` G 区）。
2. **清洗掉负样本**：把「SQD 偏红」这类负面反馈过滤掉 → 生成内容失去可信度，
   被数码党一眼识破。**负样本是资产，不是噪音。**
3. **跨声部混采**：把抖音短句喂给贴吧声部 → 声部人格崩解。
4. **跳过压力测试**：无 decoys / negatives 的概念不可用于生成（体系硬门）。
5. **只在库内循环**：不回填真实互动数据 → 权重自我强化，逐步失真。
6. **把 `humanness` 当装饰**：人味分不达标仍强行 verified → `validate` 会拦，但
   更常见的是**为了过门而调数字**，这等于自毁体系。正确做法是回炉改写。
7. **一次性冲量**：一次性灌几百条概念而不建链接、不做治理 →
   库很大但召回差。**宁少勿滥**（单次 ≤ 200 条）。
8. **未入库使用**：直接从原始语料生成评论而不落库 → 无溯源、无法治理、无法生长。
