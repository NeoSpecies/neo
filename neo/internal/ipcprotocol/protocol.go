package ipcprotocol

import (
	"encoding/json"
	"neo/internal/types"
	"time"
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

// 创建新请求
func NewRequest(service, method string, params interface{}) (*types.Request, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	return &types.Request{
		RequestID: NewRequestID(),
		Service:   service,
		Method:    method,
		Params:    paramsJSON,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// 创建新响应
func NewResponse(requestID string, data interface{}) (*types.Response, error) {
	var dataJSON json.RawMessage
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		dataJSON = json.RawMessage(jsonData)
	}

	return &types.Response{
		RequestID: requestID,
		Code:      types.ErrorCodeSuccess,
		Data:      dataJSON,
		Timestamp: time.Now().UnixMilli(),
	}, nil
}

// 创建错误响应
func NewErrorResponse(requestID string, code types.ErrorCode, message string) *types.Response {
	return &types.Response{
		RequestID: requestID,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ProcessMessage 处理IPC消息
func ProcessMessage(data []byte, registry types.ServiceRegistry, workerPool types.WorkerPool) ([]byte, error) {
	// 解析消息帧
	var frame MessageFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		resp := NewErrorResponse("", types.ErrorCodeInvalidRequest, "Invalid message format")
		return marshalResponse(resp)
	}

	// 处理请求类型消息
	if frame.Type == MessageTypeRequest {
		var req types.Request
		if err := json.Unmarshal(frame.Payload, &req); err != nil {
			resp := NewErrorResponse("", types.ErrorCodeInvalidRequest, "Invalid request payload")
			return marshalResponse(resp)
		}

		// 提交任务到工作池处理
		resultChan := make(chan *types.Response, 1)
		err := workerPool.Submit(func() {
			// 从服务注册表查找服务
			handler, exists := registry.GetHandler(req.Service)
			if !exists {
				resultChan <- NewErrorResponse(req.RequestID, types.ErrorCodeServiceNotFound, "Service not found")
				return
			}

			// 调用服务方法
			resp, err := handler.Handle(&req)
			if err != nil {
				resultChan <- NewErrorResponse(req.RequestID, types.ErrorCodeInternalError, err.Error())
				return
			}

			resultChan <- resp
		})

		if err != nil {
			resp := NewErrorResponse(req.RequestID, types.ErrorCodeInternalError, "Failed to submit task")
			return marshalResponse(resp)
		}

		// 等待结果或超时
		select {
		case resp := <-resultChan:
			return marshalResponse(resp)
		case <-time.After(time.Duration(req.Timeout) * time.Millisecond):
			resp := NewErrorResponse(req.RequestID, types.ErrorCodeTimeout, "Request timeout")
			return marshalResponse(resp)
		}
	}

	// 未知消息类型
	resp := NewErrorResponse("", types.ErrorCodeInvalidRequest, "Unknown message type")
	return marshalResponse(resp)
}

// 辅助函数：序列化响应
func marshalResponse(resp *types.Response) ([]byte, error) {
	frame := MessageFrame{
		Type:    MessageTypeResponse,
		Payload: []byte(json.RawMessage(resp.Data)),
	}
	return json.Marshal(frame)
}

// NewRequestID 生成请求ID
func NewRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

// 生成随机字符串
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
