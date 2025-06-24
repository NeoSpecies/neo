package ipcprotocol

import (
	"encoding/json"
	"time"
)

// 消息类型定义
const (
	MessageTypeRequest  = "request"
	MessageTypeResponse = "response"
	MessageTypeError    = "error"
	MessageTypeEvent    = "event"
)

// 错误码定义
type ErrorCode string

const (
	ErrorCodeSuccess       ErrorCode = "SUCCESS"
	ErrorCodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	ErrorCodeServiceNotFound ErrorCode = "SERVICE_NOT_FOUND"
	ErrorCodeMethodNotFound ErrorCode = "METHOD_NOT_FOUND"
	ErrorCodeInternalError  ErrorCode = "INTERNAL_ERROR"
	ErrorCodeTimeout        ErrorCode = "TIMEOUT"
	ErrorCodePermissionDenied ErrorCode = "PERMISSION_DENIED"
)

// 请求结构
type Request struct {
	RequestID   string          `json:"request_id"`
	Service     string          `json:"service"`
	Method      string          `json:"method"`
	Params      json.RawMessage `json:"params"`
	Timestamp   int64           `json:"timestamp"`
	Timeout     int             `json:"timeout,omitempty"` // 毫秒
}

// 响应结构
type Response struct {
	RequestID   string          `json:"request_id"`
	Code        ErrorCode       `json:"code"`
	Message     string          `json:"message,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
	Timestamp   int64           `json:"timestamp"`
}

// 消息帧结构
type MessageFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// 创建新请求
func NewRequest(service, method string, params interface{}) (*Request, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	return &Request{
		RequestID: NewRequestID(),
		Service:   service,
		Method:    method,
		Params:    paramsJSON,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// 创建新响应
func NewResponse(requestID string, data interface{}) (*Response, error) {
	var dataJSON json.RawMessage
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		dataJSON = json.RawMessage(jsonData)
	}

	return &Response{
		RequestID: requestID,
		Code:      ErrorCodeSuccess,
		Data:      dataJSON,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// 创建错误响应
func NewErrorResponse(requestID string, code ErrorCode, message string) *Response {
	return &Response{
		RequestID: requestID,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}
}
