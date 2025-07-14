package utils

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorCode 错误码类型
type ErrorCode string

// 预定义错误码
const (
	ErrCodeUnknown          ErrorCode = "UNKNOWN"
	ErrCodeInvalidArgument  ErrorCode = "INVALID_ARGUMENT"
	ErrCodeNotFound         ErrorCode = "NOT_FOUND"
	ErrCodeAlreadyExists    ErrorCode = "ALREADY_EXISTS"
	ErrCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	ErrCodeResourceExhausted ErrorCode = "RESOURCE_EXHAUSTED"
	ErrCodeFailedPrecondition ErrorCode = "FAILED_PRECONDITION"
	ErrCodeAborted          ErrorCode = "ABORTED"
	ErrCodeOutOfRange       ErrorCode = "OUT_OF_RANGE"
	ErrCodeUnimplemented    ErrorCode = "UNIMPLEMENTED"
	ErrCodeInternal         ErrorCode = "INTERNAL"
	ErrCodeUnavailable      ErrorCode = "UNAVAILABLE"
	ErrCodeDataLoss         ErrorCode = "DATA_LOSS"
	ErrCodeUnauthenticated  ErrorCode = "UNAUTHENTICATED"
	ErrCodeCanceled         ErrorCode = "CANCELED"
	ErrCodeDeadlineExceeded ErrorCode = "DEADLINE_EXCEEDED"
)

// Error 扩展错误结构
type Error struct {
	Code       ErrorCode              // 错误码
	Message    string                 // 错误消息
	Details    map[string]interface{} // 详细信息
	Cause      error                  // 原因错误
	StackTrace string                 // 堆栈跟踪
}

// Error 实现error接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 实现错误解包
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithDetail 添加详细信息
func (e *Error) WithDetail(key string, value interface{}) *Error {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// NewError 创建新错误
func NewError(code ErrorCode, message string) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		StackTrace: getStackTrace(2),
	}
}

// NewErrorf 创建格式化错误
func NewErrorf(code ErrorCode, format string, args ...interface{}) *Error {
	return &Error{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		StackTrace: getStackTrace(2),
	}
}

// WrapError 包装错误
func WrapError(err error, code ErrorCode, message string) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Code:       code,
		Message:    message,
		Cause:      err,
		StackTrace: getStackTrace(2),
	}
}

// WrapErrorf 包装格式化错误
func WrapErrorf(err error, code ErrorCode, format string, args ...interface{}) *Error {
	if err == nil {
		return nil
	}
	return &Error{
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		Cause:      err,
		StackTrace: getStackTrace(2),
	}
}

// getStackTrace 获取堆栈跟踪
func getStackTrace(skip int) string {
	var pcs [32]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	
	var sb strings.Builder
	for {
		frame, more := frames.Next()
		sb.WriteString(fmt.Sprintf("%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line))
		if !more {
			break
		}
	}
	
	return sb.String()
}

// IsError 检查是否是指定错误码
func IsError(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	
	return e.Code == code
}

// GetErrorCode 获取错误码
func GetErrorCode(err error) ErrorCode {
	if err == nil {
		return ""
	}
	
	e, ok := err.(*Error)
	if !ok {
		return ErrCodeUnknown
	}
	
	return e.Code
}

// ErrorChain 错误链
type ErrorChain struct {
	errors []*Error
}

// Add 添加错误
func (ec *ErrorChain) Add(err *Error) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// AddError 添加普通错误
func (ec *ErrorChain) AddError(err error) {
	if err != nil {
		if e, ok := err.(*Error); ok {
			ec.errors = append(ec.errors, e)
		} else {
			ec.errors = append(ec.errors, WrapError(err, ErrCodeUnknown, err.Error()))
		}
	}
}

// HasErrors 是否有错误
func (ec *ErrorChain) HasErrors() bool {
	return len(ec.errors) > 0
}

// Error 实现error接口
func (ec *ErrorChain) Error() string {
	if len(ec.errors) == 0 {
		return ""
	}
	
	var messages []string
	for _, err := range ec.errors {
		messages = append(messages, err.Error())
	}
	
	return fmt.Sprintf("multiple errors: %s", strings.Join(messages, "; "))
}

// Errors 获取所有错误
func (ec *ErrorChain) Errors() []*Error {
	return ec.errors
}

// First 获取第一个错误
func (ec *ErrorChain) First() *Error {
	if len(ec.errors) > 0 {
		return ec.errors[0]
	}
	return nil
}