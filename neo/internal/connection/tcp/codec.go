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
	ErrorTypeReadFailed  = "read_failed"
	ErrorTypeWriteFailed = "write_failed"
	ErrorTypeInvalidData = "invalid_data"
	ErrorTypeConnection  = "connection_error"
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

// 帧格式常量
const (
	FrameHeaderSize = 4       // 帧头部大小（字节）
	MaxFrameSize    = 1 << 20 // 最大帧大小（1MB）
)

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

// 读取消息帧
func (c *Codec) ReadFrame() ([]byte, error) {
	// 读取头部（4字节，大端序表示长度）
	var header [FrameHeaderSize]byte
	if _, err := io.ReadFull(c.reader, header[:]); err != nil {
		return nil, &ConnectionError{
			Type:    ErrorTypeReadFailed,
			Message: "读取帧头部失败",
			Err:     err,
		}
	}

	// 解析长度
	frameSize := binary.BigEndian.Uint32(header[:])

	// 验证长度
	if frameSize == 0 {
		return nil, &ConnectionError{
			Type:    ErrorTypeInvalidData,
			Message: "帧大小不能为0",
		}
	}

	if frameSize > MaxFrameSize {
		return nil, &ConnectionError{
			Type:    ErrorTypeInvalidData,
			Message: fmt.Sprintf("帧大小超出限制: %d > %d", frameSize, MaxFrameSize),
		}
	}

	// 读取帧数据
	frameData := make([]byte, frameSize)
	if _, err := io.ReadFull(c.reader, frameData); err != nil {
		return nil, &ConnectionError{
			Type:    ErrorTypeReadFailed,
			Message: "读取帧数据失败",
			Err:     err,
		}
	}

	return frameData, nil
}

// 写入消息帧
func (c *Codec) WriteFrame(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 检查数据大小
	if len(data) == 0 {
		return &ConnectionError{
			Type:    ErrorTypeInvalidData,
			Message: "数据不能为空",
		}
	}

	if len(data) > MaxFrameSize {
		return &ConnectionError{
			Type:    ErrorTypeInvalidData,
			Message: fmt.Sprintf("数据大小超出限制: %d > %d", len(data), MaxFrameSize),
		}
	}

	// 写入头部（4字节，大端序表示长度）
	header := make([]byte, FrameHeaderSize)
	binary.BigEndian.PutUint32(header, uint32(len(data)))

	if _, err := c.writer.Write(header); err != nil {
		return &ConnectionError{
			Type:    ErrorTypeWriteFailed,
			Message: "写入帧头部失败",
			Err:     err,
		}
	}

	// 写入数据
	if _, err := c.writer.Write(data); err != nil {
		return &ConnectionError{
			Type:    ErrorTypeWriteFailed,
			Message: "写入帧数据失败",
			Err:     err,
		}
	}

	// 刷新缓冲区
	return c.writer.Flush()
}

// 读取IPC消息
func (c *Codec) ReadIPCMessage() (*ipcprotocol.MessageFrame, error) {
	// 移除原有的ReadFrame调用，直接读取数据
	frameData, err := io.ReadAll(c.reader)
	if err != nil {
		return nil, &ConnectionError{
			Type:    ErrorTypeReadFailed,
			Message: "读取数据失败",
			Err:     err,
		}
	}

	// 保留原有的协议头解析逻辑
	reader := bytes.NewReader(frameData)
	var magic uint16
	var version uint8
	var msgIDLen, methodLen uint16
	var paramLen uint32
	var checksum uint32

	// 1. 读取魔数(2字节)
	if err := binary.Read(reader, binary.BigEndian, &magic); err != nil {
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
	params := make([]byte, paramLen)
	if _, err := io.ReadFull(reader, params); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取参数内容失败", Err: err}
	}
	// 添加参数字节调试日志
	fmt.Printf("[DEBUG] 接收到的参数原始字节: % x\n", params)
	fmt.Printf("[协议头] 参数长度: %d字节\n", paramLen)
	fmt.Printf("[参数内容] %s\n", string(params))

	// 6. 读取校验和(4字节)
	if err := binary.Read(reader, binary.BigEndian, &checksum); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取校验和失败", Err: err}
	}
	fmt.Printf("[协议头] 校验和: 0x%08X\n", checksum)

	// 修复：重新构建完整协议头数据用于校验和计算
	// 1. 回到数据起始位置
	reader.Seek(0, io.SeekStart)

	// 2. 读取完整协议头（魔数+版本+消息ID+方法名+参数）
	headerSize := 2 + 1 + 2 + int(msgIDLen) + 2 + int(methodLen) + 4 + int(paramLen)
	fullHeader := make([]byte, headerSize)
	if _, err := io.ReadFull(reader, fullHeader); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "读取完整协议头失败", Err: err}
	}

	// 3. 对完整协议头计算校验和
	computedChecksum := crc32.ChecksumIEEE(fullHeader)
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

	// 解析参数为JSON
	var messageFrame ipcprotocol.MessageFrame
	if err := json.Unmarshal(params, &messageFrame); err != nil {
		return nil, &ConnectionError{Type: ErrorTypeInvalidData, Message: "解析参数JSON失败", Err: err}
	}

	return &messageFrame, nil
}

// 写入IPC消息
// 添加协议常量定义（确保使用固定大小类型）
const (
	MAGIC_NUMBER uint16 = 0xAEBD // 2字节大端魔数
	VERSION      uint8  = 0x01   // 1字节协议版本
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
	return c.WriteFrame(buffer.Bytes())
}
