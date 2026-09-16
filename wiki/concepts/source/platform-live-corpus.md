---type: Source
title: 平台实测语料库（来源总览）
atelier:
  weight: 0.627
  tier: long
  verdict: unverified
  use_count: 1
  source_ref: sources/tcl/platform-live-corpus.md
  humanness: 0.736
  resonance: 0.5
  tension: 0.65
  risk: 0
  updated: 2026-09-16
  links:
    - voice/bili-geek
    - voice/douyin-conflict
    - tension/zone-count-vs-real-control
    - tension/thin-body-vs-real-flush
    - pattern/geek-added-caveat
---

## 来源清单

| 项 | 内容 |
|---|---|
| 文件 | `sources/tcl/platform-live-corpus.md` |
| 采集日期 | 2026-09-16 |
| 采集方式 | 公开网页检索（媒体评测 / 访谈转发 / 投诉平台 / 电商社区 / 百科） |
| 分区方式 | 按「声部可提取性」分区（A 参数拆解 / B 换算之争 / C 对立冲突 / D 共识金句 / E 体验落差 / F 反例短板 / G 溯源可信度） |
| 来源数 | 9 个独立来源，覆盖 3 个可信度层级 |

## 结构 · 解释 · 批判 · 应用

**结构**：本文件与另两份来源的**组织逻辑完全不同** —— 前两份按「事实」组织，本份按「可提取形态」组织。
同一来源（如 what值得买晒单）会同时贡献 D4（偏红负共识）与 E3（挂架 DIY 反解）两条料。

**解释**：分区的意义在于**让声部蒸馏有明确的取材入口**。
`voice/bili-geek` 只读 A 区与 F 区；`voice/douyin-conflict` 只读 C 区；
`voice/hot-consensus` 只读 D 区。这避免了「拿到一堆文本却不知道提取什么」的常见失败。

**批判**：
1. 本文件**不承担事实核验职责** —— 分区说明里已明确「参数类事实以 `sqd-product-corpus.md` 为准」。
2. A3（海信分区拆解）与 B1（TCL 换算话术）均为**单方口径**，互为镜像却不互为验证。
   两者都指向同一件事：**分区的「物理 vs 逻辑」问题在行业内普遍存在，且没有任何厂商公开物理灯珠数**。
3. C2 段落里出现的 VIDDA X85 挂架问题**不是 TCL 案例**，但它是 E1 投诉的结构同构旁证 ——
   引用时必须标明品牌，不得张冠李戴。
4. 所有「媒体/自媒体稿件」均有带货倾向，参数口径与厂商一致 —— 不可作为独立验证。

**应用**：本文件是 `voice/` 与 `pattern/` 两个桶的**唯一语料入口**。
其中 A2（海外评测的括号补注习惯）与 A1 结尾的「补限定」动作，
是去 AI 味算子中「概括盖数据」与「相邻句同款」两条规则的真实正面样本 ——
即**高可信度中文评论本身就带限定条件，不需要额外添加礼貌缓冲**。

## 蒸馏范围

- 已蒸馏：A1 → `pattern/geek-admit-then-refute` + `pattern/geek-added-caveat`；
  A3/B1 → `tension/zone-count-vs-real-control` + `gap/rgb-vs-sqd-third-party-bench`；
  C1/C2/C3 → `voice/douyin-conflict` + `pattern/douyin-verdict-opener`；
  D1 → `pattern/consensus-anti-param`；E1/E2/E3 → `spec/wall-mount-tiers` + `tension/thin-body-vs-real-flush`；
  F1 → `voice/bili-geek`（负样本槽）。
- 未蒸馏：F3/F5/E4 场景细节、G 区可信度表的逐条映射。
- 不得蒸馏：把低可信度来源（dazhe.com）的数值升格为断言。
