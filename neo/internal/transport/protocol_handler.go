// 新建协议处理文件，分离解析逻辑
package transport

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"neo/internal/ipcprotocol"
)

// 协议处理器接口
type ProtocolHandler interface {
	ParseRequest(reader *bufio.Reader) (*ipcprotocol.Message, error)
	BuildResponse(req *ipcprotocol.Message, result interface{}, err error) []byte
}

// 具体实现
type DefaultProtocolHandler struct {
	magicNumber uint16
	version     byte
}

// 解析请求
func (h *DefaultProtocolHandler) ParseRequest(reader *bufio.Reader) (*ipcprotocol.Message, error) {
    // 读取完整消息数据（头部+负载）
    headerBuf := make([]byte, ipcprotocol.HeaderSize)
    if _, err := io.ReadFull(reader, headerBuf); err != nil {
        return nil, fmt.Errorf("读取协议头失败: %w", err)
    }

    // 解析头部获取总长度
    // 修复：移除手动构造MessageHeader的冗余代码
    msgIDLen := binary.BigEndian.Uint16(headerBuf[3:5])
    methodLen := binary.BigEndian.Uint16(headerBuf[5:7])
    paramLen := binary.BigEndian.Uint32(headerBuf[7:11])
    totalLen := ipcprotocol.HeaderSize + int(msgIDLen) + int(methodLen) + int(paramLen)

    // 读取完整消息
    data := make([]byte, totalLen)
    copy(data[:ipcprotocol.HeaderSize], headerBuf)
    if _, err := io.ReadFull(reader, data[ipcprotocol.HeaderSize:]); err != nil {
        return nil, fmt.Errorf("读取消息体失败: %w", err)
    }

    // 使用ipcprotocol包解析完整消息
    msg, err := ipcprotocol.UnmarshalMessage(data)
    if err != nil {
        return nil, fmt.Errorf("协议解析失败: %v", err)
    }

    // 验证魔数和版本（保留业务逻辑验证）
    if msg.Header.Magic != h.magicNumber {
        return nil, fmt.Errorf("无效魔数: 0x%X, 预期: 0x%X", msg.Header.Magic, h.magicNumber)
    }
    if msg.Header.Version > h.version {
        return nil, fmt.Errorf("不支持的协议版本: %d, 当前支持: %d", msg.Header.Version, h.version)
    }

    return msg, nil
}

// 构建响应
func (h *DefaultProtocolHandler) BuildResponse(req *ipcprotocol.Message, result interface{}, err error) []byte {
	// 解析消息ID
	msgIDLen := int(req.Header.MsgIDLen)
	if len(req.Payload) < msgIDLen {
		// 返回错误响应
		errResp := ipcprotocol.NewMessage(ipcprotocol.TypeError, "", []byte(`{"error":"无效的消息负载，消息ID不完整"}`))
		return errResp.Bytes()
	}
	requestID := string(req.Payload[:msgIDLen])

	// 构建响应体
	responseBody := struct {
		RequestID string      `json:"request_id"`
		Result    interface{} `json:"result,omitempty"`
		Error     string      `json:"error,omitempty"`
		Code      int         `json:"code,omitempty"`
	}{RequestID: requestID}

	if err != nil {
		responseBody.Error = err.Error()
		responseBody.Code = 500
	} else {
		responseBody.Result = result
		responseBody.Code = 0
	}

	// 序列化为JSON
	jsonBody, _ := json.Marshal(responseBody)

	// 创建响应消息
	return ipcprotocol.NewMessage(ipcprotocol.TypeResponse, "", jsonBody).Bytes()
}

// 构造函数
func NewDefaultProtocolHandler(magicNumber uint16, version byte) *DefaultProtocolHandler {
	return &DefaultProtocolHandler{
		magicNumber: magicNumber,
		version:     version,
	}
}
