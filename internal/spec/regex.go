package spec

import (
	"regexp"
	"sync"
)

var reCache sync.Map // pattern -> *regexp.Regexp

// Compile 编译并缓存正则。模式非法时返回错误（不 panic，交由调用方转类型化错误）。
func Compile(pattern string) (*regexp.Regexp, error) {
	if v, ok := reCache.Load(pattern); ok {
		return v.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	reCache.Store(pattern, re)
	return re, nil
}

// MustCompile 编译并缓存正则；模式非法时 panic（仅供内置常量表使用）。
func MustCompile(pattern string) *regexp.Regexp {
	re, err := Compile(pattern)
	if err != nil {
		panic("spec: invalid builtin pattern " + pattern + ": " + err.Error())
	}
	return re
}

// Match 报告 text 是否命中 pattern。
func Match(pattern, text string) bool {
	re, err := Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(text)
}

// CountAll 统计 text 中 pattern 的全部命中数（非重叠）。
func CountAll(pattern, text string) int {
	re, err := Compile(pattern)
	if err != nil {
		return 0
	}
	return len(re.FindAllStringIndex(text, -1))
}

// regexpMatch 是包内别名，避免在 spec.go 顶部引入 regexp 依赖。
func regexpMatch(pattern, text string) bool { return Match(pattern, text) }
