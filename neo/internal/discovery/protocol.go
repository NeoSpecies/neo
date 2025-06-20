package discovery

import (
	"log"
)

// MessageType 消息类型
type MessageType int

const (
	Register MessageType = iota
	Deregister
	Discover
	Heartbeat
)

// Message 通用消息结构
type Message struct {
	Type     MessageType
	Service  *Service   // 原为 *ServiceInfo
	Response []*Service // 原为 []*ServiceInfo
	Error    string
}

// 假设存在消息处理函数
func HandleMessage(msg *Message) {
	log.Printf("收到消息类型: %v, 服务信息: %+v", msg.Type, msg.Service)
	// 现有处理逻辑...
}
