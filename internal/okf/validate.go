package okf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// Severity 是校验问题的级别。
type Severity string

const (
	// SevError 阻塞落库/交付。validate 只要有一条 error 即判失败。
	SevError Severity = "error"
	// SevWarn 需人工确认，不阻塞。
	SevWarn Severity = "warn"
	// SevInfo 观察项。
	SevInfo Severity = "info"
)

// Issue 是一条校验问题。
type Issue struct {
	Severity Severity
	Code     string
	Subject  string
	Msg      string
	Fix      string
}

func (i Issue) String() string {
	s := fmt.Sprintf("[%s] %s · %s: %s", i.Severity, i.Code, i.Subject, i.Msg)
	if i.Fix != "" {
		s += " → " + i.Fix
	}
	return s
}

// Report 是校验报告。
type Report struct {
	Issues   []Issue
	Concepts int
	Usable   int
	Verified int
}

// Errors 返回错误级问题数。
func (r *Report) Errors() int {
	n := 0
	for _, i := range r.Issues {
		if i.Severity == SevError {
			n++
		}
	}
	return n
}

// Warnings 返回告警级问题数。
func (r *Report) Warnings() int {
	n := 0
	for _, i := range r.Issues {
		if i.Severity == SevWarn {
			n++
		}
	}
	return n
}

// Valid 报告校验门是否放行。
func (r *Report) Valid() bool { return r.Errors() == 0 }

func (r *Report) add(sev Severity, code, subject, msg, fix string) {
	r.Issues = append(r.Issues, Issue{Severity: sev, Code: code, Subject: subject, Msg: msg, Fix: fix})
}

// testPrompts 是 darwin 格式的测试集。
type testPrompts struct {
	Concept   string   `json:"concept"`
	Triggers  []string `json:"triggers"`
	Decoys    []string `json:"decoys"`
	Negatives []string `json:"negatives"`
}

// Validate 是全库一致性校验——落库与交付前的硬门。
//
// 校验项（对照 okf-wiki 规范第 5 节 + 本领域扩展门禁）：
//  1. 每个概念 frontmatter 完整（type 在词表内、atelier.* 齐全、路径桶与 type 一致）
//  2. 路径与概念 ID 一致（无改名漂移、无解析失败文件）
//  3. Source Map 双向闭合
//  4. verified 概念必有关联 test-prompts.json（含诱饵负例 + 兄弟互斥负例）
//  5. verified 概念必须过 人味门 与 合规门
//  6. index.md 与实际文件一致
//  7. log.md 无断档（每个概念 ID 至少在日志里出现过一次）
func (w *Wiki) Validate() (*Report, error) {
	rep := &Report{}
	cs, err := w.List()
	if err != nil {
		return nil, err
	}
	rep.Concepts = len(cs)

	seen := map[string]bool{}
	for _, c := range cs {
		if seen[c.ID] {
			rep.add(SevError, "dup-id", c.ID, "概念 ID 重复", "路径即 ID，请改名避免冲突")
		}
		seen[c.ID] = true

		if strings.HasPrefix(c.Body, "[解析失败]") {
			rep.add(SevError, "parse-failed", c.ID, strings.TrimPrefix(c.Body, "[解析失败] "),
				"修好 frontmatter 后重跑 validate")
			continue
		}

		for _, iss := range ValidateStructural(c) {
			rep.add(SevError, "structural", c.ID, iss, "")
		}

		if c.Verified() {
			rep.Verified++
			// 门 1：人味
			if c.Atelier.Humanness < spec.HumannessGate {
				rep.add(SevError, "humanness-gate", c.ID,
					fmt.Sprintf("verdict=verified 但人味分 %.2f < 门槛 %.2f", c.Atelier.Humanness, spec.HumannessGate),
					"重跑去 AI 味算子并回炉改写，达标后才能标 verified")
			}
			// 门 2：合规
			if c.Atelier.Risk > spec.RiskGate {
				rep.add(SevError, "risk-gate", c.ID,
					fmt.Sprintf("verdict=verified 但风险分 %.2f > 门槛 %.2f", c.Atelier.Risk, spec.RiskGate),
					"先建 Risk Rule 概念并改写为合规表达")
			}
			// 门 3：测试集
			tp := w.TestPath(c.ID)
			raw, rerr := os.ReadFile(tp)
			if rerr != nil {
				rep.add(SevError, "test-prompts-missing", c.ID,
					"verified 概念缺 test-prompts.json: "+tp,
					"用 `tvop distill darwin` 生成（必须含 decoys 与 negatives）")
			} else {
				var t testPrompts
				if jerr := json.Unmarshal(raw, &t); jerr != nil {
					rep.add(SevError, "test-prompts-invalid", c.ID, "test-prompts.json 不是合法 JSON: "+jerr.Error(), "")
				} else {
					if len(t.Triggers) == 0 {
						rep.add(SevError, "test-prompts-empty", c.ID, "test-prompts.json 缺 triggers", "")
					}
					if len(t.Decoys) == 0 {
						rep.add(SevError, "test-prompts-no-decoy", c.ID,
							"test-prompts.json 缺诱饵负例（decoys）", "诱饵负例不可省略")
					}
					if len(t.Negatives) == 0 {
						rep.add(SevError, "test-prompts-no-sibling", c.ID,
							"test-prompts.json 缺兄弟概念互斥负例（negatives）", "")
					}
				}
			}
		}

		if c.Usable() {
			rep.Usable++
		} else if c.Verified() {
			rep.add(SevWarn, "verified-but-unusable", c.ID, "已 verified 但不可用于生成："+c.UnusableBecause(), "")
		}

		// Source Map 双向闭合（正向）
		if c.Atelier.SourceRef == "" {
			rep.add(SevWarn, "source-ref-empty", c.ID, "缺 atelier.source_ref", "补齐来源指针，保证可溯源")
		} else if _, ok := w.ResolveRef(c.Atelier.SourceRef); !ok {
			rep.add(SevWarn, "source-ref-dangling", c.ID,
				"source_ref 解析不到实体文件: "+c.Atelier.SourceRef,
				"若来源未随包分发，请在 README 明示；否则补回文件")
		}
		sm, serr := w.ReadSourceMap(c.ID)
		if serr != nil {
			rep.add(SevError, "source-map-missing", c.ID, serr.Error(), "用 `tvop okf sourcemap` 建条目")
		} else {
			if sm.Concept != c.ID {
				rep.add(SevError, "source-map-mismatch", c.ID,
					"Source Map 的 concept 字段为 "+sm.Concept+"，与概念 ID 不一致", "")
			}
			if len(sm.Sources) == 0 {
				rep.add(SevError, "source-map-no-source", c.ID, "Source Map 没有任何 sources 条目", "")
			}
		}
	}

	// Source Map 反向闭合：条目指向的概念必须存在
	if sIds, err := w.listSourceMapIDs(); err == nil {
		for _, id := range sIds {
			if !seen[id] {
				rep.add(SevError, "source-map-orphan", id,
					"Source Map 存在但概念文件不存在（反向不闭合）", "删除孤立条目或补回概念")
			}
		}
	} else {
		return nil, err
	}

	// index.md 一致性
	if n, err := w.IndexCount(); err != nil {
		rep.add(SevError, "index-missing", "index.md", err.Error(), "跑 `tvop okf reindex`")
	} else if n != len(cs) {
		rep.add(SevError, "index-stale", "index.md",
			fmt.Sprintf("index.md 有 %d 行、实际 %d 个概念", n, len(cs)), "跑 `tvop okf reindex`")
	}

	// log.md 无断档
	lines, lerr := w.LogLines()
	if lerr != nil {
		rep.add(SevError, "log-missing", "log.md", lerr.Error(), "跑 `tvop init`")
	} else {
		joined := strings.Join(lines, "\n")
		for _, c := range cs {
			if !strings.Contains(joined, c.ID) {
				rep.add(SevError, "log-gap", c.ID, "log.md 里找不到该概念的写入记录", "补记日志或重建概念")
			}
		}
	}

	if rep.Verified == 0 && rep.Concepts > 0 {
		rep.add(SevInfo, "no-verified", "-", "库内尚无 verified 概念，生成链路只能产出草稿", "")
	}
	return rep, nil
}

func (w *Wiki) listSourceMapIDs() ([]string, error) {
	root := filepath.Join(w.Root, DirSourceMap)
	var out []string
	var walk func(dir string) error
	walk = func(dir string) error {
		ents, err := os.ReadDir(dir)
		if err != nil {
			return nil
		}
		for _, e := range ents {
			p := filepath.Join(dir, e.Name())
			if e.IsDir() {
				if err := walk(p); err != nil {
					return err
				}
				continue
			}
			if !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			// 概念 ID 统一用 `/` 作命名空间分隔符；磁盘层用 filepath.Rel
			// 求相对路径后转回 `/`，保证 Windows 反斜杠路径不出错配。
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				continue
			}
			id := filepath.ToSlash(strings.TrimSuffix(rel, ".md"))
			out = append(out, id)
		}
		return nil
	}
	if err := walk(root); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

// WriteTestPrompts 写入 darwin 格式测试集。
func (w *Wiki) WriteTestPrompts(id string, triggers, decoys, negatives []string) error {
	t := testPrompts{Concept: id, Triggers: triggers, Decoys: decoys, Negatives: negatives}
	raw, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	p := w.TestPath(id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, append(raw, '\n'), 0o644)
}
