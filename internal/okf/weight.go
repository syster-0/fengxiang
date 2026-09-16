package okf

import (
	"math"

	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// Round3 保留三位小数。
func Round3(f float64) float64 { return math.Round(f*1000) / 1000 }

// Clamp01 把值夹到 [0,1]。
func Clamp01(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// WeightInput 是五因子权重计算的输入。
type WeightInput struct {
	UseCount     int     // 读取即记账的累计次数（创建即 1）
	DaysSinceUse float64 // 距最近一次使用/写入的天数
	InLinks      int     // 被别的概念链接（入链）
	OutLinks     int     // 链出去的概念数（出链）
	Explicit     float64 // 显式反馈 0-1：人工确认 / 发布后真实互动回流
	Quality      float64 // 质量分 0-1：蒸馏阶段 4 压力测试得分
}

// NewConceptWeightInput 返回概念创建时的权重输入。
//
// 取值使 ComputeWeight 恰好得到 spec.InitialWeight（0.48）：
// use 1/4 = 0.25 → 0.075；recency 1.0 → 0.200；link 0 → 0；
// explicit 0.50 → 0.100；quality 0.70 → 0.105；合计 0.480。
func NewConceptWeightInput() WeightInput {
	return WeightInput{
		UseCount:     1,
		DaysSinceUse: 0,
		Explicit:     0.50,
		Quality:      0.70,
	}
}

// 饱和度参数：读满 4 次即满分；45 天半衰；6 条链接即满分。
const (
	useSaturation       = 4.0
	recencyHalfLifeDays = 45.0
	linkSaturation      = 6.0
)

// ComputeWeight 计算五因子权重：
//
//	weight = α·use + β·recency + γ·link + δ·explicit + ε·quality
//
// 各因子先归一到 [0,1]，故 weight ∈ [0,1]。
func ComputeWeight(in WeightInput) float64 {
	use := Clamp01(float64(in.UseCount) / useSaturation)
	recency := math.Exp(-math.Max(0, in.DaysSinceUse) / recencyHalfLifeDays)
	link := Clamp01(float64(in.InLinks+in.OutLinks) / linkSaturation)
	explicit := Clamp01(in.Explicit)
	quality := Clamp01(in.Quality)

	w := spec.AlphaUse*use +
		spec.BetaRecency*recency +
		spec.GammaLink*link +
		spec.DeltaExplicit*explicit +
		spec.EpsilonQual*quality
	return Round3(Clamp01(w))
}

// DecayRecency 返回按新近度衰减后的权重（治理 job 用，不改 use_count）。
func DecayRecency(w float64, daysSinceUse float64) float64 {
	base := NewConceptWeightInput()
	base.DaysSinceUse = daysSinceUse
	// 用当前权重反推不出因子，故按新近度项单独衰减：
	recency0 := math.Exp(0)
	recency1 := math.Exp(-math.Max(0, daysSinceUse) / recencyHalfLifeDays)
	delta := spec.BetaRecency * (recency1 - recency0)
	return Round3(Clamp01(w + delta))
}

// Rank 是生成排序分：知识价值 × 人味 × 共鸣 × 张力，再乘合规惩罚。
//
//	rank = weight × (0.5 + 0.5·humanness) × (1 + 0.4·resonance) × (1 + 0.2·tension) × penalty
//
// 人味用 (0.5+0.5·h) 而非直接乘，是为了让未打分的概念（h=0）只衰减到一半，
// 不至于因为尚未跑算子而被完全埋掉。
func Rank(a Atelier) float64 {
	penalty := 1.0
	if a.Risk > spec.RiskGate {
		penalty = 0.15
	}
	r := a.Weight *
		(0.5 + 0.5*Clamp01(a.Humanness)) *
		(1 + 0.4*Clamp01(a.Resonance)) *
		(1 + 0.2*Clamp01(a.Tension)) *
		penalty
	return Round3(r)
}
