// Package spec 是运行时契约：概念类型词表、声部枚举、权重口径、人味算子表。
//
// 词表封闭：任何未在本文件声明的 type / voice / operator 一律拒用。
// 新增类型必须一次性写进本文件并同步 okf 校验器——禁止运行中临时造类型。
package spec

import "strings"

// ---------------------------------------------------------------------------
// 1. 概念类型词表（M3 type，封闭）
// ---------------------------------------------------------------------------

// 基座词表：体系通用，任何领域专家都继承，不重复声明。
const (
	TypePlaybook     = "Playbook"
	TypeRunbook      = "Runbook"
	TypeTranscript   = "Transcript Segment"
	TypeDecision     = "Decision"
	TypeGlossary     = "Glossary"
	TypeStyleGuide   = "Style Guide"
	TypeKnowledgeGap = "Knowledge Gap"
	TypeSource       = "Source"
)

// 领域扩展词表：电视行业舆情运营专属，一次性声明。
const (
	// TypeCommentPattern 评论范式：可复用的表达骨架 + 触发场景 + 声部归属。
	TypeCommentPattern = "Comment Pattern"
	// TypeVoicePersona 声部人格：nuwa 蒸馏出的某声部表达 DNA。
	TypeVoicePersona = "Voice Persona"
	// TypeTensionAxis 对立轴：一个争议话题的站位谱系与两侧论据。
	TypeTensionAxis = "Tension Axis"
	// TypeModelSpec 选型参数锚：尺寸/背光/分区/刷新/接口/芯片等硬参数的常识基线。
	TypeModelSpec = "Model Spec"
	// TypeMarketClaim 厂商话术：卖点声明与可反驳点。
	TypeMarketClaim = "Market Claim"
	// TypeRiskRule 合规红线：广告法绝对化用语、贬损竞品、未证实参数。
	TypeRiskRule = "Risk Rule"
	// TypeSignal 舆情信号：权重分析产物（热度、起量、异常）。
	TypeSignal = "Signal"
	// TypeAudienceSegment 受众切片：客厅党、游戏党、父母换机等目标人群。
	TypeAudienceSegment = "Audience Segment"
)

// BaseTypes 是基座词表。
var BaseTypes = []string{
	TypePlaybook, TypeRunbook, TypeTranscript, TypeDecision,
	TypeGlossary, TypeStyleGuide, TypeKnowledgeGap, TypeSource,
}

// DomainTypes 是电视舆情领域的扩展词表。
var DomainTypes = []string{
	TypeCommentPattern, TypeVoicePersona, TypeTensionAxis, TypeModelSpec,
	TypeMarketClaim, TypeRiskRule, TypeSignal, TypeAudienceSegment,
}

// TypeDoc 描述一个概念类型的约束。
type TypeDoc struct {
	Type      string
	Base      bool
	Zh        string
	PathHint  string
	Requires  []string // frontmatter 必填字段（除通用 atelier.* 之外）
	BodyShape string
}

// TypeDocs 是全部类型的落地约束表。
var TypeDocs = []TypeDoc{
	{Type: TypePlaybook, Base: true, Zh: "流程/方法论", PathHint: "playbook/",
		Requires: []string{"voice"}, BodyShape: "R 触发场景 / I 框架 / A 步骤 / B 失效边界"},
	{Type: TypeRunbook, Base: true, Zh: "可执行操作手册", PathHint: "runbook/",
		Requires: []string{"command"}, BodyShape: "前置条件 / 命令序列 / 期望输出 / 失败处置"},
	{Type: TypeTranscript, Base: true, Zh: "转写语义段", PathHint: "transcript/",
		Requires: []string{"voice", "platform"}, BodyShape: "原文 + 时间戳/楼层锚点 + 语义标注"},
	{Type: TypeDecision, Base: true, Zh: "决策记录（含反例）", PathHint: "decision/",
		Requires: []string{}, BodyShape: "背景 / 决定 / 为什么不做另一条路 / 复评条件"},
	{Type: TypeGlossary, Base: true, Zh: "术语表", PathHint: "glossary/",
		Requires: []string{}, BodyShape: "术语 / 定义 / 误用 / 互斥负例"},
	{Type: TypeStyleGuide, Base: true, Zh: "风格指南（人格 DNA）", PathHint: "style/",
		Requires: []string{"voice"}, BodyShape: "句长分布 / 标点习惯 / 用词偏好 / 钩子节奏"},
	{Type: TypeKnowledgeGap, Base: true, Zh: "知识缺口", PathHint: "gap/",
		Requires: []string{}, BodyShape: "缺口描述 / 触发任务 / 补投喂来源 / 状态"},
	{Type: TypeSource, Base: true, Zh: "来源总览（Adler 阶段 0 产物）", PathHint: "source/",
		Requires: []string{}, BodyShape: "来源清单 / 结构-解释-批判-应用四步 / 蒸馏范围"},

	{Type: TypeCommentPattern, Base: false, Zh: "评论范式", PathHint: "pattern/",
		Requires: []string{"voice"}, BodyShape: "R 原文样本 / I 骨架抽象 / A1 触发场景 / A2 迁移 / E 可直接用的句式 / B 禁用边界"},
	{Type: TypeVoicePersona, Base: false, Zh: "声部人格", PathHint: "voice/",
		Requires: []string{"voice"}, BodyShape: "R 语料来源 / I 表达 DNA / A1 该声部怎么开场与收尾 / E 生成参数 / B 越界特征"},
	{Type: TypeTensionAxis, Base: false, Zh: "对立轴", PathHint: "tension/",
		Requires: []string{}, BodyShape: "议题 / A 侧论据 / B 侧论据 / 中立带 / 可引战点 / 合规边界"},
	{Type: TypeModelSpec, Base: false, Zh: "选型参数锚", PathHint: "spec/",
		Requires: []string{}, BodyShape: "参数名 / 常识基线 / 常见误读 / 与竞品差异 / 可验证来源"},
	{Type: TypeMarketClaim, Base: false, Zh: "厂商话术", PathHint: "claim/",
		Requires: []string{}, BodyShape: "原话术 / 事实内核 / 已证实的部分 / 可反驳点 / 反击话术"},
	{Type: TypeRiskRule, Base: false, Zh: "合规红线", PathHint: "risk/",
		Requires: []string{}, BodyShape: "红线 / 触发词 / 法规或平台依据 / 替代表达"},
	{Type: TypeSignal, Base: false, Zh: "舆情信号", PathHint: "signal/",
		Requires: []string{}, BodyShape: "信号 / 观测窗口 / 强度 / 衰减 / 建议动作"},
	{Type: TypeAudienceSegment, Base: false, Zh: "受众切片", PathHint: "audience/",
		Requires: []string{}, BodyShape: "切片 / 决策驱动力 / 关心参数 / 语言习惯 / 常见异议"},
}

var typeIndex = func() map[string]TypeDoc {
	m := make(map[string]TypeDoc, len(TypeDocs))
	for _, d := range TypeDocs {
		m[d.Type] = d
	}
	return m
}()

// IsValidType 报告 t 是否在封闭词表内。
func IsValidType(t string) bool {
	_, ok := typeIndex[t]
	return ok
}

// TypeDocOf 取类型约束，第二个返回值表示是否存在。
func TypeDocOf(t string) (TypeDoc, bool) {
	d, ok := typeIndex[t]
	return d, ok
}

// AllTypes 返回基座 + 领域全部类型。
func AllTypes() []string {
	out := make([]string, 0, len(BaseTypes)+len(DomainTypes))
	out = append(out, BaseTypes...)
	out = append(out, DomainTypes...)
	return out
}

// ---------------------------------------------------------------------------
// 2. 声部枚举（comment voice，封闭）
// ---------------------------------------------------------------------------

// 声部 ID。
const (
	VoiceTiebaLong      = "tieba-long"      // 贴吧高浓度长评
	VoiceHotConsensus   = "hot-consensus"   // 全平台高赞共识
	VoiceBiliGeek       = "bili-geek"       // B站数码爱好者
	VoiceDouyinConflict = "douyin-conflict" // 抖音高对立
	VoiceXhsScene       = "xhs-scene"       // 小红书场景化（扩展声部）
	VoiceZhihuRational  = "zhihu-rational"  // 知乎理性长文（扩展声部）
	VoiceWbSpread       = "wb-spread"       // 微博传播（扩展声部）
)

// VoiceDef 是一个声部的画像。
type VoiceDef struct {
	ID string
	// MCPlatform 是 MediaCrawler 的平台标识（xhs|dy|ks|bili|wb|tieba|zhihu）。
	MCPlatform string
	Zh         string
	// Trait 是该声部的语料特征，决定蒸馏侧重。
	Trait string
	// Focus 是蒸馏时的提取侧重。
	Focus string
	// TargetTension 是该声部期望的对立度中枢（0-1），用于生成时配比。
	TargetTension float64
	// TargetLength 是该声部期望的评论字数中枢。
	TargetLength int
	// Core 标记它是否属于本专家的四个核心声部。
	Core bool
}

// Voices 是声部画像表。
var Voices = []VoiceDef{
	{ID: VoiceTiebaLong, MCPlatform: "tieba", Zh: "贴吧高浓度长评", Core: true,
		Trait: "长段、圈层黑话、强主观、连载式追楼，情绪浓度高但言之有物",
		Focus: "语域与黑话、论证的推进方式、追楼时的自我引用", TargetTension: 0.55, TargetLength: 120},
	{ID: VoiceHotConsensus, MCPlatform: "all", Zh: "全平台高赞共识", Core: true,
		Trait: "已被群体验证的表达，短、断言、金句化，转发率高",
		Focus: "共识框架、金句骨架、为什么这条能拿到高赞", TargetTension: 0.30, TargetLength: 40},
	{ID: VoiceBiliGeek, MCPlatform: "bili", Zh: "B站数码爱好者", Core: true,
		Trait: "参数与实测语汇、技术理性、玩梗、拆解厂商话术",
		Focus: "论据结构、参数精度、梗的用法与尺度", TargetTension: 0.45, TargetLength: 90},
	{ID: VoiceDouyinConflict, MCPlatform: "dy", Zh: "抖音高对立", Core: true,
		Trait: "极短、强冲突、站队明确、口语化，回复区对战",
		Focus: "张力结构、冲突起手、短句节奏", TargetTension: 0.85, TargetLength: 22},
	{ID: VoiceXhsScene, MCPlatform: "xhs", Zh: "小红书场景化", Core: false,
		Trait: "以居家场景和体感切入，弱参数强体验",
		Focus: "场景钩子、体感描述、种草与避雷结构", TargetTension: 0.25, TargetLength: 70},
	{ID: VoiceZhihuRational, MCPlatform: "zhihu", Zh: "知乎理性长文", Core: false,
		Trait: "长论证、引用与限定条件密集、语气克制",
		Focus: "论证链、限定词使用、反例处理", TargetTension: 0.20, TargetLength: 200},
	{ID: VoiceWbSpread, MCPlatform: "wb", Zh: "微博传播", Core: false,
		Trait: "话题标签驱动、转评赞分离、传播语态",
		Focus: "话题挂载、传播句式、情绪引爆点", TargetTension: 0.60, TargetLength: 35},
}

var voiceIndex = func() map[string]VoiceDef {
	m := make(map[string]VoiceDef, len(Voices))
	for _, v := range Voices {
		m[v.ID] = v
	}
	return m
}()

// IsValidVoice 报告 v 是否在封闭声部表内。
func IsValidVoice(v string) bool {
	_, ok := voiceIndex[v]
	return ok
}

// VoiceOf 取声部画像。
func VoiceOf(v string) (VoiceDef, bool) {
	d, ok := voiceIndex[v]
	return d, ok
}

// CoreVoices 返回四个核心声部。
func CoreVoices() []VoiceDef {
	var out []VoiceDef
	for _, v := range Voices {
		if v.Core {
			out = append(out, v)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// 3. 权重口径（M1 侧计算契约）
// ---------------------------------------------------------------------------

// 五因子权重：weight = α·use + β·recency + γ·link + δ·explicit + ε·quality
const (
	AlphaUse      = 0.30 // 使用次数（读取即记账）
	BetaRecency   = 0.20 // 新近度
	GammaLink     = 0.15 // 链接度（Zettelkasten 出入链）
	DeltaExplicit = 0.20 // 显式反馈（人工确认 / 真实互动回流）
	EpsilonQual   = 0.15 // 质量分（蒸馏阶段 4 压力测试得分）
)

// 分层阈值。
const (
	// InitialWeight 概念创建时的初始权重，创建即记一次使用。
	InitialWeight = 0.48
	// PromoteWeight 晋升高权重门槛：weight ≥ 该值且 verified 且稳定。
	PromoteWeight = 0.70
	// DecayIdleDays 长期零使用的天数门槛，超过则降 archive。
	DecayIdleDays = 45
	// LongTierDays 进入 long 层后至少稳定的天数。
	LongTierDays = 7
)

// 领域扩展门禁。
const (
	// HumannessGate 人味门：低于该值不得标 verified，不得用于生成。
	HumannessGate = 0.60
	// RiskGate 合规门：风险分高于该值禁止用于生成，必须先进 Risk Rule 复核。
	RiskGate = 0.40
	// MaxCrawlPerRun 单次扫描条数上限（遵守平台礼貌与体系"宁少勿滥"）。
	MaxCrawlPerRun = 200
	// DefaultTopK 召回默认返回条数。
	DefaultTopK = 12
)

// Tier 取值。
const (
	TierShort   = "short"
	TierLong    = "long"
	TierArchive = "archive"
)

// Verdict 取值。
const (
	VerdictUnverified = "unverified"
	VerdictVerified   = "verified"
)

// IsValidTier 报告 tier 是否合法。
func IsValidTier(t string) bool {
	switch t {
	case TierShort, TierLong, TierArchive:
		return true
	}
	return false
}

// IsValidVerdict 报告 verdict 是否合法。
func IsValidVerdict(v string) bool {
	switch v {
	case VerdictUnverified, VerdictVerified:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// 4. 人味算子表（去 AI 味，封闭）
// ---------------------------------------------------------------------------

// OperatorKind 区分算子的作用方式。
type OperatorKind string

const (
	// OpExcess 计量型：频率越高越像 AI，可算超出人类基线的程度。
	OpExcess OperatorKind = "excess"
	// OpStructural 结构型：需要切句/分段才能判定。
	OpStructural OperatorKind = "structural"
)

// OperatorPer 是算子的分母口径。
type OperatorPer string

const (
	PerKiloChars OperatorPer = "1k_chars"
	PerHundredP  OperatorPer = "100_paras"
)

// Operator 是一条去 AI 味改写算子。
//
// 每条算子的 Trigger 都能落到具体句段与词形——这是收录前提。
// 依赖语义理解（比喻是否贴切、案例是否堆砌）的一律不收录。
type Operator struct {
	ID    int
	Code  string
	Name  string
	Kind  OperatorKind
	Per   OperatorPer
	Ratio float64 // 实测区分力 R = 生成侧频率 ÷ 人类侧频率
	// AIBase / HumanBase 是两个基线的实测频率（单位由 Per 决定）。
	AIBase    float64
	HumanBase float64
	// Source 标明基线来源：research（原文实测）或 calibrated（本专家校准，非原文实测）。
	Source string
	// Trigger 是触发标记的人类可读描述（写进规则文档、给模型看）。
	Trigger string
	// Fix 是改法。
	Fix string
	// Patterns 是正则触发集（用于可定位的特征）。
	Patterns []string
}

// TasteOperators 是去 AI 味规则集，按优先级排序。
//
// 基线数据来自 lieflat-less-ai-tone 的 283 万字对照语料研究
// （629 篇 / 2,826,972 汉字，生成侧 300 篇 / 人类侧 329 篇）。
var TasteOperators = []Operator{
	{ID: 1, Code: "reversal", Name: "翻案腔", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 3.4, AIBase: 0.73, HumanBase: 0.22, Source: "research",
		Trigger: "先立一个读者并没有的误解，再把它推翻：不是A而是B / 并非A而是B / 不在于A而在于B / 与其说A不如说B / 表面A实际B / 看似A实则B / 你以为A其实B / 回头才发现 / 说到底 / 答案恰恰相反",
		Fix:     "直接从正面下判断，先给判断再给依据",
		Patterns: []string{
			`不是[^，。！？；\n]{0,24}而是`,
			`并非[^，。！？；\n]{0,24}而是`,
			`不在于[^，。！？；\n]{0,24}而在于`,
			`与其说[^，。！？；\n]{0,24}不如说`,
			`表面[^，。！？；\n]{0,24}(实际|实则)`,
			`看似[^，。！？；\n]{0,24}(实则|其实)`,
			`你以为[^，。！？；\n]{0,24}其实`,
			`回头才发现`,
			`说到底是`,
			`答案恰恰相反`,
		}},
	{ID: 2, Code: "enum-dunhao", Name: "顿号罗列过密", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 1.8, AIBase: 3.21, HumanBase: 1.78, Source: "research",
		Trigger: "一个分句内出现两个以上顿号，连起三项以上并列成分",
		Fix:     "能概括就别逐项列举；必须保留三项以上时改变其中一项的句法",
		Patterns: []string{
			`[^，。！？；、\s]{1,12}、[^，。！？；、\s]{1,12}、[^，。！？；、\s]{1,12}`,
		}},
	{ID: 3, Code: "adjacent-shape", Name: "相邻句结构同款", Kind: OpStructural, Per: PerHundredP,
		Ratio: 2.0, AIBase: 9.41, HumanBase: 4.81, Source: "research",
		Trigger: "相邻两句以上逗号数量相同、成分顺序相同、长度接近",
		Fix:     "打散其中一句的句法：合并、拆分、换语序或改短句。只改句内，不动段落顺序"},
	{ID: 4, Code: "em-dash", Name: "破折号滥用", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 3.0, AIBase: 2.38, HumanBase: 0.80, Source: "research",
		Trigger:  "用破折号制造停顿后的揭晓，或插入没有必要的补充",
		Fix:      "直接写完整句子，或改用逗号和句号",
		Patterns: []string{`——`, `--`}},
	{ID: 5, Code: "colon-abuse", Name: "冒号滥用", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 3.8, AIBase: 0.58, HumanBase: 0.11, Source: "research",
		Trigger: "提示语引出内容（一句话总结：/ 核心是：/ 关键在于：/ 原因如下：/ 本质上：/ 换句话说：），或空转句只宣布下面有列表而以冒号收尾",
		Fix:     "提示语不承担信息就删掉它，直接写内容；承担衔接则把冒号换成句号或逗号",
		Patterns: []string{
			`(一句话总结|核心是|关键在于|原因如下|结论|本质上|换句话说|说到底是|重点在于|要点如下)[:：]`,
			`(^|\n)[^\n]{0,24}[:：]\s*\n\s*([-*]|\d+[.、])`,
		}},
	{ID: 6, Code: "ordinal-heading", Name: "序数词当小标题", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 3.1, AIBase: 0.19, HumanBase: 0.06, Source: "research",
		Trigger: "小标题（# 开头或独立成行的加粗文本）以「一、二、三」或「第一、第二」编号，且连续三个以上",
		Fix:     "删掉编号，保留小标题原有文字。不改措辞、顺序或层级",
		Patterns: []string{
			`(?m)^#{1,6}\s*(一|二|三|四|五|六|七|八|九|十)、`,
			`(?m)^#{1,6}\s*第[一二三四五六七八九十]+[、.]`,
			`(?m)^\*\*\s*(一|二|三|四|五|六|七|八|九|十)、`,
			`(?m)^\*\*\s*第[一二三四五六七八九十]+[、.]`,
		}},
	{ID: 7, Code: "personified-metaphor", Name: "拟人化喻体", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 7.3, AIBase: 0.018, HumanBase: 0.002, Source: "research",
		Trigger: "像/相当于 + 一个/一位 + 职业角色（导师、秘书、助手、顾问、管家、审查员、实习生…），且喻体带褒义修饰或后接「不仅…更…」",
		Fix:     "换成这东西实际做了什么，或改用一个具体的人当喻体",
		Patterns: []string{
			`(像|相当于)\s*(一个|一位|一名)?\s*(智慧的|全能的|永不疲倦的|永不疲倦、|贴身的|专业的|出色的|顶级的)?\s*(导师|秘书|助手|顾问|管家|审查员|实习生|教练|侦探|翻译官|图书管理员)`,
		}},
	{ID: 8, Code: "abstraction-covers-data", Name: "概括表述盖掉已有数据", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 1.9, AIBase: 0.30, HumanBase: 0.10, Source: "calibrated",
		Trigger: "句中用「显著提升/大幅增长/明显改善/大量的/众多/实现了…提升/完成了对…的/进行了…优化」等概括说法，而同段或相邻段已经写着对应的具体数值、时间或对象",
		Fix:     "把已有的具体值提到概括词的位置。原文没有具体数据时只恢复动词，不许自己编数字",
		Patterns: []string{
			`(显著提升|显著提高|大幅增长|大幅提升|明显改善|明显提升|效率的提升|能力的提升)`,
			`(大量的|众多的|许多的|很多的)`,
			`(实现了[^，。！？；\n]{0,10}(提升|跃升|增长|优化))`,
			`(完成了对[^，。！？；\n]{0,14}的)`,
			`(进行了[^，。！？；\n]{0,10}(优化|调整|升级|改造))`,
		}},
	{ID: 9, Code: "banned-opener", Name: "禁用起手式", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 3.2, AIBase: 0.025, HumanBase: 0.008, Source: "research",
		Trigger: "说白了 / 说穿了 / 先说结论",
		Fix:     "删掉后直接给判断",
		Patterns: []string{
			`(^|\n)\s*(说白了|说穿了|先说结论)[，,、]?`,
			`(说白了|说穿了)[，,]`,
		}},
	{ID: 10, Code: "translationese", Name: "翻译腔（五种）", Kind: OpExcess, Per: PerKiloChars,
		Ratio: 3.0, AIBase: 1.23, HumanBase: 0.41, Source: "research",
		Trigger: "只处理五种：过长前置定语（修饰超十五字或「的」字连用两个以上）／「当…时」前置时间从句／前置话题壳（对于…来说、对…而言、就…而言、关于…、在…方面）／句首连接词当路标（然而、因此、此外、与此同时、换言之、总而言之）／「这意味着」式复述句",
		Fix:     "过长定语在段内拆成两个分句；删掉「当」和「时」；把话题壳的对象放到主语位；连接词移到主语后或换成「不过/其实/也」；复述句并入前一句",
		Patterns: []string{
			`[^` + posBreakers + `]{15,}的[^` + posBreakers + `]{1,8}`,
			`[^的` + posBreakers + `]{1,8}的[^的` + posBreakers + `]{1,8}的[^的` + posBreakers + `]{1,8}的`,
			`当[^，。！？；\n]{2,}时[，,]`,
			`(^|\n|，|。|！|？|；)(对于[^，。！？；\n]{2,}来说|对[^，。！？；\n]{2,}而言|就[^，。！？；\n]{2,}而言|关于[^，。！？；\n]{2,}[，,]|在[^，。！？；\n]{2,}方面)`,
			`(^|\n|。|！|？|；)\s*(然而|因此|此外|与此同时|换言之|总而言之)[，,]`,
			`(这意味着|这表明|这说明|换句话说)`,
		}},
	{ID: 11, Code: "para-opener-no-ref", Name: "段首零回指评论", Kind: OpStructural, Per: PerHundredP,
		Ratio: 4.4, AIBase: 0.61, HumanBase: 0.14, Source: "research",
		Trigger: "非首段以「听起来／看起来／说白了／值得注意的是／更重要的是／关键在于／问题在于／意味着／不难看出」等评论语开头，且整句没有「这／那／其／此／上面」等回指上文的成分",
		Fix:     "补一个回指词，或点明评论对象。多数情况加一个「这」字就够",
		Patterns: []string{
			`^(听起来|看起来|说白了|值得注意的是|更重要的是|关键在于|问题在于|意味着|不难看出)`,
		}},
}

// NonDiscriminating 是不作为改写理由的特征（15 项）。
//
// 这些看着像 AI 味，实测站不住。它们是硬约束：不得据此改文字。
// 其中三项方向与流行认知相反（人类用得更多），删改会让文本更不像人写。
var NonDiscriminating = []struct {
	Feature string
	Why     string
}{
	{"句长、段落长度不够参差", "实测与人类写作无差别（R=0.87 / 0.94）。不要为制造节奏调句长或拆段落"},
	{"单字虚词偏少（就/很/了）", "AI 确实偏少（R=0.45），但方向是补不是删，且补虚词无法验收"},
	{"反复写全称、少用代词", "实测人类比 AI 更常重复同一名词（R=0.50）"},
	{"被动句", "抽象被动（被认为/被视为）频率 0.09，远低于收录门槛"},
	{"名词化、长句本身", "现代汉语书面语正常写法，人类同样这么用（R=0.52）"},
	{"正文里的首先…其次", "与人类写作无差别。第 6 条只管小标题编号"},
	{"句内同构排比", "人类用得不比 AI 少（R=0.61）。只处理句间重复，见第 3 条"},
	{"问句、设问、问句小标题", "正文设问人类是 AI 的 17 倍，删掉更不像人写"},
	{"比喻本身、比喻独立成段", "人类用得比 AI 多（R=0.42，独立成段 8 倍）。只处理理想化拟人喻体，见第 7 条"},
	{"抽象名词配具体动词", "两边都几乎不写（R=0.70）"},
	{"口语连接词偏少", "AI 确实偏少（R=0.26）——这是人味增强项，方向是补"},
	{"序数词充当正文", "句首「首先，」绝对量过低，不构成判据"},
	{"译文句式：以一种…的方式", "实测 0.01/千字，低于门槛"},
	{"译文句式：使得…能够", "实测 0.03/千字，低于门槛"},
	{"译文句式：扮演…角色", "实测 0.02/千字，低于门槛"},
}

// HumanMarkers 是人味增强项：人类侧显著高于生成侧，生成时必须刻意补足。
var HumanMarkers = []struct {
	Feature string
	Ratio   float64
	Note    string
}{
	{"正文设问", 0.05, "人类频率是生成侧的 17 倍。评论区里自问自答是真人最常用的推进方式"},
	{"比喻标记", 0.42, "人类是生成侧的 2.4 倍，独立成段是 8 倍。喻体要取具体的人或物"},
	{"口语连接词", 0.26, "人类是生成侧的约 4 倍。补「其实、反倒、干脆、话说回来」这类"},
	{"单字虚词", 0.45, "人类是生成侧的约 2 倍。评论区里「就、很、了、吧、啊」是真人指纹"},
	{"具体数字密度", 1.0, "人类数字密度是生成侧的 2.8 倍。有实测数据就报出来，别用概括词盖"},
}

// ---------------------------------------------------------------------------
// 5. 合规红线（电视行业舆情专属）
// ---------------------------------------------------------------------------

// RiskPattern 是一条合规触发词。
type RiskPattern struct {
	Code     string
	Zh       string
	Severity float64 // 0-1，累加后与 RiskGate 比较
	Patterns []string
	Fix      string
}

// posBreakers 是「分句边界」字符集合，供需判断跨分句结构的算子使用（如翻译腔的长定语）。
//
// 必须包含**全角/半角冒号**与引号、括号：冒号之后是新分句，
// 冒号前的文字并不修饰其后的名词。漏掉 `：` 会让
// 「我的建议就一句：下单前先问清楚送的架子」被判成「过长前置定语」——
// 这是中文长句里最高频的假阳性来源。
const posBreakers = `，。！？；、：:""「」『』（）()…—\s`

// RiskPatterns 是内容生成前的合规预检表。
//
// 依据：《广告法》第九条绝对化用语禁止、第十三条贬低同行禁止，
// 以及各平台社区规范对拉踩引战的限制。
var RiskPatterns = []RiskPattern{
	{Code: "absolute-claim", Zh: "绝对化用语", Severity: 0.55,
		// 反例（为何必须收窄）：
		//   `第一` 裸用会误伤序数（第一步 / 第一引用源 / 第一优先）；
		//   `唯一` 裸用会误伤事实陈述（唯一能验证的是…）；
		//   `绝对` 裸用会误伤术语与统计（绝对化用语 / 声量绝对值 / 绝对量最大）。
		// 广告法针对的是**等级断言**，故一律收窄到断言形态。
		Patterns: []string{
			`(最好|最强|最佳|第[一1](?:名|品牌|梯队|阵营|销量|爆款)|唯一(?:的)?(?:选择|品牌|方案|官方|渠道)|顶级|国家级|世界级|最高端|绝对(?:领先|第一|优势|保证|安全|可靠|不会|不)|百分之百|全网最低)`,
		},
		Fix: "换成可验证的相对表述：在同价位里 / 我这台测下来 / 目前看到的评测里"},
	{Code: "defame-rival", Zh: "贬损竞品", Severity: 0.50,
		Patterns: []string{`(垃圾|智商税|割韭菜|骗钱|破烂|别买|翻车|智商检测机|偷工减料)`},
		Fix:      "只评价可测的事实，不做人格或动机指控；改说「这个价位的取舍我不接受」"},
	{Code: "unverified-param", Zh: "未证实参数", Severity: 0.40,
		Patterns: []string{`(实测峰值|官方宣称\d|民间传说|据说.{0,6}(分区|尼特|赫兹))`},
		Fix:      "标注来源与测量条件；拿不到来源就删掉数字，只留主观体感"},
	{Code: "medical-eye", Zh: "护眼功效断言", Severity: 0.60,
		Patterns: []string{`(护眼|不伤眼|治疗|缓解近视|保护视力).{0,10}(保证|绝对|一定|彻底)`},
		Fix:      "改为体验描述：低亮度下看久了眼睛没那么累"},
}

// metaLineMarkers 标记「正在讨论/约束某表达」而非「正在使用该表达」的行。
//
// 合规规则表、禁用词清单、越界特征表这类**元讨论**文本，
// 必然包含它们所要约束的那些词。若不剔除，规矩自己就会判自己违规 ——
// 这是知识库落库时最常见的假阳性来源。
var metaLineMarkers = []string{
	"禁用", "禁止", "不得", "不可", "红线", "越界", "拦截", "违禁",
	"替代", "误用", "避免", "反例", "示例", "检", "改写", "算子", "命中",
	"触发", "用语", "表述", "词表", "清单", "类别", "类：", "类（", "级：",
}

// quotedSpans 匹配需要剔除的引注片段（引用他人措辞不等于自己主张）。
// 置为包级切片便于整篇（跨行）剔除，避免逐行处理时漏掉跨行引注。
var quotedSpans = []string{
	`「[^」]*」`, `『[^』]*』`, `“[^”]*”`, `"[^"]*"`, `（例[：:][^）]*）`,
}

// stripForRiskScan 生成「合规预检」用的干净文本：
//  1. 整篇剔除引注片段（「有人说这是智商税」里的引语不算主张）；
//  2. 剔除元讨论行（禁用词表 / 越界特征 / 替代表达等）；
//  3. 剔除纯词条清单行（顿号密集的短词列举 = 词表，不是用法）。
//
// 该预处理只服务于 RiskScan；原文一字不改。
func stripForRiskScan(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if hasAnyMarker(line) || looksLikeTokenList(line) {
			b.WriteByte('\n')
			continue
		}
		b.WriteString(stripQuoted(line))
		b.WriteByte('\n')
	}
	return b.String()
}

func hasAnyMarker(line string) bool {
	for _, m := range metaLineMarkers {
		if strings.Contains(line, m) {
			return true
		}
	}
	return false
}

func stripQuoted(line string) string {
	for _, p := range quotedSpans {
		re, err := Compile(p)
		if err != nil {
			continue
		}
		line = re.ReplaceAllString(line, "")
	}
	return line
}

// looksLikeTokenList 判定一行是否为「词条清单」。
//
// 判据：以顿号/逗号切分后条目数 ≥3，且（平均长度 ≤5 或最长条目 ≤8 字符）、
// 且句末无句号/感叹号/问号。
// 例：「乱收费、欺诈、虚假宣传、智商税、造假、虚标」→ 清单。
// 反例：「这个价位的取舍我不接受，最好别买。」→ 有条目但带句末标点，是主张。
func looksLikeTokenList(line string) bool {
	t := strings.TrimSpace(line)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "#") {
		return false
	}
	// 表格行与引用块：结构化数据 / 他人原话，单元格常为词条而非主张。
	if strings.HasPrefix(t, "|") || strings.HasPrefix(t, ">") {
		return true
	}
	if strings.ContainsAny(t, "。！？") {
		return false
	}
	items := splitTokenItems(t)
	if len(items) < 3 {
		return false
	}
	total, maxLen := 0, 0
	for _, it := range items {
		n := len([]rune(it))
		total += n
		if n > maxLen {
			maxLen = n
		}
	}
	return float64(total)/float64(len(items)) <= 5.0 || maxLen <= 8
}

func splitTokenItems(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool {
		switch r {
		case '、', '，', ',', ';', '；', '|', '/':
			return true
		}
		return false
	})
	var out []string
	for _, f := range fields {
		f = strings.TrimSpace(strings.Trim(f, "*`（）()：: "))
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

// RiskScan 返回命中的红线编码与累计风险分。
//
// 先做 stripForRiskScan 预处理，再逐条匹配 —— 见该函数的注释说明为何必须剥离。
func RiskScan(text string) ([]string, float64) {
	return riskScanRaw(stripForRiskScan(text))
}

// riskScanRaw 是不经预处理的裸扫描，供需要最严口径的场景（如生成稿逐句自查）使用。
func riskScanRaw(text string) ([]string, float64) {
	var hits []string
	var score float64
	for _, rp := range RiskPatterns {
		matched := false
		for _, p := range rp.Patterns {
			if regexpMatch(p, text) {
				matched = true
				break
			}
		}
		if matched {
			hits = append(hits, rp.Code)
			score += rp.Severity
		}
	}
	if score > 1 {
		score = 1
	}
	return hits, score
}

// JoinTypes 便于打印词表。
func JoinTypes(sep string) string { return strings.Join(AllTypes(), sep) }
