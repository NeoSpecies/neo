package ipcprotocol

import (
	"context"
	"encoding/json"
	"fmt"
	"neo/internal/discovery"
	"neo/internal/types"
	"time"
)

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
func ProcessMessage(data []byte, registry *types.ServiceRegistry, workerPool types.WorkerPool) ([]byte, error) {
	// 解析消息帧
	var frame types.MessageFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		resp := NewErrorResponse("", types.ErrorCodeInvalidRequest, "Invalid message format")
		return marshalResponse(resp)
	}

	// 处理请求类型消息
	if frame.Type == string(types.MessageTypeRequest) {
		var req types.Request
		if err := json.Unmarshal(frame.Payload, &req); err != nil {
			resp := NewErrorResponse("", types.ErrorCodeInvalidRequest, "Invalid request payload")
			return marshalResponse(resp)
		}

		// 新增：处理注册请求
		if req.Service == "discovery" && req.Method == "register" {
			// 解析注册请求参数
			var registerParams struct {
				ServiceID string `json:"service_id"`
				Name      string `json:"name"`
				Address   string `json:"address"`
				Port      int    `json:"port"`
			}
			if err := json.Unmarshal(req.Params, &registerParams); err != nil {
				resp := NewErrorResponse(req.RequestID, types.ErrorCodeInvalidRequest, "Invalid register parameters")
				return marshalResponse(resp)
			}

			// 创建服务实例
			service := &types.Service{
				ID:        registerParams.ServiceID,
				Name:      registerParams.Name,
				Address:   registerParams.Address,
				Port:      registerParams.Port,
				Status:    "active",
				UpdatedAt: time.Now(),
				ExpireAt:  time.Now().Add(30 * time.Minute), // 设置30分钟租约
			}

			// 注册服务
			// 修复：使用正确的服务发现注册方式，与main.go保持一致
			storage := discovery.NewInMemoryStorage()
			discoveryInstance := types.NewDiscovery(storage)
			discoveryService := &discovery.DiscoveryService{Discovery: discoveryInstance}
			if err := discoveryService.Register(context.Background(), service); err != nil {
				resp := NewErrorResponse(req.RequestID, types.ErrorCodeInternalError, fmt.Sprintf("Failed to register service: %v", err))
				return marshalResponse(resp)
			}

			// 返回包含result字段的响应
			result := map[string]interface{}{
				"result": service,
			}
			resp, err := NewResponse(req.RequestID, result)
			if err != nil {
				return marshalResponse(NewErrorResponse(req.RequestID, types.ErrorCodeInternalError, "Failed to create response"))
			}
			return marshalResponse(resp)
		}

		// 原有代码：提交任务到工作池处理
		task := &types.IPCTask{
			TaskID:   req.RequestID,
			Req:      &req,
			Registry: registry,
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
    // 移除MessageFrame包装，直接返回响应数据
    return resp.Data, nil
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
