---type: Model Spec
title: SQD-Mini LED 三大核心技术栈
atelier:
  weight: 0.627
  tier: long
  verdict: unverified
  use_count: 1
  source_ref: sources/tcl/sqd-product-corpus.md
  humanness: 0.865
  resonance: 0.3
  tension: 0.35
  risk: 0
  updated: 2026-09-16
  links:
    - spec/wanxiang-zone-inflation
    - spec/global-vs-window-gamut
    - claim/one-zone-beats-many
---

## 参数名

**SQD-Mini LED** —— TCL 2025-09-15 首创，「Super Quantum Dot」体系。三件套：

| # | 名称 | 管什么 | 关键量化 |
|---|---|---|---|
| ① | **万象分区**（Precise Dimming Zones） | 控光 | 七大底层技术；亮度 +53.8%；背光均匀性 +59.4%；控光能力 +41.2%；光学稳定性 ×8；26bit 动态调光 |
| ② | **超级量子点**（Super QLED） | 背光 | 色点精度 +69%；复合纳米金刚结构；理论寿命 6 万 → **10 万小时** |
| ③ | **晶粹高色阻 / 蝶翼华曜屏** | 屏幕 | 色域 +33%；色纯度 +40%；反射率 ≈1.8%；视角 >178°；原生对比度 7000:1（HVA 2.0） |

## 常识基线

- 普通 Mini LED：分区数从几百到一两千，无系统级控光链路，亮度为瞬时峰值。
- OLED：自发光、黑位纯，但峰值亮度与寿命受限、大尺寸成本高。
- **SQD 的定位是「液晶背光阵营的控光与色域上限」，不是 OLED 的替代品** —— 本库一律不做「超越 OLED」表述。

## 常见误读

1. **把「万象分区」当成普通分区数** —— 官方表述为「一区顶多区」，一颗灯珠即一个有效分区。
   于是「8052 个万象分区」与「16000 级普通分区」不可直接数字比较（见 `spec/wanxiang-zone-inflation`）。
2. **把「绚彩 XDR 10000nits」当成可长时间维持的整屏亮度** —— 它是 HDR 小窗口峰值 + 持久高亮的组合，
   不是整屏 10000nits（见 `spec/xdr-sustained-vs-transient`）。
3. **把「100% BT.2020 全局高色域」当成所有内容都满色域** —— 官方基线是「不挑片源、不随画面复杂而衰减」，
   与「纯色测试图跑满」是两个不同命题（见 `spec/global-vs-window-gamut`）。
4. **把晶粹高色阻当成面板** —— 它是屏幕上的**色阻材料层**，不是发光器件。

## 与竞品差异

| 维度 | SQD 路径 | RGB-Mini LED 路径 |
|---|---|---|
| 白光来源 | 蓝光灯珠 + 超级量子点转换 | 红绿蓝三色灯珠混光 |
| 串色风险 | 背光始终纯净白光，源头规避 | 相邻分区异色灯珠互扰，天生存在 |
| 分区效率 | 1 灯珠 = 1 分区 | 3 灯珠一组 = 1 分区（官方口径） |
| 混光距离 | 短，机身可做薄（X11L ≈20–24mm） | 长，需大 OD |

## 可验证来源

TCL 官网产品页、2026-03-17 春季发布会官方资料、新华社 2026-03-18 报道、
`platform-live-corpus.md` A1/A2 实测描写、frandroid 独立评测。
**不可验证**：七大底层技术的分项百分比（+53.8% / +59.4% / +41.2%）无第三方复现记录，引用需标注「官方口径」。
