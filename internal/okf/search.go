package okf

import (
	"sort"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// Query 是召回请求。
type Query struct {
	Terms        []string // 关键词（全部命中加分，任一命中即入选）
	Type         string   // 限定类型
	Voice        string   // 限定声部
	MinHumanness float64  // 人味下限
	MaxRisk      float64  // 风险上限（0 表示不限制）
	OnlyUsable   bool     // 只要可用于生成的
	IncludeArch  bool     // 是否包含 archive
	Top          int
}

// Hit 是一条召回结果。
type Hit struct {
	Concept *Concept
	Score   float64 // 命中分（字段加权）
	Rank    float64 // 排序分（命中分 × 权重 × tier × 人味）
	Snippet string
	Fields  []string // 命中字段：title / id / body / voice / type
}

// Search 实现召回协议：
// grep 检索 → 命中 × weight × tier 排序 → 返回概念集合。
func (w *Wiki) Search(q Query) ([]Hit, error) {
	cs, err := w.List()
	if err != nil {
		return nil, err
	}
	if q.Top <= 0 {
		q.Top = spec.DefaultTopK
	}
	terms := make([]string, 0, len(q.Terms))
	for _, t := range q.Terms {
		if t = strings.TrimSpace(strings.ToLower(t)); t != "" {
			terms = append(terms, t)
		}
	}

	var hits []Hit
	for _, c := range cs {
		if c.Archived() && !q.IncludeArch {
			continue
		}
		if c.Type == "" {
			continue // 解析失败的概念不参与召回
		}
		if q.Type != "" && c.Type != q.Type {
			continue
		}
		if q.Voice != "" && c.Voice != q.Voice {
			continue
		}
		if q.MinHumanness > 0 && c.Atelier.Humanness < q.MinHumanness {
			continue
		}
		if q.MaxRisk > 0 && c.Atelier.Risk > q.MaxRisk {
			continue
		}
		if q.OnlyUsable && !c.Usable() {
			continue
		}

		var score float64
		var fields []string
		idL := strings.ToLower(c.ID)
		titleL := strings.ToLower(c.Title)
		bodyL := strings.ToLower(c.Body)
		typeL := strings.ToLower(c.Type)
		voiceL := strings.ToLower(c.Voice)

		for _, t := range terms {
			if strings.Contains(titleL, t) {
				score += 3
				fields = appendUnique(fields, "title")
			}
			if strings.Contains(idL, t) {
				score += 2
				fields = appendUnique(fields, "id")
			}
			if strings.Contains(bodyL, t) {
				score += 1
				fields = appendUnique(fields, "body")
			}
			if strings.Contains(typeL, t) || strings.Contains(voiceL, t) {
				score += 1.5
				fields = appendUnique(fields, "meta")
			}
		}
		if len(terms) > 0 && score == 0 {
			continue
		}
		if len(terms) == 0 {
			score = 1
		}

		tierMul := 1.0
		switch c.Atelier.Tier {
		case spec.TierLong:
			tierMul = 1.5
		case spec.TierArchive:
			tierMul = 0.4
		}
		rank := Round3(score * tierMul * c.Rank())
		hits = append(hits, Hit{
			Concept: c,
			Score:   score,
			Rank:    rank,
			Snippet: Snippet(c.Body, terms, 140),
			Fields:  fields,
		})
	}

	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Rank != hits[j].Rank {
			return hits[i].Rank > hits[j].Rank
		}
		return hits[i].Concept.ID < hits[j].Concept.ID
	})
	if len(hits) > q.Top {
		hits = hits[:q.Top]
	}
	return hits, nil
}

// AccountUse 执行「读取即记账」：use_count++，并按五因子重算权重。
func (w *Wiki) AccountUse(c *Concept) error {
	c.Atelier.UseCount++
	in, out := len(c.Atelier.Links), 0
	for _, l := range c.Atelier.Links {
		_ = l
		out++
	}
	wi := NewConceptWeightInput()
	wi.UseCount = c.Atelier.UseCount
	wi.InLinks = in
	wi.OutLinks = out
	wi.Explicit = 0.5
	if c.Verified() {
		wi.Explicit = 0.75 // 已过压测的概念，显式反馈基线更高
	}
	wi.Quality = 0.7
	if c.Verified() {
		wi.Quality = 0.85
	}
	c.Atelier.Weight = ComputeWeight(wi)
	return w.Put(c, "recall", "读取即记账 use_count="+itoa(c.Atelier.UseCount))
}

// Gap 是一条知识缺口。
type Gap struct {
	Term    string
	Reason  string
	Suggest string
}

// DetectGaps 按任务要素清单检测知识缺口。
//
// 判定口径：某个关键词在全库零命中，或只命中不可用于生成的概念，
// 就是缺口——缺什么补投喂什么。
func (w *Wiki) DetectGaps(terms []string, voice string) ([]Gap, error) {
	var gaps []Gap
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		all, err := w.Search(Query{Terms: []string{t}, Voice: voice, Top: 50, IncludeArch: true})
		if err != nil {
			return nil, err
		}
		if len(all) == 0 {
			gaps = append(gaps, Gap{Term: t, Reason: "全库零命中",
				Suggest: "按声部 " + voice + " 补投喂「" + t + "」相关语料并蒸馏入库"})
			continue
		}
		usable := 0
		for _, h := range all {
			if h.Concept.Usable() {
				usable++
			}
		}
		if usable == 0 {
			gaps = append(gaps, Gap{Term: t, Reason: "命中的概念全部不可用于生成",
				Suggest: "先把命中概念的 verdict / 人味分 / 风险分提到门槛之上"})
		}
	}
	return gaps, nil
}

// Snippet 截取包含任一关键词的片段。
func Snippet(body string, terms []string, width int) string {
	flat := strings.Join(strings.Fields(body), " ")
	if len(flat) <= width {
		return flat
	}
	pos := -1
	low := strings.ToLower(flat)
	for _, t := range terms {
		if i := strings.Index(low, t); i >= 0 && (pos < 0 || i < pos) {
			pos = i
		}
	}
	if pos < 0 {
		pos = 0
	}
	start := pos - width/3
	if start < 0 {
		start = 0
	}
	end := start + width
	if end > len(flat) {
		end = len(flat)
		start = end - width
		if start < 0 {
			start = 0
		}
	}
	s := flat[start:end]
	if start > 0 {
		s = "…" + s
	}
	if end < len(flat) {
		s += "…"
	}
	return s
}

func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
