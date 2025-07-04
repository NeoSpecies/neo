package types

import (
	"encoding/json"
	"fmt"
)

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

// ipcTask 实现Task接口
// 用于包装IPC请求任务并提交到工作池
// IPC任务结构体（重命名为大写开头以导出）
type IPCTask struct {
	TaskID   string
	Req      *Request
	Registry *ServiceRegistry
}

// ID 返回任务ID，实现Task接口
func (t *IPCTask) ID() string {
	return t.TaskID
}

// Execute 执行任务，实现Task接口
func (t *IPCTask) Execute() (interface{}, error) {
	// 从服务注册表查找服务
	handler, exists := t.Registry.GetHandler(t.Req.Service)
	if !exists {
		return nil, fmt.Errorf("service not found: %s", t.Req.Service)
	}

	// 调用服务方法
	resp, err := handler.Handle(t.Req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
