// Command tvop 是 tv-opinion-atelier 的 M1 命令行出口。
//
// 子命令：
//
//	init        初始化 wiki 骨架
//	doctor      环境自检（Go/Python/MediaCrawler/wiki 状态）
//	spec        打印运行时契约（词表 / 声部 / 权重 / 规则 / 平台）
//	okf         知识库操作（new/get/list/reindex/validate/sourcemap/test/account）
//	search      召回（命中 × weight × tier × 人味 排序）
//	gaps        知识缺口检测
//	taste       去 AI 味打分与规则查询
//	weight      权重解释
//	govern      治理 job（audit/promote/decay/recall/recompute/report）
//	crawl       MediaCrawler 适配（platforms/plan/run/ingest/select/feed）
//	distill     RIA-TV++ 蒸馏计划（plan）
//	serve       HTTP 服务
//	mcp         MCP 服务（stdio）
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sycglier/tv-opinion-atelier/internal/crawl"
	"github.com/sycglier/tv-opinion-atelier/internal/errx"
	"github.com/sycglier/tv-opinion-atelier/internal/govern"
	"github.com/sycglier/tv-opinion-atelier/internal/okf"
	"github.com/sycglier/tv-opinion-atelier/internal/server"
	"github.com/sycglier/tv-opinion-atelier/internal/spec"
	"github.com/sycglier/tv-opinion-atelier/internal/taste"
)

const (
	exitOK      = 0
	exitFail    = 1
	exitInvalid = 2
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return exitInvalid
	}
	cmd, rest := args[0], args[1:]
	switch cmd {
	case "-h", "--help", "help":
		usage()
		return exitOK
	case "version", "--version":
		fmt.Println("tvop 1.0.0")
		return exitOK
	case "init":
		return cmdInit(rest)
	case "doctor":
		return cmdDoctor(rest)
	case "spec":
		return cmdSpec(rest)
	case "okf":
		return cmdOKF(rest)
	case "search":
		return cmdSearch(rest)
	case "gaps":
		return cmdGaps(rest)
	case "taste":
		return cmdTaste(rest)
	case "weight":
		return cmdWeight(rest)
	case "govern":
		return cmdGovern(rest)
	case "crawl":
		return cmdCrawl(rest)
	case "repo":
		return cmdRepo(rest)
	case "distill":
		return cmdDistill(rest)
	case "serve":
		return cmdServe(rest)
	case "mcp":
		return cmdMCP(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知子命令：%s\n\n", cmd)
		usage()
		return exitInvalid
	}
}

func usage() {
	fmt.Print(`tvop — 电视行业舆情运营知识库 CLI（M1 架构层出口）

用法：tvop <子命令> [参数]

  init      初始化 wiki 骨架（幂等）
  doctor    环境自检
  spec [types|voices|weights|rules|platforms]
  okf <new|get|list|reindex|validate|sourcemap|test|account>
  search <关键词...>      召回（命中 × weight × tier × 人味）
  gaps <要素...>          知识缺口检测
  taste <score|rules>     去 AI 味打分 / 规则表
  weight explain <概念 ID>  权重因子拆解
  govern <audit|promote|decay|recall|recompute|report>
  crawl <platforms|plan|run|ingest|select|feed|tension>
  repo <status|pull>      分发自感知：检查远端更新 / 安全拉取（ff-only）
  distill plan            生成 RIA-TV++ 蒸馏计划
  serve [--addr :8080]    HTTP 服务
  mcp                     MCP 服务（stdio）

全局环境变量：
  TVOP_WIKI            指定 wiki 目录（默认 <仓库根>/wiki）
  MEDIACRAWLER_ROOT    指定 MediaCrawler 仓库根
`)
}

// ---------------------------------------------------------------------------
// wiki 定位
// ---------------------------------------------------------------------------

func wikiRoot() string {
	if v := strings.TrimSpace(os.Getenv("TVOP_WIKI")); v != "" {
		return v
	}
	if wd, err := os.Getwd(); err == nil {
		if p := filepath.Join(wd, "wiki"); isDir(p) {
			return p
		}
	}
	exe, err := os.Executable()
	if err == nil {
		if r, ok := findMarkedRoot(filepath.Dir(exe)); ok {
			return filepath.Join(r, "wiki")
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return filepath.Join(wd, "wiki")
	}
	return "wiki"
}

// findMarkedRoot 从 dir 向上找 go.mod 作为定位锚点。
func findMarkedRoot(dir string) (string, bool) {
	dir, _ = filepath.Abs(dir)
	for i := 0; i < 8 && dir != "" && dir != "/" && dir != "."; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", false
}

func openWiki() (*okf.Wiki, error) { return okf.Open(wikiRoot()) }

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// ---------------------------------------------------------------------------
// init / doctor / spec
// ---------------------------------------------------------------------------

func cmdInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	wiki := fs.String("wiki", "", "wiki 目录（默认自动定位）")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	root := *wiki
	if root == "" {
		root = wikiRoot()
	}
	created, err := okf.Init(root)
	if err != nil {
		return fail(err)
	}
	fmt.Println("wiki:", root)
	if len(created) == 0 {
		fmt.Println("已存在，无变更（幂等）")
		return exitOK
	}
	for _, c := range created {
		fmt.Println("  +", c)
	}
	return exitOK
}

func cmdDoctor(args []string) int {
	fmt.Println("tvop doctor")
	fmt.Println("  wiki        :", wikiRoot())
	if w, err := openWiki(); err == nil {
		if cs, err := w.List(); err == nil {
			fmt.Printf("  概念        : %d\n", len(cs))
		}
		fmt.Println("  仓库根      :", w.RepoRoot())
	} else {
		fmt.Println("  仓库状态    : 未初始化（先跑 tvop init）")
	}
	if r, ok := detectCrawler(); ok {
		fmt.Println("  MediaCrawler:", r)
		fmt.Println("  驱动        :", strings.Join(crawl.DetectRunner(r), " "))
	} else {
		fmt.Println("  MediaCrawler: 未找到（设置 MEDIACRAWLER_ROOT，或克隆到 <仓库根>/third_party/MediaCrawler）")
	}
	fmt.Printf("  词表        : %d 类型 / %d 声部 / %d 条人味规则 / %d 条合规红线\n",
		len(spec.AllTypes()), len(spec.Voices), len(spec.TasteOperators), len(spec.RiskPatterns))
	// 分发自感知：报告 git 同步状态，提醒接收方拉取知识库增量。
	if w, werr := openWiki(); werr == nil {
		if repo := w.RepoRoot(); repo != "" {
			if _, statErr := os.Stat(filepath.Join(repo, ".git")); statErr == nil {
				remote, rerr := gitRun(repo, 5*time.Second, "remote", "get-url", "origin")
				if rerr == nil && remote != "" {
					fmt.Println("  分发远端  :", remote)
					fmt.Println("  同步检查  : tvop repo status（落后远端时会提示 tvop repo pull）")
				} else {
					fmt.Println("  分发远端  : 未配置（git 分发后自动出现；zip 直解无更新通道）")
				}
			} else {
				fmt.Println("  分发通道  : 非 git 仓库（无增量更新；建议用 git clone 安装）")
			}
		}
	}
	return exitOK
}

func detectCrawler() (string, bool) {
	if v := strings.TrimSpace(os.Getenv("MEDIACRAWLER_ROOT")); v != "" {
		if r, ok := crawl.DetectProjectRoot(v); ok {
			return r, true
		}
	}
	if w, err := openWiki(); err == nil {
		repo := w.RepoRoot()
		for _, c := range []string{
			filepath.Join(repo, "third_party", "MediaCrawler"),
			filepath.Join(repo, "MediaCrawler"),
			filepath.Join(filepath.Dir(repo), "MediaCrawler"),
		} {
			if r, ok := crawl.DetectProjectRoot(c); ok {
				return r, true
			}
		}
	}
	if wd, err := os.Getwd(); err == nil {
		if r, ok := crawl.DetectProjectRoot(wd); ok {
			return r, true
		}
	}
	return "", false
}

func cmdSpec(args []string) int {
	what := "types"
	if len(args) > 0 {
		what = args[0]
	}
	switch what {
	case "types":
		fmt.Print("概念类型词表（封闭）\n\n")
		fmt.Printf("  %-18s %-6s %-10s %s\n", "TYPE", "LAYER", "BUCKET", "必填字段")
		for _, d := range spec.TypeDocs {
			layer := "领域"
			if d.Base {
				layer = "基座"
			}
			req := strings.Join(d.Requires, ",")
			if req == "" {
				req = "—"
			}
			fmt.Printf("  %-18s %-6s %-10s %s\n", d.Type, layer, strings.TrimSuffix(d.PathHint, "/"), req)
		}
	case "voices":
		fmt.Print("声部画像（封闭）\n\n")
		for _, v := range spec.Voices {
			core := ""
			if v.Core {
				core = " ★核心"
			}
			fmt.Printf("  %-16s %-10s%s\n    %s\n    蒸馏侧重：%s\n    目标张力 %.2f · 目标长度 %d 字\n",
				v.ID, v.MCPlatform, core, v.Trait, v.Focus, v.TargetTension, v.TargetLength)
		}
		if v := crawl.DefaultVoiceFor(crawl.PlatformKS); v == "" {
			fmt.Println("\n  注：快手（ks）未设专属声部——需显式 --voice 指定，否则按语料特征归入现有声部。")
		}
	case "weights":
		fmt.Printf("五因子权重  weight = α·use + β·recency + γ·link + δ·explicit + ε·quality\n\n")
		fmt.Printf("  α use      = %.2f  读满 %d 次即满分（创建即记 1 次）\n", spec.AlphaUse, 4)
		fmt.Printf("  β recency  = %.2f  45 天半衰\n", spec.BetaRecency)
		fmt.Printf("  γ link     = %.2f  6 条链接即满分\n", spec.GammaLink)
		fmt.Printf("  δ explicit = %.2f  人工确认 / 真实互动回流\n", spec.DeltaExplicit)
		fmt.Printf("  ε quality  = %.2f  蒸馏阶段 4 压力测试得分\n\n", spec.EpsilonQual)
		fmt.Printf("  初始权重 %.2f · 晋升门槛 %.2f · 衰减 %d 天\n", spec.InitialWeight, spec.PromoteWeight, spec.DecayIdleDays)
		fmt.Printf("  人味门 %.2f · 合规门 %.2f · 单次扫描上限 %d 条\n", spec.HumannessGate, spec.RiskGate, spec.MaxCrawlPerRun)
		fmt.Println("\n生成排序分：rank = weight × (0.5+0.5·humanness) × (1+0.4·resonance) × (1+0.2·tension) × penalty")
	case "rules":
		fmt.Print("去 AI 味规则集（11 条，封闭）\n\n")
		for _, op := range spec.TasteOperators {
			fmt.Printf("  %2d. %-16s R=%.1f  %s  [%s]\n", op.ID, op.Name, op.Ratio, op.Per, op.Source)
			fmt.Printf("      触发：%s\n", op.Trigger)
			fmt.Printf("      改法：%s\n", op.Fix)
		}
		fmt.Printf("\n不作为改写理由的特征（%d 项，硬约束）：\n", len(spec.NonDiscriminating))
		for _, n := range spec.NonDiscriminating {
			fmt.Printf("  - %s：%s\n", n.Feature, n.Why)
		}
		fmt.Printf("\n人味增强项（%d 项，生成时补足）：\n", len(spec.HumanMarkers))
		for _, h := range spec.HumanMarkers {
			fmt.Printf("  + %s（R=%.2f）：%s\n", h.Feature, h.Ratio, h.Note)
		}
		fmt.Printf("\n合规红线（%d 条）：\n", len(spec.RiskPatterns))
		for _, r := range spec.RiskPatterns {
			fmt.Printf("  ! %s %s（风险 %.2f）→ %s\n", r.Code, r.Zh, r.Severity, r.Fix)
		}
	case "platforms":
		for _, p := range crawl.AllPlatforms {
			v := crawl.DefaultVoiceFor(p)
			if v == "" {
				v = "（需显式指定）"
			}
			fmt.Printf("  %-6s → %s\n", p, v)
		}
	case "json":
		return printJSON(map[string]any{
			"types": spec.AllTypes(), "voices": spec.Voices,
			"operators": spec.TasteOperators, "non_discriminating": spec.NonDiscriminating,
			"human_markers": spec.HumanMarkers, "risk_patterns": spec.RiskPatterns,
			"platforms": crawl.AllPlatforms,
		})
	default:
		fmt.Fprintf(os.Stderr, "未知 spec 主题：%s（可选 types/voices/weights/rules/platforms/json）\n", what)
		return exitInvalid
	}
	return exitOK
}

// ---------------------------------------------------------------------------
// okf
// ---------------------------------------------------------------------------

func cmdOKF(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop okf <new|get|list|reindex|validate|sourcemap|test|account>")
		return exitInvalid
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "new":
		return okfNew(rest)
	case "get":
		return okfGet(rest)
	case "list":
		return okfList(rest)
	case "reindex":
		return okfReindex(rest)
	case "validate":
		return okfValidate(rest)
	case "sourcemap":
		return okfSourcemap(rest)
	case "test":
		return okfTest(rest)
	case "account":
		return okfAccount(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 okf 子命令：%s\n", sub)
		return exitInvalid
	}
}

func okfNew(args []string) int {
	fs := flag.NewFlagSet("okf new", flag.ContinueOnError)
	typ := fs.String("type", "", "概念类型（词表内）")
	slug := fs.String("slug", "", "短名（路径片段）")
	title := fs.String("title", "", "标题")
	voice := fs.String("voice", "", "声部")
	platform := fs.String("platform", "", "平台")
	command := fs.String("command", "", "命令（Runbook 用）")
	tier := fs.String("tier", spec.TierShort, "层级")
	verdict := fs.String("verdict", spec.VerdictUnverified, "结论")
	sourceRef := fs.String("source-ref", "", "来源指针")
	bodyFile := fs.String("body-file", "", "正文文件")
	links := fs.String("links", "", "出链，逗号分隔")
	humanness := fs.Float64("humanness", 0, "人味分 0-1")
	resonance := fs.Float64("resonance", 0, "共鸣度 0-1")
	tension := fs.Float64("tension", 0, "对立度 0-1")
	risk := fs.Float64("risk", 0, "风险分 0-1")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if *typ == "" || *slug == "" {
		fmt.Fprintln(os.Stderr, "必须提供 --type 与 --slug")
		return exitInvalid
	}
	id, err := okf.IDFor(*typ, *slug)
	if err != nil {
		return fail(err)
	}
	body := ""
	if *bodyFile != "" {
		b, rerr := os.ReadFile(*bodyFile)
		if rerr != nil {
			return fail(errx.Wrap(errx.KindNotFound, "okf.new", *bodyFile, rerr))
		}
		body = string(b)
	}
	if *title == "" {
		*title = *slug
	}
	wi := okf.NewConceptWeightInput()
	c := &okf.Concept{
		ID: id, Type: *typ, Title: *title, Voice: *voice, Platform: *platform, Command: *command,
		Top: map[string]string{},
		Atelier: okf.Atelier{
			Weight: okf.ComputeWeight(wi), Tier: *tier, Verdict: *verdict,
			UseCount: 1, SourceRef: *sourceRef,
			Humanness: *humanness, Resonance: *resonance, Tension: *tension, Risk: *risk,
			Links: splitNonEmpty(*links),
		},
		Body: body,
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	if err := w.Put(c, "okf-new", "创建概念 "+*typ); err != nil {
		return fail(err)
	}
	fmt.Println("已创建:", id)
	fmt.Println("  weight:", c.Atelier.Weight, " tier:", c.Atelier.Tier, " verdict:", c.Atelier.Verdict)
	return exitOK
}

func okfGet(args []string) int {
	fs := flag.NewFlagSet("okf get", flag.ContinueOnError)
	account := fs.Bool("account", false, "读取即记账（use_count++ 并重算权重）")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "用法：tvop okf get <概念 ID> [--account]")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	c, err := w.Get(fs.Arg(0))
	if err != nil {
		return fail(err)
	}
	if *account {
		if err := w.AccountUse(c); err != nil {
			return fail(err)
		}
	}
	if *asJSON {
		return printJSON(map[string]any{
			"id": c.ID, "type": c.Type, "title": c.Title, "voice": c.Voice,
			"platform": c.Platform, "atelier": c.Atelier, "usable": c.Usable(),
			"unusable_because": c.UnusableBecause(), "rank": c.Rank(), "body": c.Body,
		})
	}
	fmt.Println("ID       :", c.ID)
	fmt.Println("type     :", c.Type)
	fmt.Println("title    :", c.Title)
	if c.Voice != "" {
		fmt.Println("voice    :", c.Voice)
	}
	fmt.Printf("weight   : %.3f   tier: %s   verdict: %s   use_count: %d\n",
		c.Atelier.Weight, c.Atelier.Tier, c.Atelier.Verdict, c.Atelier.UseCount)
	fmt.Printf("humanness: %.3f   resonance: %.3f   tension: %.3f   risk: %.3f\n",
		c.Atelier.Humanness, c.Atelier.Resonance, c.Atelier.Tension, c.Atelier.Risk)
	fmt.Printf("rank     : %.3f   可用于生成: %v\n", c.Rank(), c.Usable())
	if b := c.UnusableBecause(); b != "" {
		fmt.Println("不可用原因:", b)
	}
	fmt.Println("source   :", c.Atelier.SourceRef)
	if len(c.Atelier.Links) > 0 {
		fmt.Println("links    :", strings.Join(c.Atelier.Links, ", "))
	}
	fmt.Println("---")
	fmt.Println(strings.TrimRight(c.Body, "\n"))
	return exitOK
}

func okfList(args []string) int {
	fs := flag.NewFlagSet("okf list", flag.ContinueOnError)
	typ := fs.String("type", "", "按类型过滤")
	voice := fs.String("voice", "", "按声部过滤")
	tier := fs.String("tier", "", "按层级过滤")
	usable := fs.Bool("usable", false, "只列可用于生成的")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	cs, err := w.List()
	if err != nil {
		return fail(err)
	}
	var kept []*okf.Concept
	for _, c := range cs {
		if *typ != "" && c.Type != *typ {
			continue
		}
		if *voice != "" && c.Voice != *voice {
			continue
		}
		if *tier != "" && c.Atelier.Tier != *tier {
			continue
		}
		if *usable && !c.Usable() {
			continue
		}
		kept = append(kept, c)
	}
	if *asJSON {
		out := make([]map[string]any, 0, len(kept))
		for _, c := range kept {
			out = append(out, map[string]any{
				"id": c.ID, "type": c.Type, "title": c.Title, "voice": c.Voice,
				"tier": c.Atelier.Tier, "verdict": c.Atelier.Verdict,
				"weight": c.Atelier.Weight, "humanness": c.Atelier.Humanness,
				"risk": c.Atelier.Risk, "rank": c.Rank(), "usable": c.Usable(),
			})
		}
		return printJSON(map[string]any{"count": len(out), "concepts": out})
	}
	fmt.Printf("%-42s %-16s %-14s %-6s %-11s %6s %6s %6s %s\n",
		"ID", "TYPE", "VOICE", "TIER", "VERDICT", "WEIGHT", "HUMAN", "RISK", "可用")
	for _, c := range kept {
		u := ""
		if c.Usable() {
			u = "✓"
		}
		fmt.Printf("%-42s %-16s %-14s %-6s %-11s %6.3f %6.2f %6.2f %s\n",
			c.ID, c.Type, c.Voice, c.Atelier.Tier, c.Atelier.Verdict,
			c.Atelier.Weight, c.Atelier.Humanness, c.Atelier.Risk, u)
	}
	fmt.Printf("共 %d 个概念\n", len(kept))
	return exitOK
}

func okfReindex(_ []string) int {
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	n, err := w.Reindex()
	if err != nil {
		return fail(err)
	}
	fmt.Printf("index.md 已重建，%d 个概念\n", n)
	return exitOK
}

func okfValidate(args []string) int {
	fs := flag.NewFlagSet("okf validate", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "JSON 输出")
	quiet := fs.Bool("quiet", false, "只输出结论")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	rep, err := w.Validate()
	if err != nil {
		return fail(err)
	}
	if *asJSON {
		printJSON(reportJSONOut(rep))
	} else if !*quiet {
		for _, i := range rep.Issues {
			fmt.Println(i.String())
		}
	}
	if !*quiet {
		fmt.Printf("\n概念 %d · verified %d · 可用于生成 %d · 错误 %d · 告警 %d\n",
			rep.Concepts, rep.Verified, rep.Usable, rep.Errors(), rep.Warnings())
	}
	if rep.Valid() {
		fmt.Println("✅ okf validate 全绿")
		return exitOK
	}
	fmt.Println("❌ okf validate 未通过（校验不过 = 落库失败，禁止带病服务）")
	return exitFail
}

func reportJSONOut(rep *okf.Report) map[string]any {
	issues := make([]map[string]any, 0, len(rep.Issues))
	for _, i := range rep.Issues {
		issues = append(issues, map[string]any{
			"severity": string(i.Severity), "code": i.Code, "subject": i.Subject,
			"msg": i.Msg, "fix": i.Fix,
		})
	}
	return map[string]any{
		"valid": rep.Valid(), "errors": rep.Errors(), "warnings": rep.Warnings(),
		"concepts": rep.Concepts, "verified": rep.Verified, "usable": rep.Usable, "issues": issues,
	}
}

func okfSourcemap(args []string) int {
	fs := flag.NewFlagSet("okf sourcemap", flag.ContinueOnError)
	id := fs.String("id", "", "概念 ID")
	sources := fs.String("source", "", "来源，格式 kind:path（可重复，逗号分隔）")
	fetched := fs.String("fetched", "", "抓取日期")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if *id == "" {
		fmt.Fprintln(os.Stderr, "必须提供 --id")
		return exitInvalid
	}
	var entries []okf.SourceEntry
	for _, s := range strings.Split(*sources, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		kind, path := "manual", s
		if k, p, ok := strings.Cut(s, ":"); ok {
			kind, path = strings.TrimSpace(k), strings.TrimSpace(p)
		}
		entries = append(entries, okf.SourceEntry{Kind: kind, Path: path, Fetched: *fetched})
	}
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "至少需要一个 --source kind:path")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	if err := w.WriteSourceMap(*id, *fetched, entries); err != nil {
		return fail(err)
	}
	if err := w.AppendLog("okf-sourcemap", *id, fmt.Sprintf("登记 %d 条来源", len(entries))); err != nil {
		return fail(err)
	}
	fmt.Println("已写入 Source Map:", w.SourceMapPath(*id))
	return exitOK
}

func okfTest(args []string) int {
	fs := flag.NewFlagSet("okf test", flag.ContinueOnError)
	id := fs.String("id", "", "概念 ID")
	triggers := fs.String("triggers", "", "触发用例，分号分隔")
	decoys := fs.String("decoys", "", "诱饵负例，分号分隔")
	negatives := fs.String("negatives", "", "兄弟互斥负例，分号分隔")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if *id == "" || *triggers == "" || *decoys == "" || *negatives == "" {
		fmt.Fprintln(os.Stderr, "必须提供 --id / --triggers / --decoys / --negatives（诱饵与互斥负例不可省略）")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	if err := w.WriteTestPrompts(*id, splitSemi(*triggers), splitSemi(*decoys), splitSemi(*negatives)); err != nil {
		return fail(err)
	}
	if err := w.AppendLog("okf-test", *id, "写入 test-prompts.json"); err != nil {
		return fail(err)
	}
	fmt.Println("已写入:", w.TestPath(*id))
	return exitOK
}

func okfAccount(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop okf account <概念 ID>...")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	for _, id := range args {
		c, gerr := w.Get(id)
		if gerr != nil {
			return fail(gerr)
		}
		if aerr := w.AccountUse(c); aerr != nil {
			return fail(aerr)
		}
		fmt.Printf("%s use_count=%d weight=%.3f\n", c.ID, c.Atelier.UseCount, c.Atelier.Weight)
	}
	return exitOK
}

// ---------------------------------------------------------------------------
// search / gaps
// ---------------------------------------------------------------------------

func cmdSearch(args []string) int {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	voice := fs.String("voice", "", "限定声部")
	typ := fs.String("type", "", "限定类型")
	top := fs.Int("top", spec.DefaultTopK, "返回条数")
	usable := fs.Bool("usable", false, "只返回可用于生成的")
	arch := fs.Bool("arch", false, "包含 archive")
	noAccount := fs.Bool("no-account", false, "不记账")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	terms := fs.Args()
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	hits, err := w.Search(okf.Query{Terms: terms, Voice: *voice, Type: *typ,
		Top: *top, OnlyUsable: *usable, IncludeArch: *arch})
	if err != nil {
		return fail(err)
	}
	if *asJSON {
		out := make([]map[string]any, 0, len(hits))
		for _, h := range hits {
			out = append(out, map[string]any{
				"id": h.Concept.ID, "type": h.Concept.Type, "title": h.Concept.Title,
				"voice": h.Concept.Voice, "tier": h.Concept.Atelier.Tier,
				"verdict": h.Concept.Atelier.Verdict, "weight": h.Concept.Atelier.Weight,
				"humanness": h.Concept.Atelier.Humanness, "risk": h.Concept.Atelier.Risk,
				"rank": h.Rank, "score": h.Score, "fields": h.Fields,
				"snippet": h.Snippet, "usable": h.Concept.Usable(),
			})
		}
		return printJSON(map[string]any{"count": len(out), "hits": out})
	}
	if len(hits) == 0 {
		fmt.Println("（零命中）—— 这是知识缺口，跑 tvop gaps 或补投喂后重新蒸馏")
		return exitOK
	}
	for i, h := range hits {
		u := "草稿"
		if h.Concept.Usable() {
			u = "可用"
		}
		fmt.Printf("%2d. [%s] %-42s w=%.2f h=%.2f r=%.2f rank=%.3f\n",
			i+1, u, h.Concept.ID, h.Concept.Atelier.Weight,
			h.Concept.Atelier.Humanness, h.Concept.Atelier.Risk, h.Rank)
		if h.Snippet != "" {
			fmt.Println("    " + h.Snippet)
		}
	}
	if !*noAccount {
		n := 0
		for _, h := range hits {
			if err := w.AccountUse(h.Concept); err != nil {
				continue
			}
			n++
		}
		fmt.Printf("\n读取即记账：%d 个概念 use_count++ 并重算权重\n", n)
	}
	return exitOK
}

func cmdGaps(args []string) int {
	fs := flag.NewFlagSet("gaps", flag.ContinueOnError)
	voice := fs.String("voice", "", "限定声部")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	gaps, err := w.DetectGaps(fs.Args(), *voice)
	if err != nil {
		return fail(err)
	}
	if len(gaps) == 0 {
		fmt.Println("无缺口：全部要素都有可用于生成的概念支撑")
		return exitOK
	}
	for _, g := range gaps {
		fmt.Printf("[缺口] %s —— %s\n       → %s\n", g.Term, g.Reason, g.Suggest)
	}
	return exitOK
}

// ---------------------------------------------------------------------------
// taste / weight
// ---------------------------------------------------------------------------

func cmdTaste(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop taste <score|rules>")
		return exitInvalid
	}
	switch args[0] {
	case "rules":
		return cmdSpec([]string{"rules"})
	case "score":
		fs := flag.NewFlagSet("taste score", flag.ContinueOnError)
		file := fs.String("file", "", "从文件读取文本")
		asJSON := fs.Bool("json", false, "JSON 输出")
		quiet := fs.Bool("quiet", false, "只输出分数")
		if err := fs.Parse(args[1:]); err != nil {
			return exitInvalid
		}
		var text string
		if *file != "" {
			b, err := os.ReadFile(*file)
			if err != nil {
				return fail(errx.Wrap(errx.KindNotFound, "taste.score", *file, err))
			}
			text = string(b)
		} else {
			text = strings.Join(fs.Args(), " ")
		}
		if strings.TrimSpace(text) == "" {
			fmt.Fprintln(os.Stderr, "没有可评估的文本（用 --file 或直接给文本）")
			return exitInvalid
		}
		rep := taste.Score(text)
		if *asJSON {
			return printJSON(tasteReportOut(rep))
		}
		if *quiet {
			fmt.Printf("%.3f\n", rep.Humanness)
			if rep.Gate {
				return exitOK
			}
			return exitFail
		}
		fmt.Printf("字数 %d · 段 %d · 句 %d\n", rep.Chars, rep.Paras, rep.Sentences)
		fmt.Printf("AI 味 %.3f  →  人味分 %.3f  （门槛 %.2f：%s）\n",
			rep.AITone, rep.Humanness, spec.HumannessGate, gateText(rep.Gate))
		if len(rep.Hits) > 0 {
			fmt.Println("\n命中（按超出人类基线的程度排序）：")
			for _, h := range rep.Hits {
				fmt.Printf("  %2d. %-16s ×%-3d  频率 %.2f/%s（超基线 %.0f%%）权重 %.3f\n",
					h.Op.ID, h.Op.Name, h.Count, h.Rate, h.Op.Per, h.Excess*100, h.Weight)
				fmt.Printf("      → %s\n", h.Op.Fix)
				for _, e := range h.Examples {
					fmt.Printf("        例：%s\n", e)
				}
			}
		} else {
			fmt.Println("\n未命中任何规则——按白名单原则，这段文本无需改写。")
		}
		fmt.Println("\n人味增强项（只观测，不构成改写理由）：")
		for _, m := range rep.Markers {
			mark := "✓"
			if m.Deficit {
				mark = "△ 偏少"
			}
			fmt.Printf("  %s %-28s 实测 %.2f/千字   人类基线 %.2f\n", mark, m.Feature, m.Rate, m.Baseline)
		}
		if len(rep.RiskHits) > 0 {
			fmt.Printf("\n⚠ 合规风险 %.2f：%s\n", rep.Risk, strings.Join(rep.RiskHits, ", "))
		}
		for _, n := range rep.Notes {
			fmt.Println("· " + n)
		}
		if rep.Gate {
			return exitOK
		}
		return exitFail
	default:
		fmt.Fprintf(os.Stderr, "未知 taste 子命令：%s\n", args[0])
		return exitInvalid
	}
}

func tasteReportOut(rep taste.Report) map[string]any {
	hits := make([]map[string]any, 0, len(rep.Hits))
	for _, h := range rep.Hits {
		hits = append(hits, map[string]any{
			"id": h.Op.ID, "code": h.Op.Code, "name": h.Op.Name, "count": h.Count,
			"rate": h.Rate, "excess": h.Excess, "weight": h.Weight,
			"per": h.Op.Per, "source": h.Op.Source, "fix": h.Op.Fix, "examples": h.Examples,
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
		"gate_threshold": spec.HumannessGate, "risk": rep.Risk, "risk_hits": rep.RiskHits,
		"risk_gate": spec.RiskGate, "hits": hits, "markers": markers,
		"hints": rep.Hints(), "notes": rep.Notes,
	}
}

func gateText(ok bool) string {
	if ok {
		return "通过"
	}
	return "未通过，需回炉"
}

func cmdWeight(args []string) int {
	if len(args) < 2 || args[0] != "explain" {
		fmt.Fprintln(os.Stderr, "用法：tvop weight explain <概念 ID>")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	c, err := w.Get(args[1])
	if err != nil {
		return fail(err)
	}
	use := float64(c.Atelier.UseCount) / 4.0
	if use > 1 {
		use = 1
	}
	link := float64(len(c.Atelier.Links)*2) / 6.0
	if link > 1 {
		link = 1
	}
	explicit, quality := 0.5, 0.7
	if c.Verified() {
		explicit, quality = 0.75, 0.85
	}
	days := 0.0
	if t, perr := time.ParseInLocation("2006-01-02", c.Atelier.Updated, time.Local); perr == nil {
		days = time.Since(t).Hours() / 24
	}
	recency := 1.0
	if days > 0 {
		recency = expNeg(days / 45.0)
	}
	fmt.Printf("概念 %s\n\n", c.ID)
	fmt.Printf("  α·use      = %.2f × %.3f = %.3f   （use_count=%d，读满 4 次满分）\n",
		spec.AlphaUse, use, spec.AlphaUse*use, c.Atelier.UseCount)
	fmt.Printf("  β·recency  = %.2f × %.3f = %.3f   （最近 %s，%.0f 天前，45 天半衰）\n",
		spec.BetaRecency, recency, spec.BetaRecency*recency, c.Atelier.Updated, days)
	fmt.Printf("  γ·link     = %.2f × %.3f = %.3f   （%d 条出链，6 条满分）\n",
		spec.GammaLink, link, spec.GammaLink*link, len(c.Atelier.Links))
	fmt.Printf("  δ·explicit = %.2f × %.3f = %.3f   （%s）\n",
		spec.DeltaExplicit, explicit, spec.DeltaExplicit*explicit,
		verifiedNote(c.Verified()))
	fmt.Printf("  ε·quality  = %.2f × %.3f = %.3f   （%s）\n",
		spec.EpsilonQual, quality, spec.EpsilonQual*quality, verifiedNote(c.Verified()))
	fmt.Printf("\n  → 现权重 %.3f（磁盘值）\n", c.Atelier.Weight)
	fmt.Printf("\n生成排序分 rank = %.3f\n", c.Rank())
	fmt.Printf("  weight %.3f × (0.5+0.5×%.2f) × (1+0.4×%.2f) × (1+0.2×%.2f) × penalty(%s)\n",
		c.Atelier.Weight, c.Atelier.Humanness, c.Atelier.Resonance, c.Atelier.Tension, penaltyNote(c.Atelier.Risk))
	return exitOK
}

func expNeg(x float64) float64 {
	// 避免引入 math 只为 exp(-x) 时的额外依赖；这里用级数展开到 1e-9。
	if x < 0 {
		x = 0
	}
	sum, term := 1.0, 1.0
	for i := 1; i < 40; i++ {
		term *= -x / float64(i)
		sum += term
		if term < 1e-12 && term > -1e-12 {
			break
		}
	}
	if sum < 0 {
		return 0
	}
	return sum
}

func verifiedNote(v bool) string {
	if v {
		return "已 verified，显式反馈与质量基线更高"
	}
	return "未 verified，用中性基线"
}

func penaltyNote(risk float64) string {
	if risk > spec.RiskGate {
		return "风险超门，×0.15"
	}
	return "1.0"
}

// ---------------------------------------------------------------------------
// govern
// ---------------------------------------------------------------------------

func cmdGovern(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop govern <audit|promote|decay|recall|recompute|report>")
		return exitInvalid
	}
	sub, rest := args[0], args[1:]
	fs := flag.NewFlagSet("govern "+sub, flag.ContinueOnError)
	dry := fs.Bool("dry", false, "只演练不写入")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(rest); err != nil {
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}

	switch sub {
	case "audit":
		rep, st, err := govern.Audit(w)
		if err != nil {
			return fail(err)
		}
		if *asJSON {
			return printJSON(map[string]any{"report": reportJSONOut(rep), "stats": st})
		}
		for _, i := range rep.Issues {
			fmt.Println(i.String())
		}
		fmt.Println()
		fmt.Println(st.Line())
		fmt.Println(st.HumannessLine())
		fmt.Printf("校验：错误 %d · 告警 %d · %s\n", rep.Errors(), rep.Warnings(), validText(rep.Valid()))
		for _, f := range govern.FeedPlan(st) {
			fmt.Println("补投喂 →", f)
		}
		if rep.Valid() {
			return exitOK
		}
		return exitFail

	case "report":
		st, err := govern.StatsOf(w)
		if err != nil {
			return fail(err)
		}
		if *asJSON {
			return printJSON(st)
		}
		fmt.Println(st.Line())
		fmt.Println(st.HumannessLine())
		fmt.Println("\n按类型：")
		printSortedMap(st.ByType)
		fmt.Println("按声部：")
		printSortedMap(st.ByVoice)
		if len(st.Unusable) > 0 {
			fmt.Println("不可用原因：")
			printSortedMap(st.Unusable)
		}
		for _, f := range govern.FeedPlan(st) {
			fmt.Println("补投喂 →", f)
		}
		return exitOK

	case "promote", "decay", "recompute":
		var changes []govern.Change
		var err error
		switch sub {
		case "promote":
			changes, err = govern.Promote(w, *dry)
		case "decay":
			changes, err = govern.Decay(w, *dry, time.Now())
		case "recompute":
			changes, err = govern.Recompute(w, *dry)
		}
		if err != nil {
			return fail(err)
		}
		if *asJSON {
			return printJSON(map[string]any{"changes": changes, "count": len(changes), "dry": *dry})
		}
		if len(changes) == 0 {
			fmt.Println("无变更")
			return exitOK
		}
		for _, c := range changes {
			fmt.Println(c.String())
		}
		fmt.Printf("共 %d 项%s\n", len(changes), dryNote(*dry))
		if !*dry {
			if _, rerr := w.Reindex(); rerr != nil {
				return fail(rerr)
			}
			fmt.Println("index.md 已重建")
		}
		return exitOK

	case "recall":
		if fs.NArg() == 0 {
			fmt.Fprintln(os.Stderr, "用法：tvop govern recall <概念 ID>...")
			return exitInvalid
		}
		changes, err := govern.Recall(w, fs.Args())
		if err != nil {
			return fail(err)
		}
		for _, c := range changes {
			fmt.Println(c.String())
		}
		fmt.Println("已召回并重建索引")
		return exitOK

	default:
		fmt.Fprintf(os.Stderr, "未知 govern 子命令：%s\n", sub)
		return exitInvalid
	}
}

func validText(v bool) string {
	if v {
		return "全绿"
	}
	return "未通过"
}

func dryNote(d bool) string {
	if d {
		return "（演练，未写入）"
	}
	return ""
}

func printSortedMap(m map[string]int) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Printf("  %-28s %d\n", k, m[k])
	}
	if len(keys) == 0 {
		fmt.Println("  （空）")
	}
}

// ---------------------------------------------------------------------------
// crawl
// ---------------------------------------------------------------------------

func cmdCrawl(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop crawl <platforms|plan|run|ingest|select|feed|tension>")
		return exitInvalid
	}
	sub, rest := args[0], args[1:]
	switch sub {
	case "platforms":
		return cmdSpec([]string{"platforms"})
	case "plan":
		return crawlPlan(rest)
	case "run":
		return crawlRun(rest)
	case "ingest":
		return crawlIngest(rest)
	case "select", "tension":
		return crawlSelect(sub, rest)
	case "feed":
		return crawlFeed(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 crawl 子命令：%s\n", sub)
		return exitInvalid
	}
}

func crawlPlan(args []string) int {
	fs := flag.NewFlagSet("crawl plan", flag.ContinueOnError)
	platform := fs.String("platform", "", "平台")
	keywords := fs.String("keywords", "", "关键词，逗号分隔")
	voice := fs.String("voice", "", "声部")
	typ := fs.String("type", "search", "search/detail/creator")
	login := fs.String("lt", "qrcode", "qrcode/phone/cookie")
	maxNotes := fs.Int("max-notes", 30, "最多抓取条目")
	maxComments := fs.Int("max-comments", 50, "每条最多评论")
	sub := fs.Bool("sub-comment", false, "抓二级评论")
	headless := fs.Bool("headless", false, "无头模式")
	out := fs.String("save-path", "", "输出目录")
	root := fs.String("root", "", "MediaCrawler 仓库根")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	proot := *root
	if proot == "" {
		r, ok := detectCrawler()
		if !ok {
			fmt.Fprintln(os.Stderr, "未找到 MediaCrawler 仓库根：设置 MEDIACRAWLER_ROOT，或用 --root 指定")
			return exitFail
		}
		proot = r
	}
	if *platform == "" || *keywords == "" {
		fmt.Fprintln(os.Stderr, "必须提供 --platform 与 --keywords")
		return exitInvalid
	}
	plan, err := crawl.BuildPlan(proot, crawl.Options{
		Platform: *platform, Keywords: splitNonEmpty(*keywords), Voice: *voice,
		Type: *typ, Login: *login, MaxNotes: *maxNotes, MaxCommentsPerNote: *maxComments,
		GetComment: true, GetSubComment: *sub, Headless: *headless, SaveDataPath: *out,
	})
	if err != nil {
		return fail(err)
	}
	if *asJSON {
		return printJSON(map[string]any{
			"platform": plan.Options.Platform, "voice": plan.Voice, "voice_zh": voiceZh(plan.Voice),
			"project_root": plan.ProjectRoot, "workdir": plan.WorkDir, "argv": plan.Command(),
			"shell": plan.ShellLine(), "estimate_comments": plan.Estimate, "confirm_required": true,
		})
	}
	fmt.Println("爬取计划（未执行）")
	fmt.Printf("  平台      : %s\n", plan.Options.Platform)
	fmt.Printf("  声部      : %s %s\n", plan.Voice, voiceZh(plan.Voice))
	fmt.Printf("  关键词    : %s\n", strings.Join(plan.Options.Keywords, " / "))
	fmt.Printf("  条目上限  : %d（评论上限 %d/条，预计 ≤ %d 条）\n",
		plan.Options.MaxNotes, plan.Options.MaxCommentsPerNote, plan.Estimate)
	fmt.Printf("  仓库根    : %s\n", plan.ProjectRoot)
	fmt.Printf("  驱动      : %s\n", strings.Join(plan.Runner, " "))
	fmt.Printf("\n执行命令：\n  %s\n", plan.ShellLine())
	fmt.Println("\n⚠ 爬取会消耗账号额度并可能触发平台风控，需人工确认后运行：tvop crawl run --yes ...")
	return exitOK
}

func voiceZh(v string) string {
	if d, ok := spec.VoiceOf(v); ok {
		return "（" + d.Zh + "）"
	}
	if v == "" {
		return "（未设专属声部）"
	}
	return ""
}

func crawlRun(args []string) int {
	fs := flag.NewFlagSet("crawl run", flag.ContinueOnError)
	platform := fs.String("platform", "", "平台")
	keywords := fs.String("keywords", "", "关键词，逗号分隔")
	voice := fs.String("voice", "", "声部")
	typ := fs.String("type", "search", "search/detail/creator")
	login := fs.String("lt", "qrcode", "qrcode/phone/cookie")
	maxNotes := fs.Int("max-notes", 30, "最多抓取条目")
	maxComments := fs.Int("max-comments", 50, "每条最多评论")
	sub := fs.Bool("sub-comment", false, "抓二级评论")
	headless := fs.Bool("headless", false, "无头模式")
	out := fs.String("save-path", "", "输出目录")
	root := fs.String("root", "", "MediaCrawler 仓库根")
	yes := fs.Bool("yes", false, "确认执行（必须显式给出）")
	timeout := fs.Duration("timeout", 30*time.Minute, "超时")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	proot := *root
	if proot == "" {
		r, ok := detectCrawler()
		if !ok {
			fmt.Fprintln(os.Stderr, "未找到 MediaCrawler 仓库根")
			return exitFail
		}
		proot = r
	}
	if *platform == "" || *keywords == "" {
		fmt.Fprintln(os.Stderr, "必须提供 --platform 与 --keywords")
		return exitInvalid
	}
	plan, err := crawl.BuildPlan(proot, crawl.Options{
		Platform: *platform, Keywords: splitNonEmpty(*keywords), Voice: *voice,
		Type: *typ, Login: *login, MaxNotes: *maxNotes, MaxCommentsPerNote: *maxComments,
		GetComment: true, GetSubComment: *sub, Headless: *headless, SaveDataPath: *out,
		Confirm: *yes,
	})
	if err != nil {
		return fail(err)
	}
	fmt.Println("执行：", plan.ShellLine())
	res, err := crawl.Run(plan, *timeout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "爬取失败（耗时 %s）：%v\n", res.Duration.Round(time.Second), err)
		fmt.Fprintln(os.Stderr, "此类失败不自动重试。请先确认没有产生半批数据，再人工重跑。")
		return exitFail
	}
	fmt.Printf("完成，退出码 %d，耗时 %s\n", res.ExitCode, res.Duration.Round(time.Second))
	if s := strings.TrimSpace(res.Stdout); s != "" {
		fmt.Println(tailLines(s, 20))
	}
	return exitOK
}

func crawlIngest(args []string) int {
	fs := flag.NewFlagSet("crawl ingest", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop crawl ingest <jsonl 或目录>...")
		return exitInvalid
	}
	res, err := crawl.Ingest(fs.Args()...)
	if err != nil {
		return fail(err)
	}
	if *asJSON {
		return printJSON(map[string]any{
			"comments": len(res.Comments), "notes": len(res.Notes),
			"files": res.Files, "skipped": res.Skipped,
		})
	}
	fmt.Printf("归一化完成：评论 %d 条 · 作品 %d 条 · 跳过 %d 条\n", len(res.Comments), len(res.Notes), res.Skipped)
	for _, f := range res.Files {
		fmt.Println("  ←", f)
	}
	byVoice := map[string]int{}
	for _, c := range res.Comments {
		byVoice[c.Voice]++
	}
	if len(byVoice) > 0 {
		fmt.Println("按声部分布：")
		printSortedMap(byVoice)
	}
	return exitOK
}

func crawlSelect(mode string, args []string) int {
	fs := flag.NewFlagSet("crawl "+mode, flag.ContinueOnError)
	voice := fs.String("voice", "", "声部")
	platform := fs.String("platform", "", "平台")
	minLikes := fs.Int("min-likes", 0, "最低点赞")
	minLen := fs.Int("min-len", 0, "最短字数")
	maxLen := fs.Int("max-len", 0, "最长字数")
	top := fs.Int("top", 20, "取前 N 条")
	sortBy := fs.String("sort", "like", "like/length/recent")
	minTension := fs.Int("tension-likes", 100, "对立度配对的点赞阈值")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "需要至少一个 jsonl/目录输入")
		return exitInvalid
	}
	res, err := crawl.Ingest(fs.Args()...)
	if err != nil {
		return fail(err)
	}
	if mode == "tension" {
		pairs := crawl.TensionPairs(res.Comments, *minTension)
		if *asJSON {
			return printJSON(map[string]any{"pairs": pairs, "count": len(pairs)})
		}
		fmt.Printf("对立轴配对 %d 组（同作品下双向高赞）\n\n", len(pairs))
		for i, p := range pairs {
			fmt.Printf("%2d. note=%s\n    A(%d赞)：%s\n    B(%d赞)：%s\n",
				i+1, p[0].NoteID, p[0].LikeCount, clipLine(p[0].Content, 90),
				p[1].LikeCount, clipLine(p[1].Content, 90))
		}
		return exitOK
	}
	sel := crawl.Select(res.Comments, crawl.SelectOptions{
		Voice: *voice, Platform: *platform, MinLikes: *minLikes,
		MinLen: *minLen, MaxLen: *maxLen, Top: *top, SortBy: *sortBy,
	})
	if *asJSON {
		return printJSON(map[string]any{"count": len(sel), "comments": sel})
	}
	fmt.Printf("选出 %d 条（声部 %s · 排序 %s）\n\n", len(sel), orDash(*voice), *sortBy)
	for i, c := range sel {
		fmt.Printf("%2d. [%s] %d赞 %d字 %s\n    %s\n", i+1, c.Platform, c.LikeCount,
			len([]rune(c.Content)), c.CreateTime, clipLine(c.Content, 120))
	}
	return exitOK
}

// crawlFeed 把爬到的评论拆分投喂进库：归一化 → 选样 → 建 Transcription 草稿 + Source Map。
//
// 注意：这一步只完成 IF-1（素材入库）；RIA-TV++ 蒸馏（IF-2）由 tvop distill plan
// 产出的计划交给模型执行，产出的能力卡才写 verdict=verified。
func crawlFeed(args []string) int {
	fs := flag.NewFlagSet("crawl feed", flag.ContinueOnError)
	voice := fs.String("voice", "", "声部（缺省则按平台推断）")
	batch := fs.String("batch", time.Now().Format("2006-01-02"), "批次标识，用于命名与 Source Map")
	limit := fs.Int("limit", 25, "每个声部最多入库条数")
	minLikes := fs.Int("min-likes", 0, "最低点赞")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "用法：tvop crawl feed <jsonl 或目录>... [--voice V] [--batch ID]")
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	res, err := crawl.Ingest(fs.Args()...)
	if err != nil {
		return fail(err)
	}
	byVoice := map[string][]crawl.Comment{}
	for _, c := range res.Comments {
		v := c.Voice
		if *voice != "" {
			v = *voice
		}
		if v == "" {
			v = "unassigned"
		}
		byVoice[v] = append(byVoice[v], c)
	}

	type created struct {
		ID        string  `json:"id"`
		Voice     string  `json:"voice"`
		Count     int     `json:"count"`
		Runes     int     `json:"runes"`
		Humanness float64 `json:"humanness"`
	}
	var out []created
	for v, group := range byVoice {
		sel := crawl.Select(group, crawl.SelectOptions{Voice: "", MinLikes: *minLikes, Top: *limit, SortBy: "like"})
		if len(sel) == 0 {
			continue
		}
		var sb strings.Builder
		var refs []string
		rawFiles := map[string]bool{}
		for i, c := range sel {
			fmt.Fprintf(&sb, "### 样本 %d（%s · %d 赞）\n\n%s\n\n", i+1, c.Platform, c.LikeCount, strings.TrimSpace(c.Content))
			if c.CommentID != "" {
				refs = append(refs, c.Platform+":"+c.CommentID)
			}
			rawFiles[c.SourceFile] = true
		}
		fmt.Fprintf(&sb, "\n---\n\n本段由 `tvop crawl feed` 于 %s 从 %d 条原始评论中按声部特征选出，**尚未蒸馏**。\n",
			time.Now().Format("2006-01-02 15:04"), len(group))
		sb.WriteString("下一步：跑 `tvop distill plan` 生成 RIA-TV++ 计划，把本段升级为 Comment Pattern / Voice Persona 并发 verdict=verified。\n")

		slug := strings.ReplaceAll(v, "--", "-") + "-" + *batch
		id, ierr := okf.IDFor(spec.TypeTranscript, slug)
		if ierr != nil {
			return fail(ierr)
		}
		// Transcript 类型要求必填 platform（Concept.Platform 专字段，校验门检查）。
		// 一批可能混多平台（feed 未按平台拆分时），单平台直取，多平台全部列出。
		platSet := map[string]bool{}
		var plats []string
		for _, c := range sel {
			if c.Platform != "" && !platSet[c.Platform] {
				platSet[c.Platform] = true
				plats = append(plats, c.Platform)
			}
		}
		sort.Strings(plats)
		plat := strings.Join(plats, ",")
		body := sb.String()
		sc := taste.Score(body)
		wi := okf.NewConceptWeightInput()
		c := &okf.Concept{
			ID: id, Type: spec.TypeTranscript, Title: v + " 高浓度评论样本 " + *batch,
			Voice: v, Platform: plat, Top: map[string]string{},
			Atelier: okf.Atelier{
				Weight: okf.ComputeWeight(wi), Tier: spec.TierShort, Verdict: spec.VerdictUnverified,
				UseCount: 1, Humanness: sc.Humanness, Risk: sc.Risk,
				Resonance: resonanceOf(sel),
				Updated:   time.Now().Format("2006-01-02"),
			},
			Body: body,
		}
		if err := w.Put(c, "crawl-feed", fmt.Sprintf("投喂 %d 条 %s 评论（%d 个原始文件）", len(sel), v, len(rawFiles))); err != nil {
			return fail(err)
		}
		var entries []okf.SourceEntry
		for f := range rawFiles {
			entries = append(entries, okf.SourceEntry{Kind: "crawl_jsonl", Path: f,
				Fetched: time.Now().Format("2006-01-02"),
				Note:    fmt.Sprintf("%d/%d 条入选", len(sel), len(group))})
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
		if err := w.WriteSourceMap(id, time.Now().Format("2006-01-02"), entries); err != nil {
			return fail(err)
		}
		out = append(out, created{ID: id, Voice: v, Count: len(sel), Runes: len([]rune(body)), Humanness: sc.Humanness})
	}
	if _, err := w.Reindex(); err != nil {
		return fail(err)
	}
	if *asJSON {
		return printJSON(map[string]any{"batches": out, "count": len(out),
			"comments_in": len(res.Comments), "notes_in": len(res.Notes)})
	}
	if len(out) == 0 {
		fmt.Println("没有可入库的评论（检查 --min-likes 或输入文件）")
		return exitOK
	}
	fmt.Printf("已投喂 %d 批（共读入评论 %d 条）：\n", len(out), len(res.Comments))
	for _, o := range out {
		fmt.Printf("  %-38s 声部 %-16s %d 条 · %d 字 · 人味 %.2f\n", o.ID, o.Voice, o.Count, o.Runes, o.Humanness)
	}
	fmt.Println("\nindex.md 已重建。下一步：tvop distill plan")
	return exitOK
}

func resonanceOf(cs []crawl.Comment) float64 {
	if len(cs) == 0 {
		return 0
	}
	var sum float64
	for _, c := range cs {
		// 归一化：点赞 1000 视为满分；回复数作为加成。
		l := float64(c.LikeCount) / 1000.0
		if l > 1 {
			l = 1
		}
		r := float64(c.SubCount) / 50.0
		if r > 1 {
			r = 1
		}
		sum += 0.75*l + 0.25*r
	}
	return okf.Round3(sum / float64(len(cs)))
}

// ---------------------------------------------------------------------------
// distill
// ---------------------------------------------------------------------------

func cmdDistill(args []string) int {
	if len(args) == 0 || args[0] != "plan" {
		fmt.Fprintln(os.Stderr, "用法：tvop distill plan --sources <概念 ID>... [--target T]")
		return exitInvalid
	}
	fs := flag.NewFlagSet("distill plan", flag.ContinueOnError)
	sources := fs.String("sources", "", "待蒸馏的概念 ID，逗号分隔（留空则取全部 unverified 的 Transcript Segment）")
	target := fs.String("target", "", "目标概念类型")
	asJSON := fs.Bool("json", false, "JSON 输出")
	if err := fs.Parse(args[1:]); err != nil {
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	var ids []string
	if *sources != "" {
		ids = splitNonEmpty(*sources)
	} else {
		cs, lerr := w.List()
		if lerr != nil {
			return fail(lerr)
		}
		for _, c := range cs {
			if c.Type == spec.TypeTranscript && !c.Verified() && c.Atelier.Tier != spec.TierArchive {
				ids = append(ids, c.ID)
			}
		}
	}
	if len(ids) == 0 {
		fmt.Println("没有待蒸馏的素材。先跑 tvop crawl feed 投喂语料。")
		return exitOK
	}
	type task struct {
		Source string   `json:"source"`
		Target string   `json:"target"`
		Voice  string   `json:"voice"`
		Stages []string `json:"stages"`
		Gates  []string `json:"gates"`
	}
	var tasks []task
	for _, id := range ids {
		c, gerr := w.Get(id)
		if gerr != nil {
			return fail(gerr)
		}
		tgt := *target
		if tgt == "" {
			switch {
			case c.Voice != "" && c.Type == spec.TypeTranscript:
				tgt = spec.TypeCommentPattern + " + " + spec.TypeVoicePersona
			default:
				tgt = spec.TypePlaybook
			}
		}
		tasks = append(tasks, task{
			Source: id, Target: tgt, Voice: c.Voice,
			Stages: []string{
				"0 Adler 整体理解：结构 → 解释 → 批判 → 应用，产出 Source 概念草稿并与用户确认骨架",
				"1 五提取器并行：框架 / 原则 / 案例 / 反例 / 术语（检索式提取，漏检数必须为 0）",
				"1.5 三重验证：V1 跨域佐证（≥2 独立段落）/ V2 预测力 / V3 独特性——全过才落库",
				"1.6 晋级门：promoted 独立概念 / router 挂 Source 下可路由（不淘汰）",
				"2 RIA++ 六段能力卡：R 阅读 / I 解读 / A1 领域应用 / A2 迁移 / E 执行 / B 边界",
				"3 Zettelkasten 链接：与既有概念双向链接（写进 atelier.links）",
				"4 压力测试：生成 test-prompts.json（触发 + 诱饵负例 + 兄弟互斥负例），通过才标 verified",
				"5 确定性编译入库：权重/tier 判定 → index 更新 → 写 log.md",
			},
			Gates: []string{
				fmt.Sprintf("verdict=verified 时人味分必须 ≥ %.2f（tvop taste score 实测）", spec.HumannessGate),
				fmt.Sprintf("verdict=verified 时风险分必须 ≤ %.2f（合规红线预检）", spec.RiskGate),
				"test-prompts.json 必须含 decoys 与 negatives，缺一不可标 verified",
				"落库后必须 tvop okf validate 全绿",
			},
		})
	}
	if *asJSON {
		return printJSON(map[string]any{"tasks": tasks, "count": len(tasks),
			"target_types": spec.DomainTypes})
	}
	fmt.Printf("RIA-TV++ 蒸馏计划：%d 个素材\n\n", len(tasks))
	for i, t := range tasks {
		fmt.Printf("%2d. %s\n    声部：%s\n    目标类型：%s\n", i+1, t.Source, orDash(t.Voice), t.Target)
		if i == 0 {
			fmt.Println("    七阶段：")
			for _, s := range t.Stages {
				fmt.Println("      ·", s)
			}
			fmt.Println("    硬门：")
			for _, g := range t.Gates {
				fmt.Println("      ✓", g)
			}
		}
	}
	fmt.Println("\n本命令只产出计划。蒸馏由模型按计划执行，产物用 `tvop okf new` / 直接写概念文件落库。")
	return exitOK
}

// ---------------------------------------------------------------------------
// serve / mcp
// ---------------------------------------------------------------------------

func cmdServe(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	addr := fs.String("addr", ":8080", "监听地址")
	if err := fs.Parse(args); err != nil {
		return exitInvalid
	}
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	fmt.Println("tvop serve 监听", *addr, "wiki:", w.Root)
	if err := server.ListenAndServe(*addr, w); err != nil {
		return fail(err)
	}
	return exitOK
}

func cmdMCP(_ []string) int {
	w, err := openWiki()
	if err != nil {
		return fail(err)
	}
	if err := server.ServeMCP(w, os.Stdin, os.Stdout); err != nil {
		return fail(err)
	}
	return exitOK
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

func printJSON(v any) int {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fail(errx.Wrap(errx.KindInternal, "json", "", err))
	}
	fmt.Println(string(b))
	return exitOK
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "✗", err.Error())
	if errx.Retryable(err) {
		return exitInvalid
	}
	return exitFail
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitSemi(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func clipLine(s string, n int) string {
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) <= n {
		return string(r)
	}
	return string(r[:n]) + "…"
}

func tailLines(s string, n int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n")
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
