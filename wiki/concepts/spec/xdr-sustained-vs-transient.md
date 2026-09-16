---type: Model Spec
title: 绚彩 XDR：持久高亮与瞬时峰值的区别
atelier:
  weight: 0.577
  tier: short
  verdict: unverified
  use_count: 1
  source_ref: sources/tcl/sqd-product-corpus.md
  humanness: 0.919
  resonance: 0.4
  tension: 0.45
  risk: 0
  updated: 2026-09-16
  links:
    - spec/sqd-three-core
    - tension/spec-sheet-vs-eyes
---

## 参数名

**绚彩 XDR** —— TCL 对峰值亮度的实现方式定位：强调**持久高亮**，
即在画面高亮物体快速移动时背光仍维持高亮输出，避免亮度骤降；并针对 SDR / HDR / 超 HDR 自适应调光曲线。

## 常识基线

- 行业惯例：「峰值亮度」多为**瞬时小窗口峰值**（常以 10% 窗口、限定秒数测得）。
- 瞬时时长的差异会让同为「2000nits」的两台机器，在长镜头高亮场景下表现明显不同。
- 官方量化：X11L 10000nits / Q10M Pro 8500 / Q10M 6000 / Q9M Pro 5000 / T7M Ultra 3000 / T7M Pro 2200。

## 常见误读

1. **把标称峰值当整屏亮度** —— 10000nits 是 HDR 小窗口条件，不是整屏。三星、海信、索尼同在 3000–5000nits 量级，
   别家也有小窗口能到相近数值，差异在**能维持多久**。
2. **认为高亮度只有白天有用** —— 反例：本库 `hot-consensus` 声部的共识是「白天开窗帘，2200nits 就真不是 2200」，
   高亮度的价值在**抗环境光**，这恰恰是客厅场景的刚需，不是影音室专属。
3. **把 XDR 与 HDR 格式混为一谈** —— XDR 是背光控制策略，HDR10+ / Dolby Vision 是内容格式，两者层级不同。

## 与竞品差异

对位机型普遍宣传相近峰值，但**公开材料里只有 TCL 明确把「持久」作为差异化主张**。
这既是优势叙事，也是可被追问点：「持久」缺量化标准（维持多久？衰减多少？），目前无公开测试协议。

## 可验证来源

`sqd-product-corpus.md` 第 2.3 节；各系列参数表。
**不可验证**：持久高亮的时长阈值与衰减曲线定义。
