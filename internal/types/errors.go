package types

import "errors"

// 定义包级别的错误
var (
	// Message 相关错误
	ErrInvalidMessageID   = errors.New("invalid message id")
	ErrInvalidMessageType = errors.New("invalid message type")

	// Request 相关错误
	ErrInvalidRequestID = errors.New("invalid request id")
	ErrInvalidService   = errors.New("invalid service name")
	ErrInvalidMethod    = errors.New("invalid method name")

	// Response 相关错误
	ErrInvalidResponseID = errors.New("invalid response id")
	ErrInvalidStatus     = errors.New("invalid status code")

	// 序列化相关错误
	ErrSerializationFailed   = errors.New("serialization failed")
	ErrDeserializationFailed = errors.New("deserialization failed")
)