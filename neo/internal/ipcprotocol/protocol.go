package ipcprotocol

import (
	"encoding/json"
	"fmt"
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

// ipcTask 实现types.Task接口
// 用于包装IPC请求任务并提交到工作池
// 实现说明：2025.06.17新增，解决接口不匹配问题

type ipcTask struct {
	taskID   string
	req      *types.Request
	registry types.ServiceRegistry
}

// ID 返回任务ID，实现types.Task接口
func (t *ipcTask) ID() string {
	return t.taskID
}

// Execute 执行任务，实现types.Task接口
func (t *ipcTask) Execute() (interface{}, error) {
	// 从服务注册表查找服务
	handler, exists := t.registry.GetHandler(t.req.Service)
	if !exists {
		return nil, fmt.Errorf("service not found: %s", t.req.Service)
	}

	// 调用服务方法
	resp, err := handler.Handle(t.req)
	if err != nil {
		return nil, err
	}

	return resp, nil
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
		},
		nil
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
		},
		nil
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
	var frame types.MessageFrame
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
		task := &ipcTask{
			taskID:   req.RequestID,
			req:      &req,
			registry: registry,
		}

		// 提交任务并获取结果通道
		resultChan := workerPool.Submit(task)

		// 等待结果或超时
		select {
		case result := <-resultChan:
			if result.Error != nil {
				resp := NewErrorResponse(req.RequestID, types.ErrorCodeInternalError, result.Error.Error())
				return marshalResponse(resp)
			}

			// 将结果转换为Response类型
			response, ok := result.Result.(*types.Response)
			if !ok {
				resp := NewErrorResponse(req.RequestID, types.ErrorCodeInternalError, "invalid response type")
				return marshalResponse(resp)
			}
			return marshalResponse(response)

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
	frame := types.MessageFrame{
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
