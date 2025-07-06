/*
 * 描述: 定义IPC通信相关的消息结构、请求/响应类型、错误码和任务处理接口
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"encoding/json"
	"fmt"
)

// Request IPC请求结构体
// 封装服务调用的请求信息，包括目标服务、方法、参数和超时设置
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | RequestID      | string           | 请求唯一标识符                    |
// | Service        | string           | 目标服务名称                      |
// | Method         | string           | 调用的方法名称                    |
// | Params         | json.RawMessage  | 请求参数（JSON格式原始数据）      |
// | Timestamp      | int64            | 请求时间戳（毫秒级Unix时间）      |
// | Timeout        | int              | 超时时间（毫秒，可选）            |
// +----------------+------------------+-----------------------------------+
type Request struct {
	RequestID string          `json:"request_id"`
	Service   string          `json:"service"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	Timestamp int64           `json:"timestamp"`
	Timeout   int             `json:"timeout,omitempty"` // 毫秒
}

// Response IPC响应结构体
// 封装服务调用的响应结果，包括状态码、错误信息和返回数据
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | RequestID      | string           | 关联的请求唯一标识符              |
// | Code           | ErrorCode        | 响应状态码                        |
// | Message        | string           | 错误描述信息（可选）              |
// | Data           | json.RawMessage  | 响应数据（JSON格式原始数据，可选）|
// | Timestamp      | int64            | 响应时间戳（毫秒级Unix时间）      |
// +----------------+------------------+-----------------------------------+
type Response struct {
	RequestID string          `json:"request_id"`
	Code      ErrorCode       `json:"code"`
	Message   string          `json:"message,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// ErrorCode 错误码类型定义
// 表示IPC通信过程中的各种错误状态
// +---------------------------+-----------------------------------+
// | 常量名                    | 描述                              |
// +---------------------------+-----------------------------------+
// | ErrorCodeSuccess          | 成功状态                          |
// | ErrorCodeInvalidRequest   | 请求格式无效                      |
// | ErrorCodeServiceNotFound  | 服务未找到                        |
// | ErrorCodeMethodNotFound   | 方法未找到                        |
// | ErrorCodeInternalError    | 内部错误                          |
// | ErrorCodeTimeout          | 请求超时                          |
// | ErrorCodePermissionDenied | 权限拒绝                          |
// +---------------------------+-----------------------------------+
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

// MessageType 消息类型定义
// 表示IPC通信中的不同消息种类
// +------------------------+-----------------------------------+
// | 常量名                 | 描述                              |
// +------------------------+-----------------------------------+
// | MessageTypeRequest     | 请求类型消息                      |
// | MessageTypeResponse    | 响应类型消息                      |
// | MessageTypeError       | 错误类型消息                      |
// | MessageTypeEvent       | 事件类型消息                      |
// +------------------------+-----------------------------------+
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeError    MessageType = "error"
	MessageTypeEvent    MessageType = "event"
)

// MessageFrame 消息帧结构体
// IPC通信的基本数据单元，包含消息类型和负载数据
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Type           | string           | 消息类型（对应MessageType常量）   |
// | Payload        | json.RawMessage  | 消息负载（JSON格式原始数据）      |
// +----------------+------------------+-----------------------------------+
type MessageFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// IPCTask IPC任务结构体
// 实现Task接口，用于包装IPC请求任务并提交到工作池处理
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | TaskID         | string           | 任务唯一标识符                    |
// | Req            | *Request         | 请求对象指针                      |
// | Registry       | *ServiceRegistry | 服务注册表指针                    |
// +----------------+------------------+-----------------------------------+
type IPCTask struct {
	TaskID   string
	Req      *Request
	Registry *ServiceRegistry
}

// ID 返回任务ID，实现Task接口
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | string         | 任务唯一标识符                    |
// +----------------+-----------------------------------+
func (t *IPCTask) ID() string {
	return t.TaskID
}

// Execute 执行任务，实现Task接口
// 从服务注册表查找处理程序并执行请求
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | interface{}    | 任务执行结果（通常为*Response）   |
// | error          | 执行过程中发生的错误              |
// +----------------+-----------------------------------+
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
