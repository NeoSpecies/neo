package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"neo/internal/types"
	"time"
)

const (
	// IPC 协议版本
	IPCVersion = "1.0"
	
	// 最大消息长度 (10MB)
	MaxMessageSize = 10 * 1024 * 1024
)

// IPCCodec IPC二进制协议编解码器
type IPCCodec struct{}

// NewIPCCodec 创建IPC编解码器
func NewIPCCodec() *IPCCodec {
	return &IPCCodec{}
}

// Version 返回协议版本
func (c *IPCCodec) Version() string {
	return IPCVersion
}

// Encode 编码消息为二进制格式
// 格式: [Length:4][Type:1][ID长度:2][ID][Service长度:2][Service][Method长度:2][Method][Metadata长度:2][Metadata][Data]
func (c *IPCCodec) Encode(msg types.Message) ([]byte, error) {
	buf := new(bytes.Buffer)
	
	// 预留长度字段位置（4字节）
	buf.Write(make([]byte, 4))
	
	// 写入消息类型（1字节）
	if err := buf.WriteByte(byte(msg.Type)); err != nil {
		return nil, fmt.Errorf("failed to write message type: %w", err)
	}
	
	// 写入ID
	if err := writeString(buf, msg.ID); err != nil {
		return nil, fmt.Errorf("failed to write ID: %w", err)
	}
	
	// 写入Service
	if err := writeString(buf, msg.Service); err != nil {
		return nil, fmt.Errorf("failed to write service: %w", err)
	}
	
	// 写入Method
	if err := writeString(buf, msg.Method); err != nil {
		return nil, fmt.Errorf("failed to write method: %w", err)
	}
	
	// 写入Metadata
	if err := writeMetadata(buf, msg.Metadata); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}
	
	// 写入Body
	if _, err := buf.Write(msg.Body); err != nil {
		return nil, fmt.Errorf("failed to write body: %w", err)
	}
	
	// 获取完整数据
	data := buf.Bytes()
	
	// 写入实际长度（不包括长度字段本身）
	length := uint32(len(data) - 4)
	binary.BigEndian.PutUint32(data[:4], length)
	
	return data, nil
}

// Decode 解码二进制数据为消息
func (c *IPCCodec) Decode(data []byte) (types.Message, error) {
	if len(data) < 4 {
		return types.Message{}, fmt.Errorf("data too short: need at least 4 bytes for length")
	}
	
	// 读取消息长度
	length := binary.BigEndian.Uint32(data[:4])
	if length > MaxMessageSize {
		return types.Message{}, fmt.Errorf("message too large: %d bytes (max %d)", length, MaxMessageSize)
	}
	
	if uint32(len(data)-4) < length {
		return types.Message{}, fmt.Errorf("incomplete message: expected %d bytes, got %d", length, len(data)-4)
	}
	
	reader := bytes.NewReader(data[4:])
	
	// 读取消息类型
	typeByte, err := reader.ReadByte()
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to read message type: %w", err)
	}
	
	// 读取ID
	id, err := readString(reader)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to read ID: %w", err)
	}
	
	// 读取Service
	service, err := readString(reader)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to read service: %w", err)
	}
	
	// 读取Method
	method, err := readString(reader)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to read method: %w", err)
	}
	
	// 读取Metadata
	metadata, err := readMetadata(reader)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to read metadata: %w", err)
	}
	
	// 读取剩余的Body
	body, err := io.ReadAll(reader)
	if err != nil {
		return types.Message{}, fmt.Errorf("failed to read body: %w", err)
	}
	
	return types.Message{
		ID:        id,
		Type:      types.MessageType(typeByte),
		Service:   service,
		Method:    method,
		Metadata:  metadata,
		Body:      body,
		Timestamp: time.Now(),
	}, nil
}

// writeString 写入字符串（格式：2字节长度 + 字符串内容）
func writeString(buf *bytes.Buffer, s string) error {
	if len(s) > 65535 {
		return fmt.Errorf("string too long: %d bytes (max 65535)", len(s))
	}
	
	// 写入长度（2字节）
	if err := binary.Write(buf, binary.BigEndian, uint16(len(s))); err != nil {
		return err
	}
	
	// 写入内容
	_, err := buf.WriteString(s)
	return err
}

// readString 读取字符串
func readString(reader *bytes.Reader) (string, error) {
	// 读取长度
	var length uint16
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return "", err
	}
	
	// 读取内容
	data := make([]byte, length)
	if _, err := io.ReadFull(reader, data); err != nil {
		return "", err
	}
	
	return string(data), nil
}

// writeMetadata 写入元数据
func writeMetadata(buf *bytes.Buffer, metadata map[string]string) error {
	// 写入元数据数量（2字节）
	count := uint16(len(metadata))
	if err := binary.Write(buf, binary.BigEndian, count); err != nil {
		return err
	}
	
	// 写入每个键值对
	for k, v := range metadata {
		if err := writeString(buf, k); err != nil {
			return err
		}
		if err := writeString(buf, v); err != nil {
			return err
		}
	}
	
	return nil
}

// readMetadata 读取元数据
func readMetadata(reader *bytes.Reader) (map[string]string, error) {
	// 读取元数据数量
	var count uint16
	if err := binary.Read(reader, binary.BigEndian, &count); err != nil {
		return nil, err
	}
	
	metadata := make(map[string]string, count)
	
	// 读取每个键值对
	for i := uint16(0); i < count; i++ {
		key, err := readString(reader)
		if err != nil {
			return nil, err
		}
		
		value, err := readString(reader)
		if err != nil {
			return nil, err
		}
		
		metadata[key] = value
	}
	
	return metadata, nil
}