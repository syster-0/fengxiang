---
type: Glossary
title: Mini LED 三条技术线辨析
atelier:
  weight: 0.577
  tier: short
  verdict: unverified
  use_count: 1
  source_ref: sources/tcl/sqd-product-corpus.md
  humanness: 0.917
  resonance: 0.35
  tension: 0.3
  risk: 0
  updated: 2026-09-16
  links:
    - spec/sqd-three-core
    - spec/sqd-lineup-2026
---


## 术语

**Mini LED 三条技术线**：QD-Mini LED / RGB-Mini LED / SQD-Mini LED。
三者共用「Mini LED 背光」外衣，但发光与控色原理不同。

## 定义

| 技术线 | 原理 | TCL 代表机型 | 是否 SQD |
|---|---|---|---|
| **QD-Mini LED** | 蓝光背光 + 量子点膜，按区控光（第四代液晶电视路线） | Q10L / Q10L Pro、T5M Plus、V68M Plus | ❌ |
| **RGB-Mini LED** | 红绿蓝三色灯珠直接发光，屏下方案 | Q10M Ultra、Q9M、RM9L、Q7D | ❌ |
| **SQD-Mini LED** | 蓝光灯珠 + 超级量子点 + 晶粹高色阻，**屏上方案** | X11L、Q10M Pro、Q10M、Q9M Art、Q9M Pro、T7M Ultra、T7M Pro | ✅ |

一句话区分：**SQD 是「白光背光 + 超级量子点 + 高色阻屏」；RGB 是「三色灯珠直接发光」。**

## 误用

1. **把 #qdminiled 话题标签下的内容计入 SQD** —— 样本中大量 `#qdminiled` 内容（V68M Plus、T5M Plus）
   属 QD-Mini LED，**不应计入 SQD 产品清单**。
2. **把抖音营销内容里的多型号混排当产品归属依据** —— 样本中出现「TCL Art 7M」配 `#SQD-MiniLED` 标签
   并提及 Q9M Pro，属营销内容混排，**不能据此判定 Art 7M 使用 SQD**。
3. **把雷鸟当 TCL 主品牌同线** —— 雷鸟（RayNeo / FFALCON）为 TCL 旗下子品牌但产品线独立；
   样本中 85鹤6 Ultra（85S595C Ultra）标注为 QD-MiniLED，非 SQD。
4. **只看「Mini LED」字样下单** —— 商家宣传页常只写「Mini LED」，不标技术方案。

## 互斥负例

- 「Q9M 是 SQD」→ **错**，Q9M 是 RGB-Mini LED；Q9M **Pro** 才是 SQD。仅一字之差。
- 「Q10M 系列都叫 Q10M 所以同技术」→ **错**，Q10M / Q10M Pro 是 SQD，Q10M Ultra 是 RGB。
- 「带 SQD 标签的短视频都在讲 SQD 机型」→ **错**，标签可被营销滥用。
