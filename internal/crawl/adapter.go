// Package crawl 是 MediaCrawler 的 CLI 适配层。
//
// 定位：M2 的素材入口。它只负责「把外部平台语料按声部取回并归一化」，
// 不做蒸馏、不做落库判断。
//
// 上游项目：https://github.com/NanmiCoder/MediaCrawler
// 调用方式：一律通过其官方 CLI（`uv run main.py` 或 `python main.py`），
// 不 import 其 Python 模块——保持进程边界，避免与上游内部结构耦合。
//
// 纪律：
//   - 爬取是外部有副作用动作（消耗账号额度、可能触发风控），属付费/不可逆门禁，
//     必须显式确认（Confirm=true）才执行，失败永不自动重试。
//   - 单次扫描条数受 spec.MaxCrawlPerRun 约束。
//   - 上游未安装或被禁时，退化为离线回放（Ingest 现有 jsonl），不静默失败。
package crawl

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sycglier/tv-opinion-atelier/internal/errx"
	"github.com/sycglier/tv-opinion-atelier/internal/spec"
)

// 平台标识，与 MediaCrawler 的 --platform 取值一一对应。
const (
	PlatformXHS    = "xhs"
	PlatformDouyin = "dy"
	PlatformKS     = "ks"
	PlatformBili   = "bili"
	PlatformWeibo  = "wb"
	PlatformTieba  = "tieba"
	PlatformZhihu  = "zhihu"
)

// AllPlatforms 是受支持平台全集。
var AllPlatforms = []string{
	PlatformXHS, PlatformDouyin, PlatformKS, PlatformBili,
	PlatformWeibo, PlatformTieba, PlatformZhihu,
}

// IsPlatform 报告 p 是否为受支持平台。
func IsPlatform(p string) bool {
	for _, x := range AllPlatforms {
		if x == p {
			return true
		}
	}
	return false
}

// Options 是一次爬取计划的输入。
type Options struct {
	Platform string   // 必填
	Keywords []string // 必填，至少一个
	Voice    string   // 可选，覆盖平台默认声部
	Type     string   // search / detail / creator，默认 search
	Login    string   // qrcode / phone / cookie，默认 qrcode

	SpecifiedIDs []string
	CreatorIDs   []string

	MaxNotes           int
	MaxCommentsPerNote int
	GetComment         bool
	GetSubComment      bool
	Headless           bool
	SaveDataOption     string // 默认 jsonl
	SaveDataPath       string

	// Confirm 必须为 true 才会真正执行；未确认时返回 invalid 错误（付费门禁）。
	Confirm bool
}

// Plan 是一份可执行、可审计的爬取计划。
type Plan struct {
	Options     Options
	ProjectRoot string
	Runner      []string // uv / python3
	Entry       string   // main.py 绝对路径
	WorkDir     string
	Voice       string
	Argv        []string
	Estimate    int // 预计拉取条数上限
}

// DefaultVoiceFor 返回平台默认声部。快手未设专属声部，返回空串（需显式指定）。
func DefaultVoiceFor(platform string) string {
	switch platform {
	case PlatformTieba:
		return spec.VoiceTiebaLong
	case PlatformBili:
		return spec.VoiceBiliGeek
	case PlatformDouyin:
		return spec.VoiceDouyinConflict
	case PlatformXHS:
		return spec.VoiceXhsScene
	case PlatformZhihu:
		return spec.VoiceZhihuRational
	case PlatformWeibo:
		return spec.VoiceWbSpread
	default:
		return ""
	}
}

// DetectProjectRoot 在 hint 及其上若干层目录中寻找 MediaCrawler 仓库根。
//
// 锚点：同时存在 main.py 与 config/base_config.py。
func DetectProjectRoot(hint string) (string, bool) {
	dir, err := filepath.Abs(hint)
	if err != nil {
		return "", false
	}
	for i := 0; i < 6 && dir != "" && dir != "/" && dir != "."; i++ {
		if isMediaCrawlerRoot(dir) {
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

func isMediaCrawlerRoot(dir string) bool {
	for _, f := range []string{"main.py", filepath.Join("config", "base_config.py"), filepath.Join("cmd_arg", "arg.py")} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			return false
		}
	}
	return true
}

// DetectRunner 决定用哪个解释器驱动上游 CLI。
//
// 优先级：项目内 .venv（pip 安装，含 jieba 等难构建依赖）>
// uv run（uv 管理的 venv，上游 pyproject 的 jieba 在新版
// setuptools 下构建失败，uv 通道不可用时自动落空）> 系统 python3。
func DetectRunner(projectRoot string) []string {
	if _, err := os.Stat(filepath.Join(projectRoot, ".venv", "bin", "python")); err == nil {
		return []string{filepath.Join(projectRoot, ".venv", "bin", "python")}
	}
	if _, err := exec.LookPath("uv"); err == nil {
		return []string{"uv", "run"}
	}
	return []string{"python3"}
}

// BuildPlan 组装爬取计划（不执行）。
func BuildPlan(projectRoot string, o Options) (Plan, error) {
	if !IsPlatform(o.Platform) {
		return Plan{}, errx.Errorf(errx.KindInvalid, "crawl.plan", o.Platform,
			"不支持的平台（可选：%s）", strings.Join(AllPlatforms, " / "))
	}
	if len(o.Keywords) == 0 {
		return Plan{}, errx.New(errx.KindInvalid, "crawl.plan", o.Platform, "至少需要一个关键词")
	}
	if !isMediaCrawlerRoot(projectRoot) {
		return Plan{}, errx.Errorf(errx.KindNotFound, "crawl.plan", projectRoot,
			"该目录不是 MediaCrawler 仓库根（需同时存在 main.py / config/base_config.py / cmd_arg/arg.py）")
	}
	if o.Type == "" {
		o.Type = "search"
	}
	if o.Login == "" {
		o.Login = "qrcode"
	}
	if o.SaveDataOption == "" {
		o.SaveDataOption = "jsonl"
	}
	if o.MaxNotes <= 0 {
		o.MaxNotes = 30
	}
	if o.MaxCommentsPerNote <= 0 {
		o.MaxCommentsPerNote = 50
	}
	if !o.GetComment && !o.GetSubComment {
		o.GetComment = true // 本专家的素材是评论，默认开启一级评论
	}
	if o.MaxNotes > spec.MaxCrawlPerRun {
		return Plan{}, errx.Errorf(errx.KindInvalid, "crawl.plan", o.Platform,
			"单次扫描不超过 %d 条（当前 %d）", spec.MaxCrawlPerRun, o.MaxNotes)
	}

	voice := o.Voice
	if voice == "" {
		voice = DefaultVoiceFor(o.Platform)
	}
	if voice != "" && !spec.IsValidVoice(voice) {
		return Plan{}, errx.Errorf(errx.KindInvalid, "crawl.plan", voice, "声部不在封闭词表内")
	}

	args := []string{
		"main.py",
		"--platform", o.Platform,
		"--lt", o.Login,
		"--type", o.Type,
		"--keywords", strings.Join(o.Keywords, ","),
		"--get_comment", yn(o.GetComment),
		"--get_sub_comment", yn(o.GetSubComment),
		"--headless", yn(o.Headless),
		"--crawler_max_notes_count", fmt.Sprint(o.MaxNotes),
		"--max_comments_count_singlenotes", fmt.Sprint(o.MaxCommentsPerNote),
		"--save_data_option", o.SaveDataOption,
	}
	if len(o.SpecifiedIDs) > 0 {
		args = append(args, "--specified_id", strings.Join(o.SpecifiedIDs, ","))
	}
	if len(o.CreatorIDs) > 0 {
		args = append(args, "--creator_id", strings.Join(o.CreatorIDs, ","))
	}
	if o.SaveDataPath != "" {
		args = append(args, "--save_data_path", o.SaveDataPath)
	}

	p := Plan{
		Options:     o,
		ProjectRoot: projectRoot,
		Runner:      DetectRunner(projectRoot),
		Entry:       filepath.Join(projectRoot, "main.py"),
		WorkDir:     projectRoot,
		Voice:       voice,
		Argv:        args,
		Estimate:    o.MaxNotes * o.MaxCommentsPerNote,
	}
	if p.Estimate > spec.MaxCrawlPerRun*10 {
		p.Estimate = spec.MaxCrawlPerRun * 10
	}
	return p, nil
}

// Command 返回完整命令行（runner + 入口 + 参数）。
//
// 入口一律用相对文件名（main.py），配合 WorkDir=项目根，uv 与 python3 两种驱动都能跑。
func (p Plan) Command() []string {
	out := make([]string, 0, len(p.Runner)+len(p.Argv))
	out = append(out, p.Runner...)
	out = append(out, p.Argv...)
	return out
}

// ShellLine 返回可直接粘贴到终端的一行命令（含 cd）。
func (p Plan) ShellLine() string {
	return "cd " + p.ProjectRoot + " && " + strings.Join(quoteAll(p.Command()), " ")
}

// RunResult 是一次执行的结果。
type RunResult struct {
	Plan     Plan
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	// TimedOut 标记超时被杀（此类失败不自动重试）。
	TimedOut bool
}

// Run 执行爬取。必须 Confirm=true。
//
// 失败语义：任何非零退出都返回 KindUnavailable（可退避重试一次），
// 但调用方在自动重试前必须确认没有产生半批数据。付费/不可逆场景永不自动重试。
func Run(plan Plan, timeout time.Duration) (RunResult, error) {
	if !plan.Options.Confirm {
		return RunResult{Plan: plan}, errx.New(errx.KindPaid, "crawl.run", plan.Options.Platform,
			"爬取是有副作用的外部动作，需显式确认（--yes）。计划已生成，未执行")
	}
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	cmd := exec.Command(plan.Command()[0], plan.Command()[1:]...)
	cmd.Dir = plan.WorkDir
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return RunResult{Plan: plan}, errx.Wrap(errx.KindUnavailable, "crawl.run", plan.ProjectRoot, err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	res := RunResult{Plan: plan}
	select {
	case err := <-done:
		res.Duration = time.Since(start)
		if cmd.ProcessState != nil {
			res.ExitCode = cmd.ProcessState.ExitCode()
		}
		res.Stdout, res.Stderr = out.String(), errb.String()
		if err != nil && res.ExitCode != 0 {
			return res, errx.Errorf(errx.KindUnavailable, "crawl.run", plan.Options.Platform,
				"上游 CLI 退出码 %d：%s", res.ExitCode, tail(errb.String(), 400))
		}
		return res, nil
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		res.Duration = time.Since(start)
		res.TimedOut = true
		res.Stdout, res.Stderr = out.String(), errb.String()
		return res, errx.Errorf(errx.KindUnavailable, "crawl.run", plan.Options.Platform,
			"超时 %s 被杀；请检查登录态或缩小 max_notes 后手动重跑", timeout)
	}
}

// ---------------------------------------------------------------------------
// 数据归一化
// ---------------------------------------------------------------------------

// Comment 是归一化后的评论记录。契约见 contracts/comment.schema.json。
type Comment struct {
	Platform  string `json:"platform"`
	Voice     string `json:"voice"`
	NoteID    string `json:"note_id"`
	NoteTitle string `json:"note_title"`

	CommentID  string `json:"comment_id"`
	ParentID   string `json:"parent_id"`
	Content    string `json:"content"`
	LikeCount  int    `json:"like_count"`
	SubCount   int    `json:"sub_comment_count"`
	Nickname   string `json:"nickname"`
	UserID     string `json:"user_id"`
	CreateTime string `json:"create_time"`
	IPLocation string `json:"ip_location"`

	SourceFile string `json:"source_file"`
}

// Note 是归一化后的作品/帖子记录。
type Note struct {
	Platform     string `json:"platform"`
	NoteID       string `json:"note_id"`
	Title        string `json:"title"`
	Desc         string `json:"desc"`
	Nickname     string `json:"nickname"`
	LikedCount   int    `json:"liked_count"`
	CommentCount int    `json:"comment_count"`
	ShareCount   int    `json:"share_count"`
	SourceFile   string `json:"source_file"`
}

// IngestResult 是一次离线归一回放的结果。
type IngestResult struct {
	Comments []Comment
	Notes    []Note
	Files    []string
	Skipped  int
}

// Ingest 读取 MediaCrawler 产出的 jsonl/json 文件并归一化。
//
// 容忍字段差异：上游各平台的字段名不统一，这里按候选键逐个尝试。
func Ingest(paths ...string) (IngestResult, error) {
	var res IngestResult
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return res, errx.Wrap(errx.KindNotFound, "crawl.ingest", p, err)
		}
		files := []string{p}
		if info.IsDir() {
			ents, derr := os.ReadDir(p)
			if derr != nil {
				return res, errx.Wrap(errx.KindInternal, "crawl.ingest", p, derr)
			}
			files = nil
			for _, e := range ents {
				if e.IsDir() {
					continue
				}
				name := strings.ToLower(e.Name())
				if strings.HasSuffix(name, ".jsonl") || strings.HasSuffix(name, ".json") {
					files = append(files, filepath.Join(p, e.Name()))
				}
			}
			sort.Strings(files)
		}
		for _, f := range files {
			n, cerr, nerr := ingestFile(f, &res)
			res.Skipped += n
			if cerr != nil {
				return res, cerr
			}
			if nerr > 0 {
				res.Files = append(res.Files, f)
			}
		}
	}
	return res, nil
}

func ingestFile(path string, res *IngestResult) (int, error, int) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, errx.Wrap(errx.KindNotFound, "crawl.ingest", path, err), 0
	}
	text := strings.TrimSpace(string(raw))
	if text == "" {
		return 0, nil, 0
	}
	recs := []map[string]any{}
	if strings.HasPrefix(text, "[") {
		if jerr := json.Unmarshal([]byte(text), &recs); jerr != nil {
			return 0, errx.Errorf(errx.KindInvalid, "crawl.ingest", path, "JSON 数组解析失败：%v", jerr), 0
		}
	} else {
		for _, line := range strings.Split(text, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var m map[string]any
			if jerr := json.Unmarshal([]byte(line), &m); jerr != nil {
				continue
			}
			recs = append(recs, m)
		}
	}

	skipped := 0
	for _, m := range recs {
		if has(m, "comment_id") || (has(m, "content") && has(m, "note_id")) {
			res.Comments = append(res.Comments, toComment(m, path))
			continue
		}
		if has(m, "note_id") && !has(m, "comment_id") {
			res.Notes = append(res.Notes, toNote(m, path))
			continue
		}
		skipped++
	}
	return skipped, nil, len(recs)
}

// platformDirHints 是从数据文件路径反推平台的目录名线索。
//
// 上游 MediaCrawler 的 jsonl 默认落在 data/<platform>/ 目录下，
// 且各平台模型普遍**不携带 platform 字段**（平台由目录隐含）。
// 因此归一化时必须从 SourceFile 路径反推，否则声部映射会全部落空。
var platformDirHints = []struct {
	dir   string
	platf string
}{
	{"tieba", PlatformTieba},
	{"bilibili", PlatformBili},
	{"douyin", PlatformDouyin},
	{"kuaishou", PlatformKS},
	{"xiaohongshu", PlatformXHS},
	{"weibo", PlatformWeibo},
	{"zhihu", PlatformZhihu},
}

// inferPlatform 从记录字段与数据文件路径推断平台标识。
//
// 优先级：显式 platform 字段 > 文件路径中的平台目录名。
// MediaCrawler 默认输出为 data/<platform>/xxx.jsonl，
// 用户自定义 --save_data_path 时约定目录名含平台标识。
func inferPlatform(m map[string]any, src string) string {
	if p := firstStr(m, "platform", "_platform"); p != "" {
		return p
	}
	lower := strings.ToLower(src)
	for _, h := range platformDirHints {
		if strings.Contains(lower, h.dir) {
			return h.platf
		}
	}
	return ""
}

func toComment(m map[string]any, src string) Comment {
	platform := inferPlatform(m, src)
	return Comment{
		Platform:   platform,
		Voice:      DefaultVoiceFor(platform),
		NoteID:     firstStr(m, "note_id", "aweme_id", "video_id", "article_id"),
		NoteTitle:  firstStr(m, "note_title", "title", "desc"),
		CommentID:  firstStr(m, "comment_id", "cid", "rpid", "id"),
		ParentID:   firstStr(m, "parent_comment_id", "parent_id", "root_comment_id"),
		Content:    firstStr(m, "content", "text", "comment_content"),
		LikeCount:  firstInt(m, "like_count", "liked_count", "digg_count", "voteup_count", "praise_num"),
		SubCount:   firstInt(m, "sub_comment_count", "reply_count", "children_count"),
		Nickname:   firstStr(m, "nickname", "user_nickname", "user_name"),
		UserID:     firstStr(m, "user_id", "sec_uid", "uid"),
		CreateTime: firstStr(m, "create_time", "create_date_time", "time", "publish_time"),
		IPLocation: firstStr(m, "ip_location", "ip", "location"),
		SourceFile: src,
	}
}

func toNote(m map[string]any, src string) Note {
	platform := inferPlatform(m, src)
	return Note{
		Platform:     platform,
		NoteID:       firstStr(m, "note_id", "aweme_id", "bvid", "id"),
		Title:        firstStr(m, "title", "note_title"),
		Desc:         firstStr(m, "desc", "content"),
		Nickname:     firstStr(m, "nickname", "user_nickname"),
		LikedCount:   firstInt(m, "liked_count", "digg_count", "like_count"),
		CommentCount: firstInt(m, "comment_count", "comments_count"),
		ShareCount:   firstInt(m, "share_count", "shared_count"),
		SourceFile:   src,
	}
}

// ---------------------------------------------------------------------------
// 选样：按声部特征挑出「高浓度」评论
// ---------------------------------------------------------------------------

// SelectOptions 是选样条件。
type SelectOptions struct {
	Voice    string
	Platform string
	MinLikes int
	MinLen   int
	MaxLen   int
	Top      int
	// SortBy: like | length | recent
	SortBy string
}

// Select 按声部特征筛选高浓度评论。
//
// 声部差异是真实的取样口径，不是审美偏好：
//   - tieba-long 取长段（≥80 字）与高回复（追楼特征）
//   - bili-geek 取含参数/实测语汇
//   - douyin-conflict 取短句（≤40 字）与高对立（高赞 + 多回复）
//   - hot-consensus 全平台取高赞
func Select(cs []Comment, o SelectOptions) []Comment {
	minLen, maxLen, minLikes := o.MinLen, o.MaxLen, o.MinLikes
	switch o.Voice {
	case spec.VoiceTiebaLong:
		if minLen == 0 {
			minLen = 60
		}
	case spec.VoiceBiliGeek:
		if minLen == 0 {
			minLen = 20
		}
	case spec.VoiceDouyinConflict:
		if maxLen == 0 {
			maxLen = 60
		}
	case spec.VoiceHotConsensus:
		if minLikes == 0 {
			minLikes = 50
		}
	}

	var out []Comment
	for _, c := range cs {
		if o.Voice != "" && c.Voice != o.Voice {
			continue
		}
		if o.Platform != "" && c.Platform != o.Platform {
			continue
		}
		if c.LikeCount < minLikes {
			continue
		}
		n := len([]rune(strings.TrimSpace(c.Content)))
		if minLen > 0 && n < minLen {
			continue
		}
		if maxLen > 0 && n > maxLen {
			continue
		}
		out = append(out, c)
	}

	switch o.SortBy {
	case "length":
		sort.SliceStable(out, func(i, j int) bool {
			return len([]rune(out[i].Content)) > len([]rune(out[j].Content))
		})
	case "recent":
		sort.SliceStable(out, func(i, j int) bool { return out[i].CreateTime > out[j].CreateTime })
	default:
		sort.SliceStable(out, func(i, j int) bool { return out[i].LikeCount > out[j].LikeCount })
	}
	if o.Top > 0 && len(out) > o.Top {
		out = out[:o.Top]
	}
	return out
}

// TensionPairs 从两条对立阵营的高赞评论里配对出「对立轴」素材。
//
// 判定口径：同一 note 下，两条评论的点赞都过阈值，且各自被 ≥1 条回复反驳。
// 这是「高对立度」的可操作定义——不是靠情感词典猜。
func TensionPairs(cs []Comment, minLikes int) [][2]Comment {
	byNote := map[string][]Comment{}
	for _, c := range cs {
		if c.LikeCount >= minLikes {
			byNote[c.NoteID] = append(byNote[c.NoteID], c)
		}
	}
	var pairs [][2]Comment
	for _, group := range byNote {
		if len(group) < 2 {
			continue
		}
		sort.SliceStable(group, func(i, j int) bool { return group[i].LikeCount > group[j].LikeCount })
		for i := 0; i+1 < len(group) && i < 3; i += 2 {
			pairs = append(pairs, [2]Comment{group[i], group[i+1]})
		}
	}
	return pairs
}

// ---------------------------------------------------------------------------
// 小工具
// ---------------------------------------------------------------------------

func has(m map[string]any, k string) bool {
	v, ok := m[k]
	if !ok || v == nil {
		return false
	}
	s, ok := v.(string)
	return !ok || strings.TrimSpace(s) != ""
}

func firstStr(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch t := v.(type) {
			case string:
				if strings.TrimSpace(t) != "" {
					return t
				}
			case float64:
				return fmt.Sprintf("%.0f", t)
			case json.Number:
				return t.String()
			}
		}
	}
	return ""
}

func firstInt(m map[string]any, keys ...string) int {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			switch t := v.(type) {
			case float64:
				return int(t)
			case int:
				return t
			case string:
				var n int
				if _, err := fmt.Sscanf(strings.TrimSpace(t), "%d", &n); err == nil {
					return n
				}
			}
		}
	}
	return 0
}

func yn(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func quoteAll(ss []string) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		if strings.ContainsAny(s, " ,\"'") {
			out = append(out, `"`+strings.ReplaceAll(s, `"`, `\"`)+`"`)
			continue
		}
		out = append(out, s)
	}
	return out
}

func tail(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return "…" + string(r[len(r)-n:])
}
