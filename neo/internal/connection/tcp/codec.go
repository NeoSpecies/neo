package tcp

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"neo/internal/ipcprotocol"
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
	frameData, err := c.ReadFrame()
	if err != nil {
		return nil, err
	}

	// 解析消息帧
	var messageFrame ipcprotocol.MessageFrame
	if err := json.Unmarshal(frameData, &messageFrame); err != nil {
		return nil, &ConnectionError{
			Type:    ErrorTypeInvalidData,
			Message: "解析IPC消息失败",
			Err:     err,
		}
	}

	return &messageFrame, nil
}

// 写入IPC消息
func (c *Codec) WriteIPCMessage(messageFrame *ipcprotocol.MessageFrame) error {
	// 序列化消息
	frameData, err := json.Marshal(messageFrame)
	if err != nil {
		return &ConnectionError{
			Type:    ErrorTypeInvalidData,
			Message: "序列化IPC消息失败",
			Err:     err,
		}
	}

	// 写入帧数据
	return c.WriteFrame(frameData)
}
