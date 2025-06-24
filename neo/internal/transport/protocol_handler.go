// 新建协议处理文件，分离解析逻辑
package transport

import (
	"bufio"
	"encoding/json"
	"fmt"

	"neo/internal/ipcprotocol"
)

// 协议处理器接口
type ProtocolHandler interface {
	ParseRequest(reader *bufio.Reader) (*ipcprotocol.MessageFrame, error)
	BuildResponse(req *ipcprotocol.MessageFrame, result interface{}, err error) ([]byte, error)
}

// 具体实现
type DefaultProtocolHandler struct {
	magicNumber uint16
	version     byte
}

// 解析请求
func (h *DefaultProtocolHandler) ParseRequest(reader *bufio.Reader) (*ipcprotocol.MessageFrame, error) {
	// 读取消息帧数据
	// 注意：ipcprotocol包中没有定义HeaderSize，使用合理的缓冲区大小
	buf := make([]byte, 4096)
	n, err := reader.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("读取消息失败: %w", err)
	}

	// 解析JSON消息帧
	var frame ipcprotocol.MessageFrame
	if err := json.Unmarshal(buf[:n], &frame); err != nil {
		return nil, fmt.Errorf("解析消息帧失败: %w", err)
	}

	return &frame, nil
}

// 构建响应
func (h *DefaultProtocolHandler) BuildResponse(req *ipcprotocol.MessageFrame, result interface{}, err error) ([]byte, error) {
	// 构建响应负载
	responsePayload := struct {
		RequestID string      `json:"request_id"`
		Result    interface{} `json:"result,omitempty"`
		Error     string      `json:"error,omitempty"`
		Code      int         `json:"code,omitempty"`
	}{}

	// 尝试从请求负载中解析RequestID
	var request struct {
		RequestID string `json:"request_id"`
	}
	if err := json.Unmarshal(req.Payload, &request); err == nil {
		responsePayload.RequestID = request.RequestID
	}

	// 设置响应状态
	if err != nil {
		responsePayload.Error = err.Error()
		responsePayload.Code = 500
	} else {
		responsePayload.Result = result
		responsePayload.Code = 0
	}

	// 序列化为JSON
	payload, _ := json.Marshal(responsePayload)

	// 创建响应消息帧并序列化为字节
	frame := &ipcprotocol.MessageFrame{
		Type:    ipcprotocol.MessageTypeResponse,
		Payload: payload,
	}
	frameBytes, err := json.Marshal(frame)
	if err != nil {
		return nil, err
	}
	return frameBytes, nil
}

// 构造函数
func NewDefaultProtocolHandler(magicNumber uint16, version byte) *DefaultProtocolHandler {
	return &DefaultProtocolHandler{
		magicNumber: magicNumber,
		version:     version,
	}
}
