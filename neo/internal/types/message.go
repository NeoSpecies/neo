package types

import "encoding/json"

// 请求结构
type Request struct {
	RequestID string          `json:"request_id"`
	Service   string          `json:"service"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	Timestamp int64           `json:"timestamp"`
	Timeout   int             `json:"timeout,omitempty"` // 毫秒
}

// 响应结构
type Response struct {
	RequestID string          `json:"request_id"`
	Code      ErrorCode       `json:"code"`
	Message   string          `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// 错误码定义
type ErrorCode string

const (
	ErrorCodeSuccess          ErrorCode = "SUCCESS"
	ErrorCodeInvalidRequest   ErrorCode = "INVALID_REQUEST"
	ErrorCodeServiceNotFound  ErrorCode = "SERVICE_NOT_FOUND"
	ErrorCodeMethodNotFound   ErrorCode = "METHOD_NOT_FOUND"
	ErrorCodeInternalError    ErrorCode = "INTERNAL_ERROR"
	ErrorCodeTimeout          ErrorCode = "TIMEOUT"
	ErrorCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
)

// 消息类型定义
const (
	MessageTypeRequest  = "request"
	MessageTypeResponse = "response"
	MessageTypeError    = "error"
	MessageTypeEvent    = "event"
)

// 消息帧结构
type MessageFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
