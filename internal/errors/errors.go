// Package errors 提供跨模块共用的错误语义。
// 各模块通过 errors.Is() 匹配这些基础错误，在 handler 中映射为 HTTP 状态码。
package errors

import "errors"

// 共用错误语义。模块特有错误应包装这些基础错误，或直接定义为模块级变量。
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidInput = errors.New("invalid input")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
	ErrInternal     = errors.New("internal error")
)
