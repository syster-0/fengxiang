package okf

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sycglier/tv-opinion-atelier/internal/errx"
)

// SourceEntry 是 Source Map 里的一条来源记录。
type SourceEntry struct {
	Kind    string // crawl_jsonl / crawl_csv / transcript / manual / url
	Path    string // 相对 wiki 根或仓库根的路径
	Fetched string // 抓取/生成日期
	Note    string
}

// SourceMap 是某个概念的来源条目。
type SourceMap struct {
	Concept    string
	Distilled  string
	Sources    []SourceEntry
	Path       string
	ParseError string
}

// ReadSourceMap 读取概念的 Source Map 条目。
func (w *Wiki) ReadSourceMap(id string) (*SourceMap, error) {
	p := w.SourceMapPath(id)
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil, errx.Errorf(errx.KindNotFound, "okf.sourcemap", id, "缺 Source Map 条目 %s", p)
	}
	sm := &SourceMap{Path: p}
	inSources := false
	var cur *SourceEntry
	for _, line := range strings.Split(string(raw), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "concept:") {
			sm.Concept = strings.TrimSpace(strings.TrimPrefix(t, "concept:"))
			continue
		}
		if strings.HasPrefix(t, "distilled_at:") {
			sm.Distilled = strings.TrimSpace(strings.TrimPrefix(t, "distilled_at:"))
			continue
		}
		if t == "sources:" {
			inSources = true
			continue
		}
		if !inSources || !strings.HasPrefix(t, "- ") {
			continue
		}
		body := strings.TrimPrefix(t, "- ")
		if k, v, ok := strings.Cut(body, ":"); ok {
			cur = &SourceEntry{Kind: strings.TrimSpace(k), Path: strings.TrimSpace(v)}
			sm.Sources = append(sm.Sources, *cur)
			continue
		}
		switch {
		case strings.HasPrefix(body, "kind:"):
			cur = &SourceEntry{Kind: strings.TrimSpace(strings.TrimPrefix(body, "kind:"))}
			sm.Sources = append(sm.Sources, *cur)
		case strings.HasPrefix(body, "path:") && len(sm.Sources) > 0:
			sm.Sources[len(sm.Sources)-1].Path = strings.TrimSpace(strings.TrimPrefix(body, "path:"))
		case strings.HasPrefix(body, "fetched:") && len(sm.Sources) > 0:
			sm.Sources[len(sm.Sources)-1].Fetched = strings.TrimSpace(strings.TrimPrefix(body, "fetched:"))
		}
	}
	// 兼容 "  - kind: x" / "    path: y" 的缩进写法。
	if len(sm.Sources) == 0 {
		for _, line := range strings.Split(string(raw), "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "- kind:") {
				sm.Sources = append(sm.Sources, SourceEntry{Kind: strings.TrimSpace(strings.TrimPrefix(t, "- kind:"))})
			} else if strings.HasPrefix(t, "path:") {
				if n := len(sm.Sources); n > 0 {
					sm.Sources[n-1].Path = strings.TrimSpace(strings.TrimPrefix(t, "path:"))
				}
			} else if strings.HasPrefix(t, "fetched:") {
				if n := len(sm.Sources); n > 0 {
					sm.Sources[n-1].Fetched = strings.TrimSpace(strings.TrimPrefix(t, "fetched:"))
				}
			}
		}
	}
	if sm.Concept == "" {
		sm.ParseError = "缺 concept: 字段"
	}
	return sm, nil
}

// WriteSourceMap 写入概念的 Source Map 条目。
func (w *Wiki) WriteSourceMap(id, distilledAt string, entries []SourceEntry) error {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("concept: " + id + "\n")
	if distilledAt != "" {
		b.WriteString("distilled_at: " + distilledAt + "\n")
	}
	b.WriteString("sources:\n")
	for _, e := range entries {
		b.WriteString("- kind: " + e.Kind + "\n")
		b.WriteString("  path: " + e.Path + "\n")
		if e.Fetched != "" {
			b.WriteString("  fetched: " + e.Fetched + "\n")
		}
		if e.Note != "" {
			b.WriteString("  note: " + e.Note + "\n")
		}
	}
	b.WriteString("---\n\n")
	b.WriteString("# 来源：" + id + "\n\n")
	b.WriteString("双向索引：本条目 ↔ `concepts/" + id + ".md`。\n")
	p := w.SourceMapPath(id)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return errx.Wrap(errx.KindInternal, "okf.sourcemap", id, err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, []byte(b.String()), 0o644); err != nil {
		return errx.Wrap(errx.KindInternal, "okf.sourcemap", id, err)
	}
	if err := os.Rename(tmp, p); err != nil {
		return errx.Wrap(errx.KindInternal, "okf.sourcemap", id, err)
	}
	return nil
}

// ResolveRef 把 source_ref / Source Map 里的 path 解析到磁盘。
// 依次尝试：wiki 根、仓库根、以及相对 wiki 根的路径。
func (w *Wiki) ResolveRef(ref string) (string, bool) {
	if ref == "" {
		return "", false
	}
	cands := []string{
		filepath.Join(w.Root, filepath.FromSlash(ref)),
		filepath.Join(w.RepoRoot(), filepath.FromSlash(ref)),
		filepath.Join(filepath.Dir(w.Root), filepath.FromSlash(ref)),
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c, true
		}
	}
	return cands[0], false
}
