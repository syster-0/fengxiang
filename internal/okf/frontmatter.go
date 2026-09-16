// Package okf 实现 M3：OKF Wiki 知识库的组织、维护与更新。
//
// 格式契约：
//   - 路径即概念 ID：wiki/concepts/<path>.md 的 ID 就是 <path>（去掉 .md）。改名 = 换身份。
//   - frontmatter 必填 type（词表在 spec 包，封闭）。
//   - 顶层标量：type / title / voice / platform / command。
//   - atelier.* 命名空间承载元数据：weight / tier / verdict / use_count / source_ref，
//     以及本领域扩展：humanness / resonance / tension / risk / updated / links。
//
// 本包不做蒸馏（M2 的职责），也不跑服务（M1 出口的职责）。
package okf

import (
	"sort"
	"strconv"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/errx"
)

// Atelier 是 atelier.* 命名空间承载的元数据。
type Atelier struct {
	// 基座五字段（体系通用）。
	Weight    float64
	Tier      string
	Verdict   string
	UseCount  int
	SourceRef string

	// 领域扩展：电视舆情运营专属。
	Humanness float64 // 人味分 0-1（去 AI 味算子打分）
	Resonance float64 // 共鸣度 0-1（点赞/回复比归一）
	Tension   float64 // 对立度 0-1（正反方回复比）
	Risk      float64 // 合规风险 0-1（spec.RiskScan 产物）
	Updated   string  // 最近一次写入日期 YYYY-MM-DD
	Links     []string
}

// topOrder 是顶层标量的规范输出顺序。
var topOrder = []string{"type", "title", "voice", "platform", "command"}

// splitFrontmatter 切出 frontmatter 文本与正文。
func splitFrontmatter(content string) (head string, body string, err error) {
	s := strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(s, "---") {
		return "", "", errx.New(errx.KindInvalid, "okf.frontmatter", "",
			"缺少 YAML frontmatter（文件必须以 --- 开头）")
	}
	rest := strings.TrimPrefix(strings.TrimPrefix(s[3:], "\r\n"), "\n")
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return "", "", errx.New(errx.KindInvalid, "okf.frontmatter", "",
			"frontmatter 未闭合（找不到结束的 ---）")
	}
	head = rest[:idx]
	body = strings.TrimPrefix(rest[idx+4:], "\r\n")
	body = strings.TrimPrefix(body, "\n")
	return head, body, nil
}

// parseFrontmatter 解析受约束的 YAML 子集。
//
// 支持：顶层 key: value；唯一的嵌套块 atelier: 下的 key: value；
// 以及在嵌套块内以 "- " 开头的字符串列表。
// 不支持多行标量、锚点、流式集合——超出即报错，绝不静默吞掉。
func parseFrontmatter(head string) (map[string]string, Atelier, error) {
	top := map[string]string{}
	var at Atelier
	inAtelier := false
	listKey := ""

	for i, raw := range strings.Split(head, "\n") {
		line := strings.TrimRight(raw, "\r")
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indented := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")

		if indented {
			if !inAtelier {
				return nil, at, errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
					"第 %d 行出现缩进内容但不在 atelier: 块内", i+1)
			}
			if strings.HasPrefix(trimmed, "- ") {
				if listKey != "links" {
					return nil, at, errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
						"第 %d 行是列表项，但当前列表键不是 links", i+1)
				}
				at.Links = append(at.Links, unquote(strings.TrimSpace(trimmed[2:])))
				continue
			}
			k, v, ok := cutKV(trimmed)
			if !ok {
				return nil, at, errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
					"第 %d 行不是合法的 key: value：%q", i+1, trimmed)
			}
			if v == "" {
				listKey = k
				continue
			}
			listKey = ""
			if err := applyAtelier(&at, k, v, i+1); err != nil {
				return nil, at, err
			}
			continue
		}

		listKey = ""
		k, v, ok := cutKV(trimmed)
		if !ok {
			return nil, at, errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
				"第 %d 行不是合法的 key: value：%q", i+1, trimmed)
		}
		if k == "atelier" {
			inAtelier = true
			continue
		}
		inAtelier = false
		top[k] = unquote(v)
	}
	return top, at, nil
}

func applyAtelier(at *Atelier, k, v string, lineno int) error {
	vs := unquote(v)
	num := func() (float64, error) {
		f, err := strconv.ParseFloat(vs, 64)
		if err != nil {
			return 0, errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
				"第 %d 行 atelier.%s 应为数字，得到 %q", lineno, k, vs)
		}
		return f, nil
	}
	switch k {
	case "weight", "humanness", "resonance", "tension", "risk":
		f, err := num()
		if err != nil {
			return err
		}
		switch k {
		case "weight":
			at.Weight = f
		case "humanness":
			at.Humanness = f
		case "resonance":
			at.Resonance = f
		case "tension":
			at.Tension = f
		case "risk":
			at.Risk = f
		}
	case "use_count":
		n, err := strconv.Atoi(vs)
		if err != nil {
			return errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
				"第 %d 行 atelier.use_count 应为整数，得到 %q", lineno, vs)
		}
		at.UseCount = n
	case "tier":
		at.Tier = vs
	case "verdict":
		at.Verdict = vs
	case "source_ref":
		at.SourceRef = vs
	case "updated":
		at.Updated = vs
	case "links":
		// 空值 → 进入列表模式
	default:
		return errx.Errorf(errx.KindInvalid, "okf.frontmatter", "",
			"第 %d 行出现未声明的 atelier 字段 %q（词表封闭，禁止临时造字段）", lineno, k)
	}
	return nil
}

func cutKV(s string) (string, string, bool) {
	i := strings.Index(s, ":")
	if i <= 0 {
		return "", "", false
	}
	return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// renderFrontmatter 按规范顺序渲染 frontmatter，保证 diff 稳定。
func renderFrontmatter(top map[string]string, at Atelier) string {
	var b strings.Builder
	b.WriteString("---\n")
	for _, k := range topOrder {
		if v := top[k]; v != "" {
			b.WriteString(k + ": " + quoteIfNeeded(v) + "\n")
		}
	}
	var extra []string
	for k := range top {
		if !contains(topOrder, k) {
			extra = append(extra, k)
		}
	}
	sort.Strings(extra)
	for _, k := range extra {
		b.WriteString(k + ": " + quoteIfNeeded(top[k]) + "\n")
	}

	b.WriteString("atelier:\n")
	f := func(k string, v float64) {
		b.WriteString("  " + k + ": " + strconv.FormatFloat(Round3(v), 'f', -1, 64) + "\n")
	}
	f("weight", at.Weight)
	b.WriteString("  tier: " + at.Tier + "\n")
	b.WriteString("  verdict: " + at.Verdict + "\n")
	b.WriteString("  use_count: " + strconv.Itoa(at.UseCount) + "\n")
	b.WriteString("  source_ref: " + quoteIfNeeded(at.SourceRef) + "\n")
	f("humanness", at.Humanness)
	f("resonance", at.Resonance)
	f("tension", at.Tension)
	f("risk", at.Risk)
	if at.Updated != "" {
		b.WriteString("  updated: " + at.Updated + "\n")
	}
	if len(at.Links) > 0 {
		b.WriteString("  links:\n")
		for _, l := range at.Links {
			b.WriteString("    - " + l + "\n")
		}
	}
	b.WriteString("---\n")
	return b.String()
}

func quoteIfNeeded(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, ":#\"'\n") || strings.HasPrefix(s, " ") || strings.HasSuffix(s, " ") {
		return strconv.Quote(s)
	}
	return s
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
