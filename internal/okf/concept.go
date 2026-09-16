package okf

import (
	"path/filepath"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/errx"
	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// Concept 是库内一个概念。路径即 ID。
type Concept struct {
	ID       string // 相对 wiki/concepts 的路径，去掉 .md，正斜杠分隔
	Path     string // 磁盘绝对路径
	Type     string // 必须在 spec 封闭词表内
	Title    string
	Voice    string // 声部归属，可为空（非声部相关概念）
	Platform string // 可选：平台标识
	Command  string // 可选：Runbook 的命令
	Top      map[string]string
	Atelier  Atelier
	Body     string
}

// Rank 是生成排序分。
func (c *Concept) Rank() float64 { return Rank(c.Atelier) }

// Verified 报告是否已过三重验证。
func (c *Concept) Verified() bool { return c.Atelier.Verdict == spec.VerdictVerified }

// Archived 报告是否已归档。
func (c *Concept) Archived() bool { return c.Atelier.Tier == spec.TierArchive }

// Usable 报告该概念是否允许用于生成评论。
//
// 三道门同时满足才允许：
//  1. 已过三重验证（未入库不使用 → 未验证不生成）
//  2. 人味分 ≥ 人味门（低于门槛的语料生成出来就是 AI 味）
//  3. 合规风险 ≤ 风险门（高风险必须先过 Risk Rule 复核）
//
// 已归档的概念不参与生成。
func (c *Concept) Usable() bool {
	if c.Archived() {
		return false
	}
	if !c.Verified() {
		return false
	}
	if c.Atelier.Humanness < spec.HumannessGate {
		return false
	}
	if c.Atelier.Risk > spec.RiskGate {
		return false
	}
	return true
}

// UnusableBecause 返回不可用的原因，可用时返回空串。
func (c *Concept) UnusableBecause() string {
	switch {
	case c.Archived():
		return "已归档（tier=archive）"
	case !c.Verified():
		return "未过三重验证（verdict=" + c.Atelier.Verdict + "）"
	case c.Atelier.Humanness < spec.HumannessGate:
		return "人味分低于门槛"
	case c.Atelier.Risk > spec.RiskGate:
		return "合规风险高于门槛"
	}
	return ""
}

// Bucket 返回该类型对应的目录桶名（= PathHint 去掉斜杠）。
func Bucket(t string) (string, bool) {
	d, ok := spec.TypeDocOf(t)
	if !ok {
		return "", false
	}
	return strings.TrimSuffix(d.PathHint, "/"), true
}

// IDFor 由类型与短名拼出概念 ID。
func IDFor(t, slug string) (string, error) {
	b, ok := Bucket(t)
	if !ok {
		return "", errx.Errorf(errx.KindInvalid, "okf.id", slug,
			"类型 %q 不在封闭词表内（可选：%s）", t, spec.JoinTypes(" / "))
	}
	if slug == "" {
		return "", errx.New(errx.KindInvalid, "okf.id", "", "短名不能为空")
	}
	return b + "/" + slug, nil
}

// IDFromPath 由磁盘路径推出概念 ID。
func IDFromPath(conceptsRoot, abs string) (string, error) {
	rel, err := filepath.Rel(conceptsRoot, abs)
	if err != nil {
		return "", errx.Wrap(errx.KindInternal, "okf.id", abs, err)
	}
	rel = filepath.ToSlash(rel)
	return strings.TrimSuffix(rel, ".md"), nil
}

// ValidateStructural 做结构与词表校验。写路径第三步（提交前）必过。
func ValidateStructural(c *Concept) []string {
	var issues []string
	if c.Type == "" {
		issues = append(issues, "缺少 type")
	} else if !spec.IsValidType(c.Type) {
		issues = append(issues, "type 不在封闭词表内: "+c.Type)
	}
	if c.Title == "" {
		issues = append(issues, "缺少 title")
	}
	if c.ID == "" {
		issues = append(issues, "缺少概念 ID（路径即 ID）")
	} else {
		seg := strings.Split(c.ID, "/")
		if len(seg) < 2 {
			issues = append(issues, "概念 ID 必须是 <桶>/<短名> 形式")
		} else if b, ok := Bucket(c.Type); ok && seg[0] != b {
			issues = append(issues, "路径桶与 type 不匹配：路径在 "+seg[0]+"/，而 "+c.Type+" 应落在 "+b+"/")
		}
	}
	if c.Voice != "" && !spec.IsValidVoice(c.Voice) {
		issues = append(issues, "voice 不在封闭声部表内: "+c.Voice)
	}
	if c.Atelier.Tier == "" {
		issues = append(issues, "缺少 atelier.tier")
	} else if !spec.IsValidTier(c.Atelier.Tier) {
		issues = append(issues, "atelier.tier 非法: "+c.Atelier.Tier)
	}
	if c.Atelier.Verdict == "" {
		issues = append(issues, "缺少 atelier.verdict")
	} else if !spec.IsValidVerdict(c.Atelier.Verdict) {
		issues = append(issues, "atelier.verdict 非法: "+c.Atelier.Verdict)
	}
	if c.Atelier.Weight < 0 || c.Atelier.Weight > 1 {
		issues = append(issues, "atelier.weight 必须在 [0,1]")
	}
	for name, v := range map[string]float64{
		"humanness": c.Atelier.Humanness,
		"resonance": c.Atelier.Resonance,
		"tension":   c.Atelier.Tension,
		"risk":      c.Atelier.Risk,
	} {
		if v < 0 || v > 1 {
			issues = append(issues, "atelier."+name+" 必须在 [0,1]")
		}
	}
	if c.Atelier.UseCount < 1 {
		issues = append(issues, "atelier.use_count 至少为 1（创建即记一次使用）")
	}
	// 类型声明的必填字段。
	if d, ok := spec.TypeDocOf(c.Type); ok {
		for _, req := range d.Requires {
			if req == "voice" {
				if c.Voice == "" {
					issues = append(issues, c.Type+" 必须声明 voice")
				}
				continue
			}
			if req == "platform" {
				if c.Platform == "" {
					issues = append(issues, c.Type+" 必须声明 platform")
				}
				continue
			}
			if req == "command" {
				if c.Command == "" {
					issues = append(issues, c.Type+" 必须声明 command")
				}
				continue
			}
			if c.Top[req] == "" && c.AtelierValue(req) == "" {
				issues = append(issues, c.Type+" 必须声明 "+req)
			}
		}
	}
	return issues
}

// AtelierValue 由字段名取元数据标量（供必填校验复用）。
func (c *Concept) AtelierValue(name string) string {
	switch name {
	case "voice":
		return c.Voice
	case "platform":
		return c.Platform
	case "command":
		return c.Command
	}
	return c.Top[name]
}
