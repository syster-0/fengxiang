---
type: Knowledge Gap
title: 缺口：SQD 物理灯珠数与分组方式未公开
atelier:
  weight: 0.577
  tier: short
  verdict: unverified
  use_count: 1
  source_ref: sources/tcl/platform-live-corpus.md
  humanness: 0.933
  resonance: 0.3
  tension: 0.7
  risk: 0
  updated: 2026-09-16
  links:
    - spec/wanxiang-zone-inflation
    - tension/zone-count-vs-real-control
---


## 缺口描述

TCL 公开「万象分区数」，但**未公开物理 LED 灯珠数量与分组方式**。
官方口径是「1 灯珠 = 1 有效分区」，若该口径成立且无插值，则万象分区数应等于物理灯珠数 —— 但**无第三方拆解证实**。

## 触发任务

- 生成「分区数」相关评论时，无法回答「是不是虚标」这类追问。
- TCL 主动攻击竞品分区数不足（见 `claim/samsung-three-lies`）时，**自身欠缺对等可验证性**，
  该缺口会被反向利用。

## 补投喂来源

1. 第三方拆解机构（DisplayMate Labs 类）对 X11L / Q10M Pro / Q9M Pro 背光模组的物理灯珠计数。
2. TCL 官方技术白皮书中关于分区映射（1:1 直驱 vs 算法插值）的说明。
3. 同价位竞品对等拆解的横向对照表。

## 状态

**未补全**。当前处理方式：所有分区相关表述一律降格为「官方称」，并在
`tension/zone-count-vs-real-control` 中把「双向待证」写进中立带。
