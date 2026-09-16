package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/crawl"
	"github.com/sycglier/tv-opinion-atelier/internal/errx"
	"github.com/sycglier/tv-opinion-atelier/internal/govern"
	"github.com/sycglier/tv-opinion-atelier/internal/okf"
	"github.com/sycglier/tv-opinion-atelier/internal/spec"
	"github.com/sycglier/tv-opinion-atelier/internal/taste"
)

// mcpRequest 是 JSON-RPC 2.0 请求。
type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *mcpError       `json:"error,omitempty"`
}

type mcpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// toolDef 是一个 MCP 工具定义。schema 即运行时契约。
type toolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

// MCPTools 是封闭的工具词表。
var MCPTools = []toolDef{
	{
		Name: "tvop_search",
		Description: "按召回协议检索 OKF 知识库。返回条数按 命中 × weight × tier × 人味 排序。" +
			"仅返回 usable=true 的概念才能用于生成评论。",
		InputSchema: obj(map[string]any{
			"terms":        arr("关键词列表，任一命中即入选，全部命中加权更高", str()),
			"voice":        strEnum("限定声部", voiceIDs()),
			"type":         strEnum("限定概念类型", spec.AllTypes()),
			"top":          num("返回条数上限，默认 12"),
			"usable_only":  boolean("只返回可用于生成的概念"),
			"include_arch": boolean("是否包含 archive 层的概念"),
		}, "terms"),
	},
	{
		Name:        "tvop_get",
		Description: "按概念 ID 读取完整概念（含 RIA++ 正文、atelier 元数据、可用性判定）。",
		InputSchema: obj(map[string]any{"id": str()}, "id"),
	},
	{
		Name:        "tvop_validate",
		Description: "跑全库一致性校验门。落库与交付前必须全绿。",
		InputSchema: obj(map[string]any{}),
	},
	{
		Name: "tvop_taste_score",
		Description: "对文本跑去 AI 味算子，返回人味分（0-1）、逐条命中、人味增强项缺口与合规风险。" +
			"人味分低于 " + ftoa(spec.HumannessGate) + " 即未过门，必须回炉改写。",
		InputSchema: obj(map[string]any{"text": str()}, "text"),
	},
	{
		Name:        "tvop_gaps",
		Description: "检测知识缺口：某关键词全库零命中，或命中的概念全部不可用于生成。",
		InputSchema: obj(map[string]any{"terms": arr("要检查的要素", str()), "voice": strEnum("限定声部", voiceIDs())}, "terms"),
	},
	{
		Name:        "tvop_rules",
		Description: "返回去 AI 味 11 条规则、15 项不作为改写理由的特征、人味增强项与合规红线。",
		InputSchema: obj(map[string]any{}),
	},
	{
		Name:        "tvop_crawl_plan",
		Description: "生成 MediaCrawler 爬取计划（不执行）。执行需人工确认——爬取是有副作用的外部动作。",
		InputSchema: obj(map[string]any{
			"platform":  strEnum("平台", crawl.AllPlatforms),
			"keywords":  arr("关键词", str()),
			"voice":     strEnum("覆盖声部（可选）", voiceIDs()),
			"max_notes": num("最多抓取条目数，上限 200"),
		}, "platform", "keywords"),
	},
	{
		Name:        "tvop_stats",
		Description: "权重分析：概念数、可用数、均权重、均人味、tier/verdict/声部分布、人味分直方图与补投喂清单。",
		InputSchema: obj(map[string]any{}),
	},
}

// ServeMCP 在 stdio 上跑 MCP 服务，直到输入结束。
func ServeMCP(w *okf.Wiki, in io.Reader, out io.Writer) error {
	r := bufio.NewReaderSize(in, 1<<20)
	enc := json.NewEncoder(out)
	for {
		line, err := r.ReadBytes('\n')
		if len(strings.TrimSpace(string(line))) > 0 {
			var req mcpRequest
			if jerr := json.Unmarshal(line, &req); jerr != nil {
				_ = enc.Encode(mcpResponse{JSONRPC: "2.0", Error: &mcpError{Code: -32700, Message: "解析失败: " + jerr.Error()}})
			} else {
				resp, reply := handleMCP(w, req)
				if reply {
					if eerr := enc.Encode(resp); eerr != nil {
						return eerr
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func handleMCP(w *okf.Wiki, req mcpRequest) (mcpResponse, bool) {
	ok := func(v any) (mcpResponse, bool) {
		return mcpResponse{JSONRPC: "2.0", ID: req.ID, Result: v}, true
	}
	fail := func(code int, msg string) (mcpResponse, bool) {
		return mcpResponse{JSONRPC: "2.0", ID: req.ID, Error: &mcpError{Code: code, Message: msg}}, true
	}
	silent := func() (mcpResponse, bool) { return mcpResponse{}, false }

	switch req.Method {
	case "initialize":
		return ok(map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "tvop", "version": "1.0.0"},
			"instructions": "电视行业舆情运营知识库。生成任何评论前先 tvop_search 召回，再 tvop_taste_score 打分；" +
				"人味分低于门槛的内容禁止交付。",
		})
	case "notifications/initialized", "initialized", "notifications/cancelled":
		return silent()
	case "ping":
		return ok(map[string]any{})
	case "tools/list":
		return ok(map[string]any{"tools": MCPTools})
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &p); err != nil {
			return fail(-32602, "params 非法: "+err.Error())
		}
		res, err := callTool(w, p.Name, p.Arguments)
		if err != nil {
			return ok(map[string]any{
				"content": []map[string]any{{"type": "text", "text": err.Error()}},
				"isError": true,
			})
		}
		b, _ := json.MarshalIndent(res, "", "  ")
		return ok(map[string]any{
			"content": []map[string]any{{"type": "text", "text": string(b)}},
			"isError": false,
		})
	default:
		return fail(-32601, "未实现的方法: "+req.Method)
	}
}

// callTool 按封闭词表分派。未知工具直接拒绝——禁止臆造工具名。
func callTool(w *okf.Wiki, name string, args json.RawMessage) (any, error) {
	known := false
	for _, t := range MCPTools {
		if t.Name == name {
			known = true
			break
		}
	}
	if !known {
		return nil, errx.Errorf(errx.KindInvalid, "mcp.call", name,
			"工具不在词表内。可用：%s", toolNames())
	}

	switch name {
	case "tvop_search":
		var a struct {
			Terms       []string `json:"terms"`
			Voice       string   `json:"voice"`
			Type        string   `json:"type"`
			Top         int      `json:"top"`
			UsableOnly  bool     `json:"usable_only"`
			IncludeArch bool     `json:"include_arch"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, errx.Wrap(errx.KindInvalid, "mcp.search", "", err)
		}
		hits, err := w.Search(okf.Query{Terms: a.Terms, Voice: a.Voice, Type: a.Type,
			Top: a.Top, OnlyUsable: a.UsableOnly, IncludeArch: a.IncludeArch})
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			out = append(out, hitJSON(h))
		}
		return map[string]any{"count": len(out), "hits": out}, nil

	case "tvop_get":
		var a struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, errx.Wrap(errx.KindInvalid, "mcp.get", "", err)
		}
		c, err := w.Get(a.ID)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"id": c.ID, "type": c.Type, "title": c.Title, "voice": c.Voice, "platform": c.Platform,
			"atelier": c.Atelier, "usable": c.Usable(), "unusable_because": c.UnusableBecause(),
			"rank": c.Rank(), "body": c.Body,
		}, nil

	case "tvop_validate":
		rep, err := w.Validate()
		if err != nil {
			return nil, err
		}
		return reportJSON(rep), nil

	case "tvop_taste_score":
		var a struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, errx.Wrap(errx.KindInvalid, "mcp.taste", "", err)
		}
		if strings.TrimSpace(a.Text) == "" {
			return nil, errx.New(errx.KindInvalid, "mcp.taste", "", "text 不能为空")
		}
		return tasteJSON(taste.Score(a.Text)), nil

	case "tvop_gaps":
		var a struct {
			Terms []string `json:"terms"`
			Voice string   `json:"voice"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, errx.Wrap(errx.KindInvalid, "mcp.gaps", "", err)
		}
		gaps, err := w.DetectGaps(a.Terms, a.Voice)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(gaps))
		for _, g := range gaps {
			out = append(out, map[string]any{"term": g.Term, "reason": g.Reason, "suggest": g.Suggest})
		}
		return map[string]any{"count": len(out), "gaps": out}, nil

	case "tvop_rules":
		return map[string]any{
			"operators":          spec.TasteOperators,
			"non_discriminating": spec.NonDiscriminating,
			"human_markers":      spec.HumanMarkers,
			"risk_patterns":      spec.RiskPatterns,
			"humanness_gate":     spec.HumannessGate,
			"risk_gate":          spec.RiskGate,
		}, nil

	case "tvop_crawl_plan":
		var a struct {
			Platform string   `json:"platform"`
			Keywords []string `json:"keywords"`
			Voice    string   `json:"voice"`
			MaxNotes int      `json:"max_notes"`
		}
		if err := json.Unmarshal(args, &a); err != nil {
			return nil, errx.Wrap(errx.KindInvalid, "mcp.crawl_plan", "", err)
		}
		root := crawlProjectRoot(w)
		if root == "" {
			return nil, errx.New(errx.KindNotFound, "mcp.crawl_plan", "",
				"未找到 MediaCrawler 仓库根。请把 MediaCrawler 克隆到仓库根的 third_party/MediaCrawler，"+
					"或设置环境变量 MEDIACRAWLER_ROOT")
		}
		plan, err := crawl.BuildPlan(root, crawl.Options{
			Platform: a.Platform, Keywords: a.Keywords, Voice: a.Voice,
			MaxNotes: a.MaxNotes, GetComment: true,
		})
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"platform": plan.Options.Platform, "voice": plan.Voice,
			"project_root": plan.ProjectRoot, "workdir": plan.WorkDir,
			"argv": plan.Command(), "shell": plan.ShellLine(),
			"estimate_comments": plan.Estimate,
			"confirm_required":  true,
			"note":              "计划已生成但未执行。爬取消耗账号额度并可能触发平台风控，需人工确认后运行。",
		}, nil

	case "tvop_stats":
		st, err := govern.StatsOf(w)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"concepts": st.Concepts, "usable": st.Usable,
			"mean_weight": st.MeanWeight, "mean_humanness": st.MeanHumanness, "mean_risk": st.MeanRisk,
			"by_tier": st.ByTier, "by_verdict": st.ByVerdict, "by_voice": st.ByVoice, "by_type": st.ByType,
			"humanness_hist": st.HumannessHist, "unusable": st.Unusable,
			"feed_plan": govern.FeedPlan(st),
			"summary":   st.Line() + " · " + st.HumannessLine(),
		}, nil
	}
	return nil, errx.Errorf(errx.KindInternal, "mcp.call", name, "工具已声明但未实现")
}

// crawlProjectRoot 依次尝试：环境变量、仓库根 third_party/MediaCrawler、仓库根同级。
func crawlProjectRoot(w *okf.Wiki) string {
	if v := strings.TrimSpace(os.Getenv("MEDIACRAWLER_ROOT")); v != "" {
		if r, ok := crawl.DetectProjectRoot(v); ok {
			return r
		}
	}
	repo := w.RepoRoot()
	for _, c := range []string{
		filepath.Join(repo, "third_party", "MediaCrawler"),
		filepath.Join(repo, "MediaCrawler"),
		filepath.Join(repo, "..", "MediaCrawler"),
	} {
		if r, ok := crawl.DetectProjectRoot(c); ok {
			return r
		}
	}
	return ""
}

func toolNames() string {
	var n []string
	for _, t := range MCPTools {
		n = append(n, t.Name)
	}
	return strings.Join(n, " / ")
}

func voiceIDs() []string {
	out := make([]string, 0, len(spec.Voices))
	for _, v := range spec.Voices {
		out = append(out, v.ID)
	}
	return out
}

// ---------------------------------------------------------------------------
// JSON Schema 小构造器（保持 schema 与代码同源）
// ---------------------------------------------------------------------------

func obj(props map[string]any, required ...string) map[string]any {
	m := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
	if len(required) > 0 {
		m["required"] = required
	}
	return m
}

func str() map[string]any { return map[string]any{"type": "string"} }

func strEnum(desc string, values []string) map[string]any {
	return map[string]any{"type": "string", "description": desc, "enum": values}
}

func num(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func boolean(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func arr(desc string, items map[string]any) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": items}
}

func ftoa(f float64) string { return fmt.Sprintf("%.2f", f) }
