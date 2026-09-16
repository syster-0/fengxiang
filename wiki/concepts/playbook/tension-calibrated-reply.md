---
type: Playbook
title: 流程：张力校准与声部匹配回复
voice: douyin-conflict
atelier:
  weight: 0.7
  tier: long
  verdict: verified
  use_count: 1
  source_ref: sources/tcl/platform-live-corpus.md
  humanness: 0.795
  resonance: 0.68
  tension: 0.62
  risk: 0
  updated: 2026-09-16
  links:
    - voice/tieba-long
    - voice/hot-consensus
    - voice/bili-geek
    - voice/douyin-conflict
    - tension/thin-body-vs-real-flush
---


## R 触发场景

面对一条具体评论或一个具体场景，需要**决定用哪个声部、用多高的张力**来回复。

常见触发：
- 评论区出现高对立短评，需要接话
- 用户提出具体咨询（尺寸、安装、价格），需要专业答复
- 一条抱怨需要「同情但不越界」的回应
- 同一条内容要投放到多个平台，需要做声部适配

## I 框架

**两维定声部：话题张力 × 受众意图**

| | 受众想「听劝」 | 受众想「吵架」 | 受众想「验证」 |
|---|---|---|---|
| **低张力话题**（颜色、场景、护眼） | xhs-scene 0.25 | — | zhihu-rational 0.20 |
| **中张力话题**（尺寸、参数、价格） | hot-consensus 0.30 | wb-spread 0.60 | bili-geek 0.45 |
| **高张力话题**（安装、品牌、行业） | tieba-long 0.55 | douyin-conflict 0.85 | bili-geek 0.45 |

**首要判据不是话题，是受众意图。** 同一条「安装抱怨」：
- 给装修中的家庭 → tieba-long 0.55（给具体经验，听劝型）
- 给评论区的喷子 → douyin-conflict 0.85（短、狠、但不骂人）
- 给想核实的数码党 → bili-geek 0.45（给参数与限定条件）

## A 步骤

**Step 1 · 定张力**
先判这条内容的合理张力区间，不要直接写。参考三个锚：
- **0.20–0.30**（低）：解释型，重在讲清；不用反问，不用短句节奏。
- **0.45–0.60**（中）：有立场，但带限定条件；可用一次让步。
- **0.80–0.90**（高）：判决前置，零缓冲；**但禁止指向人和品牌**。

**Step 2 · 选声部**
按上表定位。**若受众意图不明，选中间值**（bili-geek 0.45 是最安全的默认位）。

**Step 3 · 取概念**
`tvop search <关键词>` 召回 —— 召回排序分 = 命中 × weight × tier × 人味。
只使用 `usable = true` 的概念（verified + 人味 ≥ 0.60 + 风险 ≤ 0.40）。

**Step 4 · 生成**
按所选声部的「A1 开场与收尾」+ 对应 pattern 的「E 可直接用的句式」组织。
**每个声部都有硬性包含项**（如 bili-geek 必须包含至少 1 个保留意见）。

**Step 5 · 双门自检**
- 人味：`tvop taste score <文件>`，< 0.60 则按算子回炉。
- 合规：`spec.RiskScan` 命中 4 条红线任一 → 改写。
- 声部越界：对照该声部「B 越界特征」逐条扫。

**Step 6 · 记账与回流**
`tvop okf get <id> --account` 记账；发布后回填互动数据（resonance）。

## B 失效边界

1. **张力越高越好** —— 错。高张力只在高对立场景有效；对咨询类用户用 0.85 会被读成攻击。
2. **跨声部混搭** —— 抖音体起手 + 贴吧体长论证 = 两头不靠。**一个回复只用一个声部。**
3. **高张力当骂人许可** —— 高对立 ≠ 骂人。对立的是观点，不是人。
   见 `voice/douyin-conflict` 越界特征第 1 条（硬拦）。
4. **忽略硬性包含项** —— bili-geek 不写短板、tieba-long 不给具体细节、
   douyin-conflict 没有生活化指涉 → 声部立不住。
5. **使用不合规概念** —— 风险分 > 0.40 的概念不可用于生成（合规门）。
6. **「同情但不越界」失衡** —— 回应安装抱怨时，过度共情会滑向
   「乱收费」定性（`risk/absolute-superlative`）；过于冷淡又会激化。
   正确落点：**承认落差 + 给可执行的三问清单**（见 `spec/wall-mount-tiers`）。
7. **对同一条内容多平台直接复制** —— 各平台声部生态不同，须逐平台做声部适配。
8. **跳过自检直接发布** —— 两道门是硬门，跳过等于把风险直接交付出去。
