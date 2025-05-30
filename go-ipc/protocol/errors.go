package protocol

import "errors"

var (
	// ErrInvalidMessage 无效的消息格式
	ErrInvalidMessage = errors.New("invalid message format")

	// ErrChecksumMismatch 校验和不匹配
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrMessageTooLarge 消息过大
	ErrMessageTooLarge = errors.New("message size exceeds limit")

	// ErrCompressionFailed 压缩失败
	ErrCompressionFailed = errors.New("compression failed")

	// ErrDecompressionFailed 解压失败
	ErrDecompressionFailed = errors.New("decompression failed")

	// ErrInvalidCompression 无效的压缩类型
	ErrInvalidCompression = errors.New("invalid compression type")

	// ErrMaxRetryExceeded 超过最大重试次数
	ErrMaxRetryExceeded = errors.New("max retry count exceeded")
) 