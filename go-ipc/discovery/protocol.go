package discovery

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
	Service  *ServiceInfo
	Response []*ServiceInfo
	Error    string
}
