package types

import (
	"errors"
	"fmt"
)

// 错误类型定义
const (
	ErrorTypeReadFailed       = "read_failed"
	ErrorTypeWriteFailed      = "write_failed"
	ErrorTypeInvalidData      = "invalid_data"
	ErrorTypeConnection       = "connection_error"
	ErrorTypeConnectionClosed = "connection_closed"
)

// ConnectionError 连接错误结构体
type ConnectionError struct {
	Type    string
	Message string
	Err     error
}

// 实现error接口
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

var (
	ErrInvalidMessage      = errors.New("invalid message format")
	ErrChecksumMismatch    = errors.New("checksum mismatch")
	ErrMessageTooLarge     = errors.New("message size exceeds limit")
	ErrCompressionFailed   = errors.New("compression failed")
	ErrDecompressionFailed = errors.New("decompression failed")
	ErrInvalidCompression  = errors.New("invalid compression type")
	ErrMaxRetryExceeded    = errors.New("max retry count exceeded")
	ErrNetworkError        = errors.New("network error")
	ErrBusinessError       = errors.New("business logic error")
	ErrTimeout             = errors.New("request timeout")
	ErrCircuitBreakerOpen  = errors.New("circuit breaker is open")
)
