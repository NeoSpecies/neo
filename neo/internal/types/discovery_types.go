/*
 * 描述: 定义服务发现相关的核心类型，包括服务元数据、事件通知、存储接口和服务发现组件实现
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"context"
	"sync"
	"time"
)

// Service 服务元数据结构体
// 存储服务实例的基本信息和状态
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | ID             | string           | 服务实例全局唯一标识符            |
// | Name           | string           | 服务名称                          |
// | Address        | string           | 服务IPC通信地址                   |
// | Port           | int              | 服务监听端口号                    |
// | Metadata       | map[string]string| 服务扩展元数据（键值对）          |
// | Status         | string           | 服务状态（如：运行中、已停止等）  |
// | ExpireAt       | time.Time        | 服务租约过期时间                  |
// | UpdatedAt      | time.Time        | 服务信息最后更新时间              |
// +----------------+------------------+-----------------------------------+
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
// 定义服务生命周期中的各类事件
// +---------------------+-----------------------------------+
// | 常量名              | 描述                              |
// +---------------------+-----------------------------------+
// | EventRegistered     | 服务注册事件：新服务加入集群       |
// | EventDeregistered   | 服务注销事件：服务主动退出集群     |
// | EventExpired        | 服务过期事件：服务租约到期未续约   |
// +---------------------+-----------------------------------+
type EventType int

const (
	EventRegistered   EventType = iota // 服务注册事件
	EventDeregistered                  // 服务注销事件
	EventExpired                       // 服务过期事件
)

// Event 服务变更事件结构体
// 封装服务状态变更的详细信息
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Type           | EventType        | 事件类型                          |
// | Service        | *Service         | 关联的服务实例信息                |
// +----------------+------------------+-----------------------------------+
type Event struct {
	Type    EventType // 事件类型
	Service *Service  // 关联的服务实例
}

// Storage 服务存储接口
// 定义服务元数据的持久化和查询操作规范
// +--------------------------------+------------------------------------------------+
// | 方法名                         | 描述                                           |
// +--------------------------------+------------------------------------------------+
// | Register                       | 注册服务实例到存储中                           |
// | Deregister                     | 从存储中注销指定服务实例                       |
// | Get                            | 根据服务ID查询服务实例信息                     |
// | List                           | 根据服务名称查询所有相关服务实例               |
// | Renew                          | 续约服务租约，延长服务有效期                   |
// +--------------------------------+------------------------------------------------+
type Storage interface {
	Register(ctx context.Context, s *Service) error
	Deregister(ctx context.Context, serviceID string) error
	Get(ctx context.Context, serviceID string) (*Service, error)
	List(ctx context.Context, serviceName string) ([]*Service, error)
	Renew(ctx context.Context, serviceID string) error
}

// Discovery 服务发现核心组件
// 管理服务注册、发现和事件通知的核心逻辑
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Storage        | Storage          | 服务元数据存储接口实现            |
// | Events         | chan Event       | 服务事件通知通道                  |
// | Watchers       | map[string][]chan| 服务监听器集合，按服务名分组      |
// |                | Event            |                                   |
// | Mu             | sync.RWMutex     | 并发安全读写锁                    |
// | Ctx            | context.Context  | 上下文，用于控制组件生命周期      |
// | Cancel         | context.CancelFunc| 取消函数，用于停止服务发现组件   |
// +----------------+------------------+-----------------------------------+
type Discovery struct {
	Storage  Storage
	Events   chan Event
	Watchers map[string][]chan Event
	Mu       sync.RWMutex
	Ctx      context.Context
	Cancel   context.CancelFunc
}

// NewDiscovery 创建服务发现实例
// 初始化服务发现组件并返回其实例
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | storage        | 服务元数据存储接口实现            |
// | 返回值         | 初始化后的Discovery实例           |
// +----------------+-----------------------------------+
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
// 定义服务发现过程中的通信消息格式
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Type           | MessageType      | 消息类型                          |
// | Service        | *Service         | 服务实例信息（请求时）            |
// | Response       | []*Service       | 服务列表响应（查询时）            |
// | Error          | string           | 错误信息（如有）                  |
// +----------------+------------------+-----------------------------------+
type Message struct {
	Type     MessageType
	Service  *Service
	Response []*Service
	Error    string
}

// ServiceEvent 服务事件结构体
// 服务状态变更事件的另一种表示形式
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Type           | EventType        | 事件类型（注册/注销）             |
// | Service        | *Service         | 涉及的服务实例                    |
// +----------------+------------------+-----------------------------------+
type ServiceEvent struct {
	Type    EventType // 事件类型（注册/注销）
	Service *Service  // 涉及的服务实例
}
