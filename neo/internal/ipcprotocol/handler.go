package ipcprotocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"neo/internal/types"
)

// 协议帧分隔符
const FrameDelimiter = '\x00'

// 消息处理器接口
type MessageHandler interface {
	HandleRequest(request *types.Request) (*types.Response, error)
	HandleEvent(event *types.MessageFrame) error
}

// 协议编解码器
type Codec struct {
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.Mutex
}

// 创建新的编解码器
func NewCodec(reader io.Reader, writer io.Writer) *Codec {
	return &Codec{
		reader: bufio.NewReader(reader),
		writer: bufio.NewWriter(writer),
	}
}

// 读取消息帧
func (c *Codec) ReadFrame() (*types.MessageFrame, error) {
	// 读取直到分隔符
	data, err := c.reader.ReadBytes(FrameDelimiter)
	if err != nil {
		return nil, err
	}

	// 移除分隔符
	if len(data) > 0 && data[len(data)-1] == FrameDelimiter {
		data = data[:len(data)-1]
	}

	// 空数据处理
	if len(data) == 0 {
		return nil, errors.New("空消息帧")
	}

	// 解析消息帧
	var frame types.MessageFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		return nil, errors.New("解析消息帧失败: " + err.Error())
	}

	return &frame, nil
}

// 写入消息帧
func (c *Codec) WriteFrame(frame *types.MessageFrame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 序列化消息帧
	data, err := json.Marshal(frame)
	if err != nil {
		return err
	}

	// 写入数据和分隔符
	if _, err := c.writer.Write(data); err != nil {
		return err
	}
	if err := c.writer.WriteByte(FrameDelimiter); err != nil {
		return err
	}

	// 刷新缓冲区
	return c.writer.Flush()
}

// 解析请求
func ParseRequest(payload []byte) (*types.Request, error) {
	var request types.Request
	if err := json.Unmarshal(payload, &request); err != nil {
		return nil, err
	}

	// 验证请求字段
	if request.RequestID == "" {
		return nil, errors.New("缺少request_id")
	}
	if request.Service == "" {
		return nil, errors.New("缺少service")
	}
	if request.Method == "" {
		return nil, errors.New("缺少method")
	}

	return &request, nil
}

// 解析响应
func ParseResponse(payload []byte) (*types.Response, error) {
	var response types.Response
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, err
	}

	if response.RequestID == "" {
		return nil, errors.New("缺少request_id")
	}

	return &response, nil
}
