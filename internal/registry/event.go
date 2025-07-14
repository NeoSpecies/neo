package registry

import "time"

// EventType 事件类型
type EventType int

const (
	// EventRegister 注册事件
	EventRegister EventType = iota
	// EventDeregister 注销事件
	EventDeregister
	// EventUpdate 更新事件
	EventUpdate
	// EventHealthChange 健康状态变化事件
	EventHealthChange
)

// ServiceEvent 服务事件
type ServiceEvent struct {
	// Type 事件类型
	Type EventType
	// Service 服务名称
	Service string
	// Instance 服务实例
	Instance *ServiceInstance
	// Timestamp 事件时间
	Timestamp time.Time
	// OldInstance 旧实例（用于更新事件）
	OldInstance *ServiceInstance
}

// String 返回事件类型的字符串表示
func (e EventType) String() string {
	switch e {
	case EventRegister:
		return "REGISTER"
	case EventDeregister:
		return "DEREGISTER"
	case EventUpdate:
		return "UPDATE"
	case EventHealthChange:
		return "HEALTH_CHANGE"
	default:
		return "UNKNOWN"
	}
}