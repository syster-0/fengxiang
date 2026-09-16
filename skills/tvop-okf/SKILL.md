---
name: tvop-okf
description: OKF Wiki 写入与校验。用户要求把概念落库、查一个概念的内容与权重、重建索引、跑全库校验、登记来源指针（Source Map）、写测试集、按关键词召回概念、或问「这个概念能不能用」时使用。负责写路径四步、Source Map 双向闭合与 validate 全绿门禁。
---

# tvop-okf · OKF Wiki 写入与校验

## 何时使用

- 概念要落库 / 要改。
- 要查某个概念的内容、权重、人味分、能否用于生成。
- 要重建索引、跑全库校验、登记来源。
- 要按关键词召回（找「该用哪几条概念」）。

采集走 `$tvop-crawl`；蒸馏走 `$tvop-distill`；打分走 `$tvop-taste`。

## 三条铁律

1. **路径即概念 ID。** `wiki/concepts/<桶>/<短名>.md` 的 ID 就是 `<桶>/<短名>`。
   **改名 = 换身份**，use_count 与链接不会跟着走。
2. **词表封闭。** 概念类型、声部、tier、verdict、atelier 字段都是封闭词表，
   **禁止临时造字段**。需要新字段就改 `internal/spec`，不要在概念里硬塞。
3. **校验不过 = 落库失败。** `tvop okf validate` 只要有一条 error，这批就不算落库。
   「禁止带病服务」是体系原话。

## 写路径四步（由 `Put` 保证，不要绕过）

1. 取写锁（跨进程，带超时与陈旧锁回收）；
2. 结构校验（词表 + 桶一致 + 必填字段 + 数值区间 + use_count ≥ 1）；
3. 临时文件 + 原子提交；
4. 追加 `log.md`（append-only，**每个概念 ID 至少在日志里出现过一次**，否则 validate 报 log-gap）。

> 直接手写概念文件会绕过第 4 步，导致 `log-gap`。要批量落库时，
> 必须自己补写日志行，格式：
> `- <RFC3339> | <op> | <subject> | <note>`

## 常用命令

```bash
tvop okf new  --type "Voice Persona" --slug bili-geek --title "..." --voice bili-geek \
              --tier long --verdict verified --source-ref sources/tcl/platform-live-corpus.md \
              --humanness 0.81 --risk 0 --body-file body.md --links "pattern/geek-added-caveat,voice/tieba-long"
tvop okf get  <id> [--account] [--json]     # --account 读取即记账（use_count++ 并重算权重）
tvop okf list
tvop okf reindex                            # 依实际文件重编译 index.md
tvop okf validate                           # 全库一致性校验
tvop okf sourcemap --id <id> --source url:https://... --source manual:sources/tcl/x.md --fetched 2026-09-16
tvop okf test --id <id> --triggers "…" --decoys "…" --negatives "…"
tvop okf account <id>
tvop search <关键词...>                      # 召回：命中 × weight × tier × 人味
```

## 两道门（只对 verified 概念生效）

| 门 | 阈值 | 不过怎么办 |
|---|---|---|
| 人味门 | ≥ 0.60 | **回炉改写**，不是调数字 |
| 合规门 | ≤ 0.40 | 先建 Risk Rule 概念，再改写为合规表达 |

`verified` 概念另外**必须**有关联的 `tests/<桶>/<短名>.json`（含 triggers + decoys + negatives）。
`usable = verified 且 人味 ≥ 0.60 且 风险 ≤ 0.40` —— 只有 usable 的概念才能进生成链路。

## validate 检查的七项

1. frontmatter 完整（type 在词表内、atelier.* 齐全、路径桶与 type 一致）
2. 路径与概念 ID 一致（无改名漂移、无解析失败）
3. **Source Map 双向闭合**（概念 ↔ `source-map/<桶>/<短名>.md`）
4. verified 必有关联 test-prompts.json，且含 decoys 与 negatives
5. verified 必须过 人味门 与 合规门
6. `index.md` 与实际文件一致
7. `log.md` 无断档

## 来源指针（Source Map）

- `atelier.source_ref` 必须能解析到实体文件：
  依次在 `wiki/`、包根、`wiki/` 父目录下查找。
  随包分发的来源放 `<包根>/sources/`，源指针写 `sources/tcl/xxx.md`。
- 每个概念都要有 `source-map/<id>.md`，且里面至少一条 `sources` 条目。

## 输出规范

- 落库后报：ID、type、voice、tier、verdict、weight、人味、风险、usable。
- 校验后报：**概念数 / verified 数 / 可用数 / 错误数 / 告警数**。
  有错误就逐条列出（含 `Fix` 建议），不要只报「失败」。
- 召回时每条给出：rank、weight、人味、以及**是否为草稿**。
  草稿概念不能直接用于生成，要明说。

> 冷启动实测基线（2026-09-16）：48 概念 / 33 verified / 33 可用 / 0 错误 / 0 告警。
