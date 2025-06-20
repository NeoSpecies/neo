package protocol

import "errors"

var (
	ErrInvalidMessage       = errors.New("invalid message format")
	ErrChecksumMismatch     = errors.New("checksum mismatch")
	ErrMessageTooLarge      = errors.New("message size exceeds limit")
	ErrCompressionFailed    = errors.New("compression failed")
	ErrDecompressionFailed  = errors.New("decompression failed")
	ErrInvalidCompression   = errors.New("invalid compression type")
	ErrMaxRetryExceeded     = errors.New("max retry count exceeded")
	// 新增错误分类
	ErrNetworkError         = errors.New("network error")
	ErrBusinessError        = errors.New("business logic error")
	ErrTimeout              = errors.New("request timeout")
	ErrCircuitBreakerOpen   = errors.New("circuit breaker is open") // 熔断错误
)
