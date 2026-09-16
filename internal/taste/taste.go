// Package taste 实现「去 AI 味」算子引擎与人味打分。
//
// 规则集来源：lieflat-less-ai-tone 的 283 万字对照语料研究
// （629 篇 / 2,826,972 汉字 / 95,551 句 / 45,721 段；生成侧 300 篇跨 5 个模型，
// 人类侧 329 篇）。该研究逐项检验了 26 项流行「AI 文风」候选特征，
// 11 项通过判别力检验（R ≥ 1.25 且组间一致、算子可定位、改写可操作），
// 15 项不成立。本包只实现通过检验的 11 项。
//
// 两条硬边界（来自原研究，不可违反）：
//  1. 白名单式改写：只处理 11 项。未命中任何规则的句子必须逐字保留。
//  2. 反向项必须保留：比喻、正文设问、句内同构排比、单字虚词这些
//     人类用得比 AI 更多的特征，删改会让文本更不像人写。
//
// 打分口径（数字权重分析）：
//
//	excess_i = clamp((rate_i - human_i) / (ai_i - human_i), 0, 1)
//	w_i      = R_i / ΣR
//	aiTone   = Σ w_i · excess_i
//	humanness = 1 - aiTone
//
// 即：把每一项实测频率折算成「超出人类基线多少」，再按区分力加权。
// 人类基线的文本 aiTone≈0，人味分≈1；典型的 AI 文本人味分落在 0.3–0.5。
package taste

import (
	"regexp"
	"sort"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// RuleHit 是一条规则的命中记录。
type RuleHit struct {
	Op       spec.Operator
	Count    int
	Rate     float64 // 归一后的频率（单位由 Op.Per 决定）
	Excess   float64 // 0-1，超出人类基线的程度
	Weight   float64 // 加权系数（R / ΣR）
	Examples []string
}

// MarkerHit 是一条人味增强项的观测。
type MarkerHit struct {
	Feature  string
	Count    int
	Rate     float64 // 每千字
	Baseline float64 // 人类侧基线
	Deficit  bool    // 低于基线即为缺失
	Note     string
}

// Report 是一次打分的完整结果。
type Report struct {
	Chars     int
	Paras     int
	Sentences int
	Hits      []RuleHit
	AITone    float64 // 0-1，越大越像 AI
	Humanness float64 // 1 - AITone
	Gate      bool    // 是否过人味门
	Markers   []MarkerHit
	RiskHits  []string
	Risk      float64
	Notes     []string
}

// Score 对一段文本打分。
func Score(text string) Report {
	body := stripNonProse(text)
	chars := runeLen(body)
	parasAll := splitParas(text)
	paras := len(parasAll)
	sents := splitSentences(body)

	rep := Report{Chars: chars, Paras: paras, Sentences: len(sents)}
	if chars == 0 {
		rep.AITone = 0
		rep.Humanness = 1
		rep.Gate = true
		rep.Notes = append(rep.Notes, "空文本：无可评估内容")
		return rep
	}

	kilo := float64(chars) / 1000.0
	hundredP := float64(paras) / 100.0

	totalR := 0.0
	for _, op := range spec.TasteOperators {
		totalR += op.Ratio
	}

	for _, op := range spec.TasteOperators {
		var count int
		var examples []string
		switch op.Kind {
		case spec.OpStructural:
			count, examples = structuralCount(op.ID, text, parasAll, sents)
		default:
			count, examples = patternCount(op.Patterns, body)
		}

		denom := kilo
		if op.Per == spec.PerHundredP {
			denom = hundredP
		}
		if denom <= 0 {
			denom = 1
		}
		rate := float64(count) / denom

		span := op.AIBase - op.HumanBase
		excess := 0.0
		if span > 0 {
			excess = (rate - op.HumanBase) / span
		}
		excess = clamp01(excess)

		w := op.Ratio / totalR
		if count > 0 {
			rep.Hits = append(rep.Hits, RuleHit{
				Op: op, Count: count, Rate: round3(rate),
				Excess: round3(excess), Weight: round3(w), Examples: examples,
			})
		}
	}

	sort.SliceStable(rep.Hits, func(i, j int) bool { return rep.Hits[i].Excess > rep.Hits[j].Excess })

	aiTone := 0.0
	for _, h := range rep.Hits {
		aiTone += h.Weight * h.Excess
	}
	rep.AITone = round3(clamp01(aiTone))
	rep.Humanness = round3(clamp01(1 - aiTone))
	rep.Gate = rep.Humanness >= spec.HumannessGate

	rep.Markers = markers(body, kilo)
	rep.RiskHits, rep.Risk = spec.RiskScan(body)

	// 观测提示：只提示，不构成改写理由（见 spec.NonDiscriminating）。
	for _, m := range rep.Markers {
		if m.Deficit {
			rep.Notes = append(rep.Notes, "人味增强项偏少："+m.Feature+"（生成时应刻意补足，但不得据此删改既有文本）")
		}
	}
	if !rep.Gate {
		rep.Notes = append(rep.Notes, "未过人味门，需按命中规则回炉改写后重新打分")
	}
	return rep
}

// Hints 返回针对命中的改写指导（按超出程度排序）。
func (r Report) Hints() []string {
	var out []string
	for _, h := range r.Hits {
		out = append(out, "第"+itoa(h.Op.ID)+"条 "+h.Op.Name+"：命中 "+itoa(h.Count)+" 处 → "+h.Op.Fix)
	}
	return out
}

// ---------------------------------------------------------------------------
// 内部实现
// ---------------------------------------------------------------------------

func patternCount(patterns []string, body string) (int, []string) {
	total := 0
	var examples []string
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			continue
		}
		ms := re.FindAllStringIndex(body, -1)
		total += len(ms)
		for i, m := range ms {
			if i >= 2 {
				break
			}
			examples = append(examples, clip(body, m[0], m[1]))
		}
	}
	return total, examples
}

// structuralCount 处理两条需要切句/分段的结构型规则。
func structuralCount(id int, text string, paras []para, sents []string) (int, []string) {
	switch id {
	case 3:
		// 相邻句结构同款：相邻两句逗号数相同、长度档位相同、收尾标点相同。
		count := 0
		var examples []string
		for _, p := range paras {
			ss := splitSentences(p.Text)
			if len(ss) < 2 {
				continue
			}
			for i := 1; i < len(ss); i++ {
				if skeleton(ss[i-1]) == skeleton(ss[i]) {
					count++
					if len(examples) < 3 {
						examples = append(examples, clip(ss[i-1], 0, len(ss[i-1]))+" ⇄ "+clip(ss[i], 0, len(ss[i])))
					}
				}
			}
		}
		return count, examples
	case 11:
		// 段首零回指评论：非首段以评论语开头，且整句无回指成分。
		op := operatorByID(11)
		re := spec.MustCompile(op.Patterns[0])
		count := 0
		var examples []string
		for i, p := range paras {
			if i == 0 {
				continue
			}
			first := firstSentence(p.Text)
			if !re.MatchString(strings.TrimSpace(first)) {
				continue
			}
			if hasAnaphora(first) {
				continue
			}
			count++
			if len(examples) < 3 {
				examples = append(examples, clip(first, 0, len(first)))
			}
		}
		return count, examples
	}
	return 0, nil
}

var anaphora = []string{"这", "那", "其", "此", "上面", "上述", "前述", "该", "它"}

func hasAnaphora(s string) bool {
	for _, a := range anaphora {
		if strings.Contains(s, a) {
			return true
		}
	}
	return false
}

// skeleton 是句子的「句法骨架」指纹：逗号数 + 长度档位 + 收尾标点。
func skeleton(s string) string {
	commas := strings.Count(s, "，") + strings.Count(s, ",")
	r := []rune(strings.TrimSpace(s))
	end := '。'
	if len(r) > 0 {
		end = r[len(r)-1]
	}
	return itoa(commas) + "|" + itoa(len(r)/8) + "|" + string(end)
}

type para struct {
	Index int
	Text  string
}

// splitParas 按空行切段，并剔除标题/表格/代码块/引用/列表项/图片行。
func splitParas(text string) []para {
	var out []para
	inCode := false
	for _, raw := range strings.Split(text, "\n\n") {
		block := strings.TrimSpace(raw)
		if block == "" {
			continue
		}
		if strings.HasPrefix(block, "```") {
			inCode = !inCode
			continue
		}
		if inCode {
			continue
		}
		var kept []string
		for _, line := range strings.Split(block, "\n") {
			t := strings.TrimSpace(line)
			switch {
			case t == "":
				continue
			case strings.HasPrefix(t, "#"):
				continue
			case strings.HasPrefix(t, ">"):
				continue
			case strings.HasPrefix(t, "|"):
				continue
			case strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") || strings.HasPrefix(t, "+ "):
				continue
			case isOrderedItem(t):
				continue
			case strings.HasPrefix(t, "!["):
				continue
			case strings.HasPrefix(t, "---"):
				continue
			}
			kept = append(kept, t)
		}
		if len(kept) == 0 {
			continue
		}
		out = append(out, para{Index: len(out), Text: strings.Join(kept, "")})
	}
	return out
}

func isOrderedItem(t string) bool {
	i := 0
	for i < len(t) && t[i] >= '0' && t[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(t) {
		return false
	}
	// 逐字节比较 '.'，逐 rune 比较全角顿号（多字节，不能退化为 byte 常量）。
	return t[i] == '.' || strings.HasPrefix(t[i:], "、") || strings.HasPrefix(t[i:], "．")
}

// splitSentences 以 。！？； 为界断句，保留长度 ≥4 字的片段。
func splitSentences(text string) []string {
	var out []string
	var cur []rune
	flush := func() {
		s := strings.TrimSpace(string(cur))
		if runeLen(s) >= 4 {
			out = append(out, s)
		}
		cur = cur[:0]
	}
	for _, r := range text {
		cur = append(cur, r)
		switch r {
		case '。', '！', '？', '；', '\n':
			flush()
		}
	}
	if len(cur) > 0 {
		flush()
	}
	return out
}

func firstSentence(p string) string {
	rs := []rune(p)
	for i, r := range rs {
		switch r {
		case '。', '！', '？', '；':
			return string(rs[:i+1])
		}
	}
	if len(rs) > 60 {
		return string(rs[:60])
	}
	return p
}

// stripNonProse 去掉代码块与引用块，用于按千字计频。
func stripNonProse(text string) string {
	var b strings.Builder
	inCode := false
	for _, line := range strings.Split(text, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inCode = !inCode
			continue
		}
		if inCode || strings.HasPrefix(t, ">") {
			continue
		}
		b.WriteString(line)
		b.WriteString("\n")
	}
	return b.String()
}

func markers(body string, kilo float64) []MarkerHit {
	type def struct {
		feature  string
		pattern  string
		baseline float64
		count    func(string) int
		note     string
	}
	defs := []def{
		{"正文设问", `？`, 1.83, nil, "人类频率是生成侧的 17 倍。自问自答是真人推进话题的常用方式"},
		{"比喻标记", `(像[^，。！？；\n]{0,12}(一样|那样|似的))|仿佛|好比|宛如|跟[^，。！？；\n]{0,8}似的`, 0.38, nil, "人类是生成侧的 2.4 倍。喻体要取具体的人或物，不要理想化职业人格"},
		{"口语连接词", `(其实|反倒|干脆|话说回来|偏偏|倒是|反正|讲真|老实说)`, 3.87, nil, "人类是生成侧的约 4 倍。评论区里这类词是真人指纹"},
	}
	var out []MarkerHit
	for _, d := range defs {
		re, err := regexp.Compile(d.pattern)
		if err != nil {
			continue
		}
		n := len(re.FindAllStringIndex(body, -1))
		rate := float64(n) / kilo
		out = append(out, MarkerHit{
			Feature: d.feature, Count: n, Rate: round3(rate),
			Baseline: d.baseline, Deficit: rate < d.baseline, Note: d.note,
		})
	}
	// 单字虚词与数字密度用字符集统计。
	virtual := 0
	for _, r := range body {
		switch r {
		case '就', '很', '了', '吧', '啊', '呢', '嘛':
			virtual++
		}
	}
	vr := float64(virtual) / kilo
	out = append(out, MarkerHit{Feature: "单字虚词（就/很/了/吧/啊/呢/嘛）", Count: virtual,
		Rate: round3(vr), Baseline: 6.45, Deficit: vr < 6.45,
		Note: "人类是生成侧的约 2 倍。这是不可验收项，只作观测，不据此删改"})

	digits := 0
	prevDigit := false
	for _, r := range body {
		isD := r >= '0' && r <= '9'
		if isD && !prevDigit {
			digits++
		}
		prevDigit = isD
	}
	dr := float64(digits) / kilo
	out = append(out, MarkerHit{Feature: "具体数字密度（数字串）", Count: digits,
		Rate: round3(dr), Baseline: 17.92, Deficit: dr < 17.92,
		Note: "人类数字密度是生成侧的 2.8 倍。有实测数据就报出来，别用概括词盖掉"})
	return out
}

func operatorByID(id int) spec.Operator {
	for _, op := range spec.TasteOperators {
		if op.ID == id {
			return op
		}
	}
	return spec.Operator{}
}

func clip(s string, i, j int) string {
	rs := []rune(s)
	if i < 0 {
		i = 0
	}
	if j > len(rs) {
		j = len(rs)
	}
	if i >= j {
		return ""
	}
	out := string(rs[i:j])
	if runeLen(out) > 42 {
		out = string([]rune(out)[:42]) + "…"
	}
	return out
}

func runeLen(s string) int { return len([]rune(s)) }

func clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func round3(f float64) float64 {
	return float64(int(f*1000+0.5)) / 1000
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
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
