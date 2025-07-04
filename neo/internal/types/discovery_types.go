package types

import (
	"context"
	"sync"
	"time"
)

// Service 服务元数据
type Service struct {
	ID        string            `json:"id"`         // 全局唯一ID
	Name      string            `json:"name"`       // 服务名称
	Address   string            `json:"address"`    // IPC地址
	Port      int               `json:"port"`       // 服务端口号
	Metadata  map[string]string `json:"metadata"`   // 扩展元数据
	Status    string            `json:"status"`     // 状态
	ExpireAt  time.Time         `json:"expire_at"`  // 租约过期时间
	UpdatedAt time.Time         `json:"updated_at"` // 最后更新时间
}

// EventType 服务事件类型枚举
type EventType int

const (
	EventRegistered   EventType = iota // 服务注册事件
	EventDeregistered                  // 服务注销事件
	EventExpired                       // 服务过期事件
)

// Event 服务变更事件结构体
type Event struct {
	Type    EventType // 事件类型
	Service *Service  // 关联的服务实例
}

// Storage 服务存储接口
type Storage interface {
	Register(ctx context.Context, s *Service) error
	Deregister(ctx context.Context, serviceID string) error
	Get(ctx context.Context, serviceID string) (*Service, error)
	List(ctx context.Context, serviceName string) ([]*Service, error)
	Renew(ctx context.Context, serviceID string) error
}

// Discovery 服务发现核心组件
type Discovery struct {
	Storage  Storage
	Events   chan Event
	Watchers map[string][]chan Event
	Mu       sync.RWMutex
	Ctx      context.Context
	Cancel   context.CancelFunc
}

// NewDiscovery 创建服务发现实例
func NewDiscovery(storage Storage) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &Discovery{
		Storage:  storage,
		Events:   make(chan Event, 100),
		Watchers: make(map[string][]chan Event),
		Ctx:      ctx,
		Cancel:   cancel,
	}
}

// Message 服务发现协议消息
type Message struct {
	Type     MessageType
	Service  *Service
	Response []*Service
	Error    string
}

// ServiceEvent 服务事件
type ServiceEvent struct {
	Type    EventType // 事件类型（注册/注销）
	Service *Service  // 涉及的服务实例
}
