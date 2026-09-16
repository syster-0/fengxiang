// Package server 提供 M1 的两个服务出口：HTTP 与 MCP。
//
// 出口的 schema 就是运行时契约：不在词表里的工具一律拒调，
// 禁止臆造工具名、参数名或返回字段。
package server

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sycglier/tv-opinion-atelier/internal/crawl"
	"github.com/sycglier/tv-opinion-atelier/internal/govern"
	"github.com/sycglier/tv-opinion-atelier/internal/okf"
	"github.com/sycglier/tv-opinion-atelier/internal/spec"
	"github.com/sycglier/tv-opinion-atelier/internal/taste"
)

// ListenAndServe 启动 HTTP 出口。
func ListenAndServe(addr string, w *okf.Wiki) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           NewHTTP(w),
		ReadHeaderTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

// ---------------------------------------------------------------------------
// HTTP 出口
// ---------------------------------------------------------------------------

// NewHTTP 返回 HTTP 处理器。
func NewHTTP(w *okf.Wiki) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(rw http.ResponseWriter, _ *http.Request) {
		writeJSON(rw, 200, map[string]any{"ok": true, "wiki": w.Root, "version": "tvop/1.0"})
	})

	mux.HandleFunc("/api/spec", func(rw http.ResponseWriter, _ *http.Request) {
		writeJSON(rw, 200, specDump())
	})

	mux.HandleFunc("/api/concepts", func(rw http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		terms := splitCSV(q.Get("q"))
		top, _ := strconv.Atoi(q.Get("top"))
		hits, err := w.Search(okf.Query{
			Terms:       terms,
			Type:        q.Get("type"),
			Voice:       q.Get("voice"),
			OnlyUsable:  q.Get("usable") == "1" || q.Get("usable") == "true",
			IncludeArch: q.Get("arch") == "1",
			Top:         top,
		})
		if err != nil {
			writeErr(rw, err)
			return
		}
		out := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			out = append(out, hitJSON(h))
		}
		writeJSON(rw, 200, map[string]any{"count": len(out), "hits": out})
	})

	mux.HandleFunc("/api/concepts/", func(rw http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/concepts/")
		if id == "" {
			writeJSON(rw, 400, map[string]any{"error": "缺概念 ID"})
			return
		}
		c, err := w.Get(id)
		if err != nil {
			writeErr(rw, err)
			return
		}
		writeJSON(rw, 200, map[string]any{
			"id": c.ID, "type": c.Type, "title": c.Title, "voice": c.Voice,
			"platform": c.Platform, "atelier": c.Atelier, "usable": c.Usable(),
			"unusable_because": c.UnusableBecause(), "rank": c.Rank(), "body": c.Body,
		})
	})

	mux.HandleFunc("/api/taste/score", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(rw, 405, map[string]any{"error": "只接受 POST"})
			return
		}
		var in struct {
			Text string `json:"text"`
		}
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err := json.Unmarshal(body, &in); err != nil {
			writeJSON(rw, 400, map[string]any{"error": "请求体必须是 {\"text\": \"...\"}"})
			return
		}
		writeJSON(rw, 200, tasteJSON(taste.Score(in.Text)))
	})

	mux.HandleFunc("/api/validate", func(rw http.ResponseWriter, _ *http.Request) {
		rep, err := w.Validate()
		if err != nil {
			writeErr(rw, err)
			return
		}
		writeJSON(rw, 200, reportJSON(rep))
	})

	mux.HandleFunc("/api/stats", func(rw http.ResponseWriter, _ *http.Request) {
		st, err := govern.StatsOf(w)
		if err != nil {
			writeErr(rw, err)
			return
		}
		writeJSON(rw, 200, map[string]any{
			"concepts": st.Concepts, "usable": st.Usable,
			"mean_weight": st.MeanWeight, "mean_humanness": st.MeanHumanness, "mean_risk": st.MeanRisk,
			"by_tier": st.ByTier, "by_verdict": st.ByVerdict, "by_voice": st.ByVoice, "by_type": st.ByType,
			"humanness_hist": st.HumannessHist, "unusable": st.Unusable,
			"feed_plan": govern.FeedPlan(st),
		})
	})

	mux.HandleFunc("/api/rules", func(rw http.ResponseWriter, _ *http.Request) {
		writeJSON(rw, 200, map[string]any{
			"operators":          spec.TasteOperators,
			"non_discriminating": spec.NonDiscriminating,
			"human_markers":      spec.HumanMarkers,
			"risk_patterns":      spec.RiskPatterns,
			"humanness_gate":     spec.HumannessGate,
			"risk_gate":          spec.RiskGate,
		})
	})

	return mux
}

func writeJSON(rw http.ResponseWriter, code int, v any) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")
	rw.WriteHeader(code)
	enc := json.NewEncoder(rw)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func writeErr(rw http.ResponseWriter, err error) {
	writeJSON(rw, 400, map[string]any{"error": err.Error()})
}

func hitJSON(h okf.Hit) map[string]any {
	return map[string]any{
		"id": h.Concept.ID, "type": h.Concept.Type, "title": h.Concept.Title,
		"voice": h.Concept.Voice, "tier": h.Concept.Atelier.Tier,
		"verdict": h.Concept.Atelier.Verdict, "weight": h.Concept.Atelier.Weight,
		"humanness": h.Concept.Atelier.Humanness, "risk": h.Concept.Atelier.Risk,
		"rank": h.Rank, "score": h.Score, "fields": h.Fields, "snippet": h.Snippet,
		"usable": h.Concept.Usable(),
	}
}

func reportJSON(rep *okf.Report) map[string]any {
	issues := make([]map[string]any, 0, len(rep.Issues))
	for _, i := range rep.Issues {
		issues = append(issues, map[string]any{
			"severity": i.Severity, "code": i.Code, "subject": i.Subject, "msg": i.Msg, "fix": i.Fix,
		})
	}
	return map[string]any{
		"valid": rep.Valid(), "errors": rep.Errors(), "warnings": rep.Warnings(),
		"concepts": rep.Concepts, "verified": rep.Verified, "usable": rep.Usable,
		"issues": issues,
	}
}

func tasteJSON(rep taste.Report) map[string]any {
	hits := make([]map[string]any, 0, len(rep.Hits))
	for _, h := range rep.Hits {
		hits = append(hits, map[string]any{
			"id": h.Op.ID, "code": h.Op.Code, "name": h.Op.Name,
			"count": h.Count, "rate": h.Rate, "excess": h.Excess, "weight": h.Weight,
			"per": h.Op.Per, "source": h.Op.Source, "examples": h.Examples, "fix": h.Op.Fix,
		})
	}
	markers := make([]map[string]any, 0, len(rep.Markers))
	for _, m := range rep.Markers {
		markers = append(markers, map[string]any{
			"feature": m.Feature, "count": m.Count, "rate": m.Rate,
			"baseline": m.Baseline, "deficit": m.Deficit, "note": m.Note,
		})
	}
	return map[string]any{
		"chars": rep.Chars, "paras": rep.Paras, "sentences": rep.Sentences,
		"ai_tone": rep.AITone, "humanness": rep.Humanness, "gate": rep.Gate,
		"gate_threshold": spec.HumannessGate,
		"risk":           rep.Risk, "risk_hits": rep.RiskHits, "risk_gate": spec.RiskGate,
		"hits": hits, "markers": markers, "hints": rep.Hints(), "notes": rep.Notes,
	}
}

func specDump() map[string]any {
	types := make([]map[string]any, 0, len(spec.TypeDocs))
	for _, d := range spec.TypeDocs {
		types = append(types, map[string]any{
			"type": d.Type, "base": d.Base, "zh": d.Zh,
			"path_hint": d.PathHint, "requires": d.Requires, "body_shape": d.BodyShape,
		})
	}
	voices := make([]map[string]any, 0, len(spec.Voices))
	for _, v := range spec.Voices {
		voices = append(voices, map[string]any{
			"id": v.ID, "mc_platform": v.MCPlatform, "zh": v.Zh,
			"trait": v.Trait, "focus": v.Focus,
			"target_tension": v.TargetTension, "target_length": v.TargetLength, "core": v.Core,
		})
	}
	return map[string]any{
		"types": types, "voices": voices,
		"platforms": crawl.AllPlatforms,
		"weights": map[string]float64{
			"alpha_use": spec.AlphaUse, "beta_recency": spec.BetaRecency,
			"gamma_link": spec.GammaLink, "delta_explicit": spec.DeltaExplicit,
			"epsilon_quality": spec.EpsilonQual, "initial_weight": spec.InitialWeight,
		},
		"tiers":    []string{spec.TierShort, spec.TierLong, spec.TierArchive},
		"verdicts": []string{spec.VerdictUnverified, spec.VerdictVerified},
	}
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
