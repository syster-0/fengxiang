---type: Source
title: TCL 全网舆情周报 2026-09-09~09-16（来源总览）
atelier:
  weight: 0.627
  tier: long
  verdict: unverified
  use_count: 1
  source_ref: sources/tcl/public-opinion-weekly-2026-09-16.md
  humanness: 0.816
  resonance: 0.45
  tension: 0.6
  risk: 0
  updated: 2026-09-16
  links:
    - signal/install-complaint-cluster
    - signal/regulator-record
    - tension/industry-decline-vs-brand-growth
---

## 来源清单

| 项 | 内容 |
|---|---|
| 文件 | `sources/tcl/public-opinion-weekly-2026-09-16.md`（23KB，自 249KB HTML 抽取） |
| 原始 HTML | `sources/tcl/public-opinion-weekly-2026-09-16.raw.html` |
| 监测窗口 | 2026-09-09 13:08 ~ 2026-09-16 13:10（7 天，168h） |
| 监测主体 | TCL / TCL电视 / TCL电子 / TCL华星 |
| 样本 | 检索池 1,827 条（sentiment=negative）；实读 400 条（降序 200 + 升序 200） |
| 危机等级 | **L3 仅监控**，无需启动响应 |
| 生成方 | MOSS 增长谋士 · 舆情洞察专家 |

## 结构 · 解释 · 批判 · 应用

**结构**：七节 —— 态势层（要不要紧张）→ 声量层（7 日走势与平台结构）→ 事件层（6 条核心叙事）
→ 行动层（优先级处置）→ SQD 产品清单 → 数据边界 → 证据来源。
组织逻辑是「先给结论，再给锚点」，每条叙事固定五段：判断 / 推断 / 事实 / 行动 / 确定度。

**解释**：这份周报最有价值的部分不是结论，而是**它显式登记了自己的不确定性**。
「数据边界」一节列了 8 条口径限制，其中三条对本知识库有直接约束：
情绪字段 200 条全部为「未知」（= 未标注，非「已确认负面」）；
media_level 全部「未定义」（无央级/省级媒体，是判 L3 的核心依据）；
样本中「开机率跌破40%」单篇 `similarity_num=188`（模板化批量生产，声量绝对值虚高）。

**批判**：
1. **噪声率 38%**（76/200 与 TCL 无关）—— 平台结构表里的占比是基于 124 条相关子集，不可外推到全量。
2. 报告自述「当前同名 V2 适配器无独立全文精读能力」，所有结论基于摘要 + 真实链接 —— 确定度已下调。
3. 「TCL 真实相关率 62%」与「TCL 相关情绪中性率 82.3%」两个指标的分母不同（200 vs 124），
   引用时极易混用致误。
4. 叙事 4 引用的「35% 用户吐槽」被报告自己标注为**无来源支撑、不采信** —— 这是 `risk/fabricated-stat` 的真实案例。

**应用**：作为**舆情信号基线**。7 日逐日声量序列（218 → 369 → 254）可作后续周报的同口径对照。
6 条叙事分别喂给 `tension/` 与 `signal/` 两个桶；叙事 3（T7M Pro 安装）是本库最优先蒸馏的对象。

## 蒸馏范围

- 已蒸馏：叙事 1 → `tension/industry-decline-vs-brand-growth`；叙事 2 → `claim/samsung-three-lies`；
  叙事 3 → `signal/install-complaint-cluster` + `tension/thin-body-vs-real-flush`；
  叙事 5 → `signal/regulator-record`；叙事 4 → `risk/fabricated-stat`（作反例）。
- 未蒸馏：7 日逐日明细、信源级别披露、MOSS 落盘文件路径清单。
- 不得蒸馏：被报告自身否定可信度的「35%」数字。
