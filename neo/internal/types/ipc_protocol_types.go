/*
 * 描述: 定义IPC协议相关的请求、响应处理函数及辅助方法，包括请求创建、响应生成和消息处理逻辑
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"encoding/json"
	"time"
)

// NewRequest 创建新的IPC请求实例
// 生成包含服务名、方法名、参数和时间戳的请求对象
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | service        | 目标服务名称                      |
// | method         | 调用的方法名称                    |
// | params         | 请求参数（将被序列化为JSON）      |
// | 返回值         | 新创建的Request实例和可能的错误   |
// +----------------+-----------------------------------+
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
	}, nil // 修正：添加nil作为第二个返回值并移除末尾的()
}

// NewResponse 创建新的成功响应实例
// 根据请求ID和返回数据生成标准响应对象
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | requestID      | 关联的请求ID                      |
// | data           | 响应数据（将被序列化为JSON）      |
// | 返回值         | 新创建的Response实例和可能的错误  |
// +----------------+-----------------------------------+
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
	}, nil // 修正：添加nil作为第二个返回值并移除末尾的()
}

// NewErrorResponse 创建新的错误响应实例
// 生成包含错误代码和消息的响应对象
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | requestID      | 关联的请求ID（可为空）            |
// | code           | 错误代码                          |
// | message        | 错误描述信息                      |
// | 返回值         | 新创建的Response实例              |
// +----------------+-----------------------------------+
func NewErrorResponse(requestID string, code ErrorCode, message string) *Response {
	return &Response{
		RequestID: requestID,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UnixMilli(),
	}
}

// ProcessMessage 处理IPC消息
// 解析消息帧，根据消息类型分发处理并返回响应
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 原始消息数据                      |
// | registry       | 服务注册表实例                    |
// | workerPool     | 工作池实例                        |
// | 返回值         | 序列化后的响应数据和可能的错误    |
// +----------------+-----------------------------------+
func ProcessMessage(data []byte, registry *ServiceRegistry, workerPool WorkerPool) ([]byte, error) {
	// 解析消息帧
	var frame MessageFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		resp := NewErrorResponse("", ErrorCodeInvalidRequest, "Invalid message format")
		return marshalResponse(resp)
	}

	// 处理请求类型消息
	if frame.Type == string(MessageTypeRequest) {
		var req Request
		if err := json.Unmarshal(frame.Payload, &req); err != nil {
			resp := NewErrorResponse("", ErrorCodeInvalidRequest, "Invalid request payload")
			return marshalResponse(resp)
		}

		// 提交任务到工作池处理
		// 修正：将变量名从ask改为task，确保后续使用一致
		task := &IPCTask{
			TaskID:   req.RequestID,
			Req:      &req,
			Registry: registry, // 修改为指针类型
		}

		// 提交任务并获取结果通道
		// 修正：使用正确的变量名task
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

// marshalResponse 序列化响应为消息帧
// 辅助函数，将Response转换为MessageFrame并序列化为JSON
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | resp           | 响应对象                          |
// | 返回值         | 序列化后的消息帧数据和可能的错误  |
// +----------------+-----------------------------------+
func marshalResponse(resp *Response) ([]byte, error) {
	frame := MessageFrame{
		Type:    string(MessageTypeResponse),
		Payload: []byte(json.RawMessage(resp.Data)),
	}
	return json.Marshal(frame)
}

// NewRequestID 生成唯一请求ID
// 格式为当前时间（年月日时分秒）+ 8位随机字符串
// +----------------+-----------------------------------+
// | 返回值         | 生成的唯一请求ID字符串            |
// +----------------+-----------------------------------+
func NewRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randString(8)
}

// randString 生成指定长度的随机字符串
// 用于请求ID的随机部分生成
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | n              | 生成的字符串长度                  |
// | 返回值         | 包含字母和数字的随机字符串        |
// +----------------+-----------------------------------+
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
