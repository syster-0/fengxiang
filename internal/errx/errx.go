// Package errx 提供类型化错误。
//
// 纪律（来自体系 go-runtime 规范）：
//   - 错误必须带上下文包装，禁止裸 errors.New 冒泡；
//   - paid_* 类错误（消耗配额/触发不可逆外部动作）永不自动重试；
//   - 校验类错误（Invalid）必须携带可定位的字段/路径，便于回炉。
package errx

import (
	"errors"
	"fmt"
)

// Kind 是错误分类。分类决定调用方是否允许重试。
type Kind string

const (
	// KindInvalid 输入/数据不合法。不可重试，必须回炉修正。
	KindInvalid Kind = "invalid"
	// KindNotFound 资源不存在。不可重试，通常触发补投喂。
	KindNotFound Kind = "not_found"
	// KindConflict 状态冲突（如陈旧 ID、并发写）。重读后可重试一次。
	KindConflict Kind = "conflict"
	// KindUnavailable 依赖不可用（浏览器、解释器、网络）。可退避重试。
	KindUnavailable Kind = "unavailable"
	// KindPaid 付费/不可逆外部动作失败。永不自动重试。
	KindPaid Kind = "paid_failure"
	// KindInternal 内部缺陷。不可自动重试。
	KindInternal Kind = "internal"
)

// Error 是类型化错误。
type Error struct {
	Kind    Kind
	Op      string // 操作名，如 "okf.validate"
	Subject string // 出错对象，如概念 ID 或文件路径
	Msg     string
	Err     error
}

func (e *Error) Error() string {
	var b []byte
	b = append(b, '[')
	b = append(b, string(e.Kind)...)
	b = append(b, ']')
	if e.Op != "" {
		b = append(b, ' ')
		b = append(b, e.Op...)
	}
	if e.Subject != "" {
		b = append(b, '(')
		b = append(b, e.Subject...)
		b = append(b, ')')
	}
	if e.Msg != "" {
		b = append(b, ": "...)
		b = append(b, e.Msg...)
	}
	if e.Err != nil {
		b = append(b, ": "...)
		b = append(b, e.Err.Error()...)
	}
	return string(b)
}

func (e *Error) Unwrap() error { return e.Err }

// New 构造一个类型化错误。
func New(kind Kind, op, subject, msg string) *Error {
	return &Error{Kind: kind, Op: op, Subject: subject, Msg: msg}
}

// Wrap 包装下方错误并标注类型。
func Wrap(kind Kind, op, subject string, err error) *Error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Op: op, Subject: subject, Err: err}
}

// Errorf 构造带格式化消息的类型化错误。
func Errorf(kind Kind, op, subject, format string, args ...any) *Error {
	return &Error{Kind: kind, Op: op, Subject: subject, Msg: fmt.Sprintf(format, args...)}
}

// KindOf 提取错误分类，未分类的归为 KindInternal。
func KindOf(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindInternal
}

// Retryable 报告该错误是否允许自动重试。
// 付费类与校验类永不自动重试——这是硬纪律。
func Retryable(err error) bool {
	switch KindOf(err) {
	case KindConflict, KindUnavailable:
		return true
	default:
		return false
	}
}

// IsKind 判断错误是否属于给定分类。
func IsKind(err error, kind Kind) bool { return KindOf(err) == kind }
