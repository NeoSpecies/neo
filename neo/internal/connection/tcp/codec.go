package tcp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"neo/internal/ipcprotocol"
	"sync"
	// 删除此行: "github.com/google/uuid"
)

// 错误类型定义
const (
	ErrorTypeReadFailed       = "read_failed"
	ErrorTypeWriteFailed      = "write_failed"
	ErrorTypeInvalidData      = "invalid_data"
	ErrorTypeConnection       = "connection_error"
	ErrorTypeConnectionClosed = "connection_closed" // 新增连接关闭错误类型
)

// ConnectionError 连接错误结构体
type ConnectionError struct {
	Type    string
	Message string
	Err     error
}

// 实现error接口
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// TCP编解码器
type Codec struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// 创建新的TCP编解码器
func NewCodec(reader io.Reader, writer io.Writer) *Codec {
	return &Codec{
		reader: bufio.NewReader(reader),
		writer: bufio.NewWriter(writer),
	}
}

// 读取IPC消息
func (c *Codec) ReadIPCMessage() (*ipcprotocol.MessageFrame, error) {
	// 移除帧长度前缀依赖，直接读取协议内容
	reader := bufio.NewReader(c.reader)
	var magic uint16
	var version uint8
	var msgIDLen, methodLen uint16
	var paramLen uint32
	var checksum uint32

	// 1. 读取魔数(2字节)
	if err := binary.Read(reader, binary.BigEndian, &magic); err != nil {
		if err == io.EOF {
			return nil, &ConnectionError{Type: ErrorTypeConnectionClosed, Message: "客户端断开连接", Err: err}
		}
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取魔数失败", Err: err}
	}
	fmt.Printf("[协议头] 魔数: 0x%04X\n", magic)

	// 2. 读取版本(1字节)
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取版本失败", Err: err}
	}
	fmt.Printf("[协议头] 版本: %d\n", version)

	// 3. 读取消息ID长度(2字节)和内容
	if err := binary.Read(reader, binary.BigEndian, &msgIDLen); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取消息ID长度失败", Err: err}
	}
	msgID := make([]byte, msgIDLen)
	if _, err := io.ReadFull(reader, msgID); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取消息ID内容失败", Err: err}
	}
	fmt.Printf("[协议头] 消息ID: %s\n", string(msgID))

	// 4. 读取方法名长度(2字节)和内容
	if err := binary.Read(reader, binary.BigEndian, &methodLen); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取方法名长度失败", Err: err}
	}
	method := make([]byte, methodLen)
	if _, err := io.ReadFull(reader, method); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取方法名内容失败", Err: err}
	}
	fmt.Printf("[协议头] 方法名: %s\n", string(method))

	// 5. 读取参数长度(4字节)和内容
	if err := binary.Read(reader, binary.BigEndian, &paramLen); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取参数长度失败", Err: err}
	}
	// 添加参数长度限制，防止内存溢出
	if paramLen > MaxFrameSize {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("参数大小超出限制: %d > %d", paramLen, MaxFrameSize)}
	}
	params := make([]byte, paramLen)
	if _, err := io.ReadFull(reader, params); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取参数内容失败", Err: err}
	}
	fmt.Printf("[DEBUG] 接收到的参数原始字节: % x\n", params)
	fmt.Printf("[协议头] 参数长度: %d字节\n", paramLen)
	fmt.Printf("[参数内容] %s\n", string(params))

	// 6. 读取校验和(4字节)
	if err := binary.Read(reader, binary.BigEndian, &checksum); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取校验和失败", Err: err}
	}
	fmt.Printf("[协议头] 校验和: 0x%08X\n", checksum)

	// 重新构建完整协议头数据用于校验和计算
	// 1. 创建缓冲区并写回已读取的协议头字段
	headerBuffer := new(bytes.Buffer)
	binary.Write(headerBuffer, binary.BigEndian, magic)
	binary.Write(headerBuffer, binary.BigEndian, version)
	binary.Write(headerBuffer, binary.BigEndian, msgIDLen)
	headerBuffer.Write(msgID)
	binary.Write(headerBuffer, binary.BigEndian, methodLen)
	headerBuffer.Write(method)
	binary.Write(headerBuffer, binary.BigEndian, paramLen)
	headerBuffer.Write(params)

	// 2. 对完整协议头计算校验和
	computedChecksum := crc32.ChecksumIEEE(headerBuffer.Bytes())
	fmt.Printf("[DEBUG] 计算校验和: 0x%08X, 接收校验和: 0x%08X\n", computedChecksum, checksum)

	if computedChecksum != checksum {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("校验和不匹配，期望0x%08X，实际0x%08X", checksum, computedChecksum)}
	}

	// 验证魔数
	if magic != 0xAEBD {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("魔数校验失败，期望0xAEBD，实际0x%04X", magic)}
	}

	// 验证版本
	if version != 0x01 {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("版本不匹配，期望1，实际%d", version)}
	}

	// 解析参数为JSON对象而非MessageFrame结构体
	var messageData map[string]interface{}
	if err := json.Unmarshal(params, &messageData); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("JSON解析失败: %v, 原始数据: %s", err, string(params))}
	}

	// 添加详细调试信息
	fmt.Printf("[DEBUG] 解析后的JSON参数: %+v\n", messageData)

	// 验证关键字段是否存在
	action, actionOk := messageData["action"].(string)
	serviceData, serviceOk := messageData["service"].(map[string]interface{})
	if !actionOk || !serviceOk {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "JSON缺少必要字段: action或service"}
	}

	// 提取服务注册核心信息
	serviceID := serviceData["id"].(string)
	serviceName := serviceData["name"].(string)
	address := serviceData["address"].(string)
	port := int(serviceData["port"].(float64))

	// 创建正确的MessageFrame结构
	messageFrame := &ipcprotocol.MessageFrame{
		Type:    action,
		Payload: []byte(fmt.Sprintf(`{"service_id":"%s","name":"%s","address":"%s","port":%d}`, serviceID, serviceName, address, port)),
	}

	fmt.Printf("[DEBUG] 构造的消息帧: Type=%s, Payload=%s\n", messageFrame.Type, string(messageFrame.Payload))
	return messageFrame, nil
}

// 写入IPC消息
// 添加协议常量定义（确保使用固定大小类型）
const (
	MAGIC_NUMBER uint16 = 0xAEBD  // 2字节大端魔数
	VERSION      uint8  = 0x01    // 1字节协议版本
	MaxFrameSize uint32 = 1 << 20 // 1MB 最大帧大小限制
)

// WriteIPCMessage 按协议格式写入消息
// 确保响应消息完整封装
func (c *Codec) WriteIPCMessage(frame *ipcprotocol.MessageFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 创建缓冲区
	buffer := new(bytes.Buffer)

	// 1. 写入魔数(2字节，大端序)
	if err := binary.Write(buffer, binary.BigEndian, MAGIC_NUMBER); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入魔数失败", Err: err}
	}

	// 2. 写入版本(1字节)
	if err := binary.Write(buffer, binary.BigEndian, VERSION); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入版本失败", Err: err}
	}

	// 3. 写入响应体长度(4字节)
	if err := binary.Write(buffer, binary.BigEndian, uint32(len(frame.Payload))); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入长度失败", Err: err}
	}

	// 4. 写入响应体
	if _, err := buffer.Write(frame.Payload); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入响应体失败", Err: err}
	}

	// 写入完整帧
	// return c.WriteFrame(buffer.Bytes())

	// 直接写入缓冲区内容并刷新
	if _, err := c.writer.Write(buffer.Bytes()); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入帧数据失败", Err: err}
	}
	if err := c.writer.Flush(); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "刷新缓冲区失败", Err: err}
	}
	return nil
}
