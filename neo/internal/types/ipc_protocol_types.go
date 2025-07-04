// neo/neo/internal/types/ipc_protocol_types.go
package types

import (
	"encoding/json"
	"time"
)

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

// ProcessMessage 处理IPC消息
// 修改函数参数为指针类型
func ProcessMessage(data []byte, registry *ServiceRegistry, workerPool WorkerPool) ([]byte, error) {
	// 解析消息帧
	var frame MessageFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		resp := NewErrorResponse("", ErrorCodeInvalidRequest, "Invalid message format")
		return marshalResponse(resp)
	}

	// 处理请求类型消息
	if frame.Type == MessageTypeRequest {
		var req Request
		if err := json.Unmarshal(frame.Payload, &req); err != nil {
			resp := NewErrorResponse("", ErrorCodeInvalidRequest, "Invalid request payload")
			return marshalResponse(resp)
		}

		// 提交任务到工作池处理
		task := &IPCTask{
			TaskID:   req.RequestID,
			Req:      &req,
			Registry: registry, // 修改为指针类型
		}

		// 提交任务并获取结果通道
		resultChan := workerPool.Submit(task)

		// 等待结果或超时
		select {
		case result := <-resultChan:
			if result.Error != nil {
				resp := NewErrorResponse(req.RequestID, ErrorCodeInternalError, result.Error.Error())
				return marshalResponse(resp)
			}

			// 将结果转换为Response类型
			response, ok := result.Result.(*Response)
			if !ok {
				resp := NewErrorResponse(req.RequestID, ErrorCodeInternalError, "invalid response type")
				return marshalResponse(resp)
			}
			return marshalResponse(response)

		case <-time.After(time.Duration(req.Timeout) * time.Millisecond):
			resp := NewErrorResponse(req.RequestID, ErrorCodeTimeout, "Request timeout")
			return marshalResponse(resp)
		}
	}

	// 未知消息类型
	resp := NewErrorResponse("", ErrorCodeInvalidRequest, "Unknown message type")
	return marshalResponse(resp)
}

// 辅助函数：序列化响应
func marshalResponse(resp *Response) ([]byte, error) {
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
