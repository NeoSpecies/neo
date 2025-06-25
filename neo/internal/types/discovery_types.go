package types

import (
	"context"
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
