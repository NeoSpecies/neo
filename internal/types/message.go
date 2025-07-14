package types

import "time"

// MessageType 消息类型
type MessageType uint8

const (
	// REQUEST 请求消息
	REQUEST MessageType = iota + 1
	// RESPONSE 响应消息
	RESPONSE
	// REGISTER 服务注册
	REGISTER
	// HEARTBEAT 心跳检测
	HEARTBEAT
)

// Message 表示框架中传递的通用消息结构
// 支持 JSON 序列化，作为所有消息的基础结构
type Message struct {
	ID        string            `json:"id"`                   // 消息唯一标识
	Type      MessageType       `json:"type"`                 // 消息类型
	Service   string            `json:"service"`              // 服务名称
	Method    string            `json:"method"`               // 方法名称
	Metadata  map[string]string `json:"metadata,omitempty"`   // 元数据
	Body      []byte            `json:"body"`                 // 消息正文
	Timestamp time.Time         `json:"timestamp"`            // 时间戳
}

// NewMessage 创建新消息
func NewMessage(msgType MessageType, service, method string) *Message {
	return &Message{
		ID:        GenerateID(),
		Type:      msgType,
		Service:   service,
		Method:    method,
		Metadata:  make(map[string]string),
		Timestamp: time.Now(),
	}
}

// Validate 验证消息结构
func (m *Message) Validate() error {
	if m.ID == "" {
		return ErrInvalidMessageID
	}
	if m.Type < REQUEST || m.Type > HEARTBEAT {
		return ErrInvalidMessageType
	}
	if m.Service == "" {
		return ErrInvalidService
	}
	return nil
}