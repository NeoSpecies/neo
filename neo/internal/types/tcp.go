package types

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"sync"
	"time"
)

func NewCodec(reader io.Reader, writer io.Writer) *Codec {
	return &Codec{
		reader: bufio.NewReader(reader),
		writer: bufio.NewWriter(writer),
	}
}

// Codec TCP编解码器
type Codec struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// 协议常量定义
const (
	MAGIC_NUMBER uint16 = 0xAEBD  // 2字节大端魔数
	VERSION      uint8  = 0x01    // 1字节协议版本
	MaxFrameSize uint32 = 1 << 20 // 1MB 最大帧大小限制
)

// 删除此处的ConnectionHandler定义，保留connection_types.go中的唯一实现

// TCP服务器配置
type TCPConfig struct {
	MaxConnections    int           `yaml:"max_connections"`
	MaxMsgSize        int           `yaml:"max_message_size"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	WorkerCount       int           `yaml:"worker_count"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	// 新增字段：直接存储地址信息，避免依赖config包
	Address string `yaml:"address"`
}

// GetAddress 实现ServerConfig接口方法
// 不再依赖config包，直接返回结构体中的Address字段
func (c *TCPConfig) GetAddress() string {
	return c.Address
}

// ReadIPCMessage 从连接读取IPC消息
func (c *Codec) ReadIPCMessage() (*MessageFrame, error) {
	reader := bufio.NewReader(c.reader)
	var magic uint16
	var version uint8
	var msgIDLen, methodLen uint16
	var paramLen uint32
	var checksum uint32

	// 1. 读取并验证魔数(2字节)
	if err := binary.Read(reader, binary.BigEndian, &magic); err != nil {
		if err == io.EOF {
			return nil, &ConnectionError{Type: ErrorTypeConnectionClosed, Message: "客户端断开连接", Err: err}
		}
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取魔数失败", Err: err}
	}

	// 验证魔数
	if magic != MAGIC_NUMBER {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("魔数校验失败，期望0x%04X，实际0x%04X", MAGIC_NUMBER, magic)}
	}

	// 2. 读取并验证版本(1字节)
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取版本失败", Err: err}
	}

	// 验证版本
	if version != VERSION {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("版本不匹配，期望%d，实际%d", VERSION, version)}
	}

	// 3. 读取消息ID长度(2字节)和内容
	if err := binary.Read(reader, binary.BigEndian, &msgIDLen); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取消息ID长度失败", Err: err}
	}
	msgID := make([]byte, msgIDLen)
	if _, err := io.ReadFull(reader, msgID); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取消息ID内容失败", Err: err}
	}

	// 4. 读取方法名长度(2字节)和内容
	if err := binary.Read(reader, binary.BigEndian, &methodLen); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取方法名长度失败", Err: err}
	}
	method := make([]byte, methodLen)
	if _, err := io.ReadFull(reader, method); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取方法名内容失败", Err: err}
	}

	// 5. 读取参数长度(4字节)和内容
	if err := binary.Read(reader, binary.BigEndian, &paramLen); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取参数长度失败", Err: err}
	}
	// 检查参数大小限制
	if paramLen > MaxFrameSize {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("参数大小超出限制: %d > %d", paramLen, MaxFrameSize)}
	}
	params := make([]byte, paramLen)
	if _, err := io.ReadFull(reader, params); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取参数内容失败", Err: err}
	}

	// 6. 读取并验证CRC32校验和(4字节)
	if err := binary.Read(reader, binary.BigEndian, &checksum); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取校验和失败", Err: err}
	}

	// 验证校验和
	headerBuffer := new(bytes.Buffer)
	binary.Write(headerBuffer, binary.BigEndian, magic)
	binary.Write(headerBuffer, binary.BigEndian, version)
	binary.Write(headerBuffer, binary.BigEndian, msgIDLen)
	headerBuffer.Write(msgID)
	binary.Write(headerBuffer, binary.BigEndian, methodLen)
	headerBuffer.Write(method)
	binary.Write(headerBuffer, binary.BigEndian, paramLen)
	headerBuffer.Write(params)

	computedChecksum := crc32.ChecksumIEEE(headerBuffer.Bytes())
	if computedChecksum != checksum {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("校验和不匹配，期望0x%08X，实际0x%08X", checksum, computedChecksum)}
	}

	// 解析参数为消息帧
	var messageData map[string]interface{}
	if err := json.Unmarshal(params, &messageData); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: fmt.Sprintf("JSON解析失败: %v, 原始数据: %s", err, string(params))}
	}

	// 提取服务注册核心信息
	action, actionOk := messageData["action"].(string)
	serviceData, serviceOk := messageData["service"].(map[string]interface{})
	if !actionOk || !serviceOk {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "JSON缺少必要字段: action或service"}
	}

	serviceID := serviceData["id"].(string)
	serviceName := serviceData["name"].(string)
	address := serviceData["address"].(string)
	port := int(serviceData["port"].(float64))

	// 创建消息帧
	return &MessageFrame{
		Type:    action,
		Payload: []byte(fmt.Sprintf(`{"service_id":"%s","name":"%s","address":"%s","port":%d}`, serviceID, serviceName, address, port)),
	}, nil
}

// WriteIPCMessage 将消息帧写入连接
func (c *Codec) WriteIPCMessage(frame *MessageFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	buffer := new(bytes.Buffer)

	// 1. 写入魔数(2字节)
	if err := binary.Write(buffer, binary.BigEndian, MAGIC_NUMBER); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入魔数失败", Err: err}
	}

	// 2. 写入版本(1字节)
	if err := binary.Write(buffer, binary.BigEndian, VERSION); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入版本失败", Err: err}
	}

	// 删除3. 消息ID和4. 方法名相关代码

	// 5. 写入参数长度(4字节)和内容 → 现在紧跟版本字段，与客户端匹配
	paramLen := uint32(len(frame.Payload))
	if err := binary.Write(buffer, binary.BigEndian, paramLen); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入参数长度失败", Err: err}
	}
	if _, err := buffer.Write(frame.Payload); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入参数内容失败", Err: err}
	}

	// 6. 重新计算校验和(仅包含魔数+版本+参数长度+参数内容)
	checksum := crc32.ChecksumIEEE(buffer.Bytes())
	if err := binary.Write(buffer, binary.BigEndian, checksum); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入校验和失败", Err: err}
	}

	// 写入数据并刷新缓冲区
	if _, err := c.writer.Write(buffer.Bytes()); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "写入帧数据失败", Err: err}
	}
	if err := c.writer.Flush(); err != nil {
		return &ConnectionError{Type: ErrorTypeWriteFailed, Message: "刷新缓冲区失败", Err: err}
	}

	return nil
}

// Reader 返回读取器
func (c *Codec) Reader() *bufio.Reader {
	return c.reader
}

// Writer 返回写入器
func (c *Codec) Writer() *bufio.Writer {
	return c.writer
}
func (c *TCPConfig) GetMaxConnections() int {
	return c.MaxConnections
}

func (c *TCPConfig) GetConnectionTimeout() time.Duration {
	return c.ConnectionTimeout
}

func (c *TCPConfig) GetHandlerConfig() interface{} {
	return nil
}
