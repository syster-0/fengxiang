package okf

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sycglier/tv-opinion-atelier/internal/errx"
)

// Wiki 是 M3 知识库句柄。
type Wiki struct {
	Root string // wiki/ 目录绝对路径
	// Now 可注入，便于测试。为零值时用 time.Now。
	Now func() time.Time
}

// 目录与文件名常量。
const (
	DirConcepts  = "concepts"
	DirSourceMap = "source-map"
	DirTests     = "tests"
	FileIndex    = "index.md"
	FileLog      = "log.md"
	lockName     = ".lock"
)

// Init 初始化 wiki 骨架（幂等）。
func Init(root string) ([]string, error) {
	var created []string
	for _, d := range []string{"", DirConcepts, DirSourceMap, DirTests} {
		p := filepath.Join(root, d)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.MkdirAll(p, 0o755); err != nil {
				return created, errx.Wrap(errx.KindInternal, "okf.init", p, err)
			}
			created = append(created, p)
		}
	}
	idx := filepath.Join(root, FileIndex)
	if _, err := os.Stat(idx); os.IsNotExist(err) {
		if err := os.WriteFile(idx, []byte("# OKF 概念索引\n\n> 由 `tvop okf reindex` 编译生成，请勿手工编辑。\n\n_（空库）_\n"), 0o644); err != nil {
			return created, errx.Wrap(errx.KindInternal, "okf.init", idx, err)
		}
		created = append(created, idx)
	}
	lg := filepath.Join(root, FileLog)
	if _, err := os.Stat(lg); os.IsNotExist(err) {
		if err := os.WriteFile(lg, []byte("# OKF 操作日志（append-only）\n\n"), 0o644); err != nil {
			return created, errx.Wrap(errx.KindInternal, "okf.init", lg, err)
		}
		created = append(created, lg)
	}
	return created, nil
}

// Open 打开一个已存在的 wiki。
func Open(root string) (*Wiki, error) {
	if _, err := os.Stat(filepath.Join(root, DirConcepts)); err != nil {
		return nil, errx.Errorf(errx.KindNotFound, "okf.open", root,
			"不是 wiki 目录（缺 %s/），先跑 `tvop init`", DirConcepts)
	}
	return &Wiki{Root: root}, nil
}

func (w *Wiki) now() time.Time {
	if w.Now != nil {
		return w.Now()
	}
	return time.Now()
}

// ConceptsRoot 返回 concepts/ 绝对路径。
func (w *Wiki) ConceptsRoot() string { return filepath.Join(w.Root, DirConcepts) }

// ConceptPath 由概念 ID 得到磁盘路径。
func (w *Wiki) ConceptPath(id string) (string, error) {
	if id == "" || strings.Contains(id, "..") || filepath.IsAbs(id) {
		return "", errx.Errorf(errx.KindInvalid, "okf.path", id, "非法概念 ID")
	}
	return filepath.Join(w.ConceptsRoot(), filepath.FromSlash(id)+".md"), nil
}

// SourceMapPath 返回概念对应的 Source Map 条目路径。
func (w *Wiki) SourceMapPath(id string) string {
	return filepath.Join(w.Root, DirSourceMap, filepath.FromSlash(id)+".md")
}

// TestPath 返回概念对应的 test-prompts.json 路径。
func (w *Wiki) TestPath(id string) string {
	return filepath.Join(w.Root, DirTests, filepath.FromSlash(id)+".json")
}

// RepoRoot 向上查找 go.mod 锚点得到仓库根；找不到则退回 wiki 的父目录。
func (w *Wiki) RepoRoot() string {
	dir := filepath.Dir(filepath.Clean(w.Root))
	for i := 0; i < 6 && dir != "" && dir != "/" && dir != "."; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return filepath.Dir(filepath.Clean(w.Root))
}

// List 遍历库内全部概念。
func (w *Wiki) List() ([]*Concept, error) {
	root := w.ConceptsRoot()
	var out []*Concept
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && p != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		c, cerr := w.load(p)
		if cerr != nil {
			// 解析失败的概念也要带回，让校验器报出来，而不是静默跳过。
			id, _ := IDFromPath(root, p)
			out = append(out, &Concept{
				ID: id, Path: p, Top: map[string]string{},
				Body:    "[解析失败] " + cerr.Error(),
				Atelier: Atelier{UseCount: 0},
			})
			return nil
		}
		out = append(out, c)
		return nil
	})
	if err != nil {
		return nil, errx.Wrap(errx.KindInternal, "okf.list", root, err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// Get 读取单个概念。
func (w *Wiki) Get(id string) (*Concept, error) {
	p, err := w.ConceptPath(id)
	if err != nil {
		return nil, err
	}
	if _, serr := os.Stat(p); serr != nil {
		return nil, errx.Errorf(errx.KindNotFound, "okf.get", id, "概念不存在")
	}
	return w.load(p)
}

func (w *Wiki) load(p string) (*Concept, error) {
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, errx.Wrap(errx.KindInternal, "okf.load", p, err)
	}
	head, body, err := splitFrontmatter(string(raw))
	if err != nil {
		return nil, err
	}
	top, at, err := parseFrontmatter(head)
	if err != nil {
		return nil, err
	}
	id, err := IDFromPath(w.ConceptsRoot(), p)
	if err != nil {
		return nil, err
	}
	return &Concept{
		ID:       id,
		Path:     p,
		Type:     top["type"],
		Title:    top["title"],
		Voice:    top["voice"],
		Platform: top["platform"],
		Command:  top["command"],
		Top:      top,
		Atelier:  at,
		Body:     body,
	}, nil
}

// Render 把概念渲染成文件内容（不含写入）。
func Render(c *Concept) string {
	var b strings.Builder
	b.WriteString(renderFrontmatter(c.Top, c.Atelier))
	b.WriteString("\n")
	body := strings.TrimRight(c.Body, "\n")
	b.WriteString(body)
	b.WriteString("\n")
	return b.String()
}

// Put 执行写路径四步：写锁 → 校验 → 原子提交 → log.md 追加。
//
// 读先于写：若磁盘上已有该概念且 use_count 比入参更高，说明另有写入者，
// 返回 conflict 而不是覆盖（防陈旧 ID 覆盖手动修改）。
func (w *Wiki) Put(c *Concept, op, note string) error {
	if c.Top == nil {
		c.Top = map[string]string{}
	}
	c.Top["type"] = c.Type
	c.Top["title"] = c.Title
	if c.Voice != "" {
		c.Top["voice"] = c.Voice
	}
	if c.Platform != "" {
		c.Top["platform"] = c.Platform
	}
	if c.Command != "" {
		c.Top["command"] = c.Command
	}
	if c.Atelier.Updated == "" {
		c.Atelier.Updated = w.now().Format("2006-01-02")
	}

	return w.withLock(func() error {
		// 第二步：校验
		if issues := ValidateStructural(c); len(issues) > 0 {
			return errx.Errorf(errx.KindInvalid, "okf.put", c.ID,
				"结构校验未通过：%s", strings.Join(issues, "；"))
		}
		p, err := w.ConceptPath(c.ID)
		if err != nil {
			return err
		}
		if old, oerr := w.load(p); oerr == nil && old.Type != "" {
			if old.Atelier.UseCount > c.Atelier.UseCount {
				return errx.Errorf(errx.KindConflict, "okf.put", c.ID,
					"磁盘上的 use_count=%d 高于入参 %d，疑似陈旧写入；请先重读",
					old.Atelier.UseCount, c.Atelier.UseCount)
			}
		}
		// 第三步：原子提交
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return errx.Wrap(errx.KindInternal, "okf.put", c.ID, err)
		}
		tmp := p + ".tmp"
		if err := os.WriteFile(tmp, []byte(Render(c)), 0o644); err != nil {
			return errx.Wrap(errx.KindInternal, "okf.put", c.ID, err)
		}
		if err := os.Rename(tmp, p); err != nil {
			return errx.Wrap(errx.KindInternal, "okf.put", c.ID, err)
		}
		c.Path = p
		// 第四步：追加日志
		return w.AppendLog(op, c.ID, note)
	})
}

// AppendLog 追加一行操作日志。append-only，永不覆盖。
func (w *Wiki) AppendLog(op, subject, note string) error {
	line := fmt.Sprintf("- %s | %s | %s | %s\n",
		w.now().Format(time.RFC3339), op, subject, strings.ReplaceAll(note, "\n", " "))
	f, err := os.OpenFile(filepath.Join(w.Root, FileLog), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return errx.Wrap(errx.KindInternal, "okf.log", subject, err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		return errx.Wrap(errx.KindInternal, "okf.log", subject, err)
	}
	return nil
}

// LogLines 读取日志正文行（去掉标题与空行）。
func (w *Wiki) LogLines() ([]string, error) {
	raw, err := os.ReadFile(filepath.Join(w.Root, FileLog))
	if err != nil {
		return nil, errx.Wrap(errx.KindNotFound, "okf.log", w.Root, err)
	}
	var out []string
	for _, l := range strings.Split(string(raw), "\n") {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "- ") {
			out = append(out, l)
		}
	}
	return out, nil
}

// Reindex 依据实际文件重编译 index.md（生成目录只读）。
func (w *Wiki) Reindex() (int, error) {
	cs, err := w.List()
	if err != nil {
		return 0, err
	}
	var b strings.Builder
	b.WriteString("# OKF 概念索引\n\n")
	b.WriteString("> 由 `tvop okf reindex` 编译生成，请勿手工编辑。\n\n")
	b.WriteString("| ID | type | title | voice | tier | verdict | weight | humanness | risk | usable |\n")
	b.WriteString("|---|---|---|---|---|---|---|---|---|---|\n")
	for _, c := range cs {
		usable := "—"
		if c.Usable() {
			usable = "✓"
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %.2f | %.2f | %.2f | %s |\n",
			md(c.ID), md(c.Type), md(c.Title), md(c.Voice), md(c.Atelier.Tier),
			md(c.Atelier.Verdict), c.Atelier.Weight, c.Atelier.Humanness, c.Atelier.Risk, usable)
	}
	if len(cs) == 0 {
		b.WriteString("\n_（空库）_\n")
	}
	tmp := filepath.Join(w.Root, FileIndex) + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return 0, errx.Wrap(errx.KindInternal, "okf.reindex", w.Root, err)
	}
	if err := os.Rename(tmp, filepath.Join(w.Root, FileIndex)); err != nil {
		return 0, errx.Wrap(errx.KindInternal, "okf.reindex", w.Root, err)
	}
	return len(cs), nil
}

// IndexCount 返回 index.md 表格中的数据行数（用于一致性比对）。
func (w *Wiki) IndexCount() (int, error) {
	raw, err := os.ReadFile(filepath.Join(w.Root, FileIndex))
	if err != nil {
		return 0, errx.Wrap(errx.KindNotFound, "okf.index", w.Root, err)
	}
	n := 0
	for _, l := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(l, "| ") && !strings.Contains(l, "|---") && !strings.HasPrefix(l, "| ID ") {
			n++
		}
	}
	return n, nil
}

// withLock 实现跨进程写锁：独占创建 .lock，带超时与陈旧锁回收。
func (w *Wiki) withLock(fn func() error) error {
	if err := os.MkdirAll(w.Root, 0o755); err != nil {
		return errx.Wrap(errx.KindInternal, "okf.lock", w.Root, err)
	}
	lp := filepath.Join(w.Root, lockName)
	deadline := time.Now().Add(5 * time.Second)
	for {
		f, err := os.OpenFile(lp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			break
		}
		if !os.IsExist(err) {
			return errx.Wrap(errx.KindInternal, "okf.lock", lp, err)
		}
		if st, serr := os.Stat(lp); serr == nil && time.Since(st.ModTime()) > 60*time.Second {
			_ = os.Remove(lp) // 陈旧锁回收
			continue
		}
		if time.Now().After(deadline) {
			return errx.New(errx.KindConflict, "okf.lock", lp, "取锁超时，另一个写入者仍在持锁")
		}
		time.Sleep(25 * time.Millisecond)
	}
	defer os.Remove(lp)
	return fn()
}

func md(s string) string {
	if s == "" {
		return "—"
	}
	return strings.ReplaceAll(s, "|", "\\|")
}
