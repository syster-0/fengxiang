// Package govern 实现 M3 的四类治理 job：Audit / Promote / Decay / Recall。
//
// 每轮治理必须记 log.md（由 okf.Wiki 的写路径负责），审计轨迹不可断。
package govern

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sycglier/tv-opinion-atelier/internal/errx"
	"github.com/sycglier/tv-opinion-atelier/internal/okf"
	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// Change 是一次治理动作。
type Change struct {
	ID    string
	Field string
	From  string
	To    string
	Why   string
	Dry   bool
}

func (c Change) String() string {
	s := c.ID + " " + c.Field + ": " + c.From + " → " + c.To + "（" + c.Why + "）"
	if c.Dry {
		s += " [dry-run]"
	}
	return s
}

// Stats 是一次权重分析的产物。
type Stats struct {
	Concepts      int
	ByTier        map[string]int
	ByVerdict     map[string]int
	ByVoice       map[string]int
	ByType        map[string]int
	MeanWeight    float64
	MeanHumanness float64
	MeanRisk      float64
	Usable        int
	Unusable      map[string]int
	HumannessHist map[string]int
	NeedFeed      map[string]int
}

// Audit 跑全库一致性校验并统计权重分布。
func Audit(w *okf.Wiki) (*okf.Report, Stats, error) {
	rep, err := w.Validate()
	if err != nil {
		return nil, Stats{}, err
	}
	st, err := StatsOf(w)
	if err != nil {
		return rep, st, err
	}
	return rep, st, nil
}

// StatsOf 统计权重与人味分布。
func StatsOf(w *okf.Wiki) (Stats, error) {
	cs, err := w.List()
	if err != nil {
		return Stats{}, err
	}
	st := Stats{
		ByTier: map[string]int{}, ByVerdict: map[string]int{},
		ByVoice: map[string]int{}, ByType: map[string]int{},
		Unusable: map[string]int{}, HumannessHist: map[string]int{},
		NeedFeed: map[string]int{},
	}
	var wSum, hSum, rSum float64
	for _, c := range cs {
		if c.Type == "" {
			continue
		}
		st.Concepts++
		st.ByTier[c.Atelier.Tier]++
		st.ByVerdict[c.Atelier.Verdict]++
		if c.Voice != "" {
			st.ByVoice[c.Voice]++
		}
		st.ByType[c.Type]++
		wSum += c.Atelier.Weight
		hSum += c.Atelier.Humanness
		rSum += c.Atelier.Risk
		st.HumannessHist[bucket(c.Atelier.Humanness)]++
		if c.Usable() {
			st.Usable++
		} else {
			st.Unusable[c.UnusableBecause()]++
			if c.Atelier.Humanness < spec.HumannessGate {
				st.NeedFeed[c.Voice]++
			}
		}
	}
	if st.Concepts > 0 {
		n := float64(st.Concepts)
		st.MeanWeight = okf.Round3(wSum / n)
		st.MeanHumanness = okf.Round3(hSum / n)
		st.MeanRisk = okf.Round3(rSum / n)
	}
	return st, nil
}

func bucket(h float64) string {
	switch {
	case h < 0.2:
		return "0.0-0.2"
	case h < 0.4:
		return "0.2-0.4"
	case h < 0.6:
		return "0.4-0.6（未过人味门）"
	case h < 0.8:
		return "0.6-0.8"
	default:
		return "0.8-1.0"
	}
}

// Promote 把满足条件的 short 概念晋升 long。
//
// 条件（四者同时满足）：tier=short；verdict=verified；weight ≥ PromoteWeight；
// 人味与合规双门通过。
func Promote(w *okf.Wiki, dry bool) ([]Change, error) {
	cs, err := w.List()
	if err != nil {
		return nil, err
	}
	var changes []Change
	for _, c := range cs {
		if c.Atelier.Tier != spec.TierShort {
			continue
		}
		if !c.Usable() {
			continue
		}
		if c.Atelier.Weight < spec.PromoteWeight {
			continue
		}
		ch := Change{ID: c.ID, Field: "tier", From: spec.TierShort, To: spec.TierLong,
			Why: fmt.Sprintf("weight %.2f ≥ %.2f 且 verified 且双门通过", c.Atelier.Weight, spec.PromoteWeight),
			Dry: dry}
		if dry {
			changes = append(changes, ch)
			continue
		}
		c.Atelier.Tier = spec.TierLong
		if err := w.Put(c, "govern-promote", ch.Why); err != nil {
			return changes, err
		}
		changes = append(changes, ch)
	}
	return changes, nil
}

// Decay 把长期零使用的概念降为 archive（删除即审计：写 log 并留 superseded 语义）。
func Decay(w *okf.Wiki, dry bool, now time.Time) ([]Change, error) {
	cs, err := w.List()
	if err != nil {
		return nil, err
	}
	var changes []Change
	for _, c := range cs {
		if c.Atelier.Tier == spec.TierArchive {
			continue
		}
		days, ok := daysSince(c.Atelier.Updated, now)
		if !ok {
			continue
		}
		if days <= float64(spec.DecayIdleDays) {
			continue
		}
		ch := Change{ID: c.ID, Field: "tier", From: c.Atelier.Tier, To: spec.TierArchive,
			Why: fmt.Sprintf("距最近使用 %.0f 天 > %d 天", days, spec.DecayIdleDays), Dry: dry}
		if dry {
			changes = append(changes, ch)
			continue
		}
		from := c.Atelier.Tier
		c.Atelier.Tier = spec.TierArchive
		c.Atelier.Weight = okf.DecayRecency(c.Atelier.Weight, days)
		if err := w.Put(c, "govern-decay",
			fmt.Sprintf("superseded tier %s→archive，%.0f 天零使用；weight 衰减至 %.2f", from, days, c.Atelier.Weight)); err != nil {
			return changes, err
		}
		changes = append(changes, ch)
	}
	return changes, nil
}

// Recall 把指定概念从 archive 恢复到 short 并重新记账。
func Recall(w *okf.Wiki, ids []string) ([]Change, error) {
	var changes []Change
	for _, id := range ids {
		c, err := w.Get(id)
		if err != nil {
			return changes, err
		}
		if c.Atelier.Tier != spec.TierArchive {
			changes = append(changes, Change{ID: id, Field: "tier", From: c.Atelier.Tier, To: c.Atelier.Tier,
				Why: "不在 archive，无需召回"})
			continue
		}
		from := c.Atelier.Tier
		c.Atelier.Tier = spec.TierShort
		if err := w.AccountUse(c); err != nil {
			return changes, err
		}
		if err := w.Put(c, "govern-recall", "从 archive 恢复到 short 并记账"); err != nil {
			return changes, err
		}
		changes = append(changes, Change{ID: id, Field: "tier", From: from, To: spec.TierShort,
			Why: "人工召回，重新参与生成"})
	}
	return changes, nil
}

// Recompute 重算全库权重（五因子），用于权重口径调整后的批量回填。
func Recompute(w *okf.Wiki, dry bool) ([]Change, error) {
	cs, err := w.List()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	var changes []Change
	for _, c := range cs {
		in := okf.NewConceptWeightInput()
		in.UseCount = c.Atelier.UseCount
		in.InLinks = len(c.Atelier.Links)
		in.OutLinks = len(c.Atelier.Links)
		if d, ok := daysSince(c.Atelier.Updated, now); ok {
			in.DaysSinceUse = d
		}
		if c.Verified() {
			in.Explicit = 0.75
			in.Quality = 0.85
		}
		nw := okf.ComputeWeight(in)
		if nw == c.Atelier.Weight {
			continue
		}
		ch := Change{ID: c.ID, Field: "weight",
			From: fmt.Sprintf("%.3f", c.Atelier.Weight), To: fmt.Sprintf("%.3f", nw),
			Why: "五因子重算", Dry: dry}
		if dry {
			changes = append(changes, ch)
			continue
		}
		c.Atelier.Weight = nw
		if err := w.Put(c, "govern-recompute", "五因子重算 weight"); err != nil {
			return changes, err
		}
		changes = append(changes, ch)
	}
	return changes, nil
}

// FeedPlan 依据权重分析给出「缺什么补什么」的投喂清单。
func FeedPlan(st Stats) []string {
	if len(st.NeedFeed) == 0 && st.Concepts > 0 {
		return nil
	}
	type kv struct {
		voice string
		n     int
	}
	var list []kv
	for v, n := range st.NeedFeed {
		list = append(list, kv{v, n})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].n > list[j].n })
	var out []string
	for _, e := range list {
		v := e.voice
		if v == "" {
			v = "（未标声部）"
		}
		out = append(out, fmt.Sprintf("%s：%d 个概念未过人味门，需按该声部补投喂高赞长评后重新蒸馏", v, e.n))
	}
	return out
}

// Line 输出一行统计摘要，供 CLI 与日志复用。
func (s Stats) Line() string {
	var b strings.Builder
	fmt.Fprintf(&b, "概念 %d · 可用 %d · 均权重 %.3f · 均人味 %.3f · 均风险 %.3f",
		s.Concepts, s.Usable, s.MeanWeight, s.MeanHumanness, s.MeanRisk)
	b.WriteString(" | tier")
	for _, k := range sortedKeys(s.ByTier) {
		fmt.Fprintf(&b, " %s=%d", k, s.ByTier[k])
	}
	b.WriteString(" | verdict")
	for _, k := range sortedKeys(s.ByVerdict) {
		fmt.Fprintf(&b, " %s=%d", k, s.ByVerdict[k])
	}
	return b.String()
}

// HumannessLine 输出人味分分布。
func (s Stats) HumannessLine() string {
	var b strings.Builder
	b.WriteString("人味分分布：")
	for _, k := range sortedKeys(s.HumannessHist) {
		fmt.Fprintf(&b, " %s=%d", k, s.HumannessHist[k])
	}
	return b.String()
}

func sortedKeys(m map[string]int) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func daysSince(date string, now time.Time) (float64, bool) {
	date = strings.TrimSpace(date)
	if date == "" {
		return 0, false
	}
	for _, layout := range []string{"2006-01-02", time.RFC3339, "2006-01-02 15:04:05"} {
		if t, err := time.ParseInLocation(layout, date, now.Location()); err == nil {
			return now.Sub(t).Hours() / 24, true
		}
	}
	return 0, false
}

// EnsureWritable 用于在治理前确认 wiki 可写（避免跑到一半才失败）。
func EnsureWritable(w *okf.Wiki) error {
	if w.Root == "" {
		return errx.New(errx.KindInvalid, "govern", "", "wiki 根目录为空")
	}
	return nil
}
