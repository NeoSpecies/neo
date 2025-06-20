package discovery

import (
	"context"
	"sync"
	"time"
)

// Service 服务元数据
type Service struct {
	ID        string            `json:"id"`         // 全局唯一ID
	Name      string            `json:"name"`       // 服务名称
	Address   string            `json:"address"`    // IPC地址（如unix:///tmp/service.sock）
	Port      int               `json:"port"`       // 新增：服务端口号
	Metadata  map[string]string `json:"metadata"`   // 扩展元数据
	Status    string            `json:"status"`     // 状态（healthy/unhealthy）
	ExpireAt  time.Time         `json:"expire_at"`  // 租约过期时间
	UpdatedAt time.Time         `json:"updated_at"` // 最后更新时间
}

// Storage 服务存储接口
type Storage interface {
	Register(ctx context.Context, s *Service) error                   // 注册服务
	Deregister(ctx context.Context, serviceID string) error           // 注销服务
	Get(ctx context.Context, serviceID string) (*Service, error)      // 获取单个服务
	List(ctx context.Context, serviceName string) ([]*Service, error) // 按名称列表服务
	Renew(ctx context.Context, serviceID string) error                // 续租服务
}

// ServiceEvent 服务变更事件类型
type ServiceEvent struct {
	Type    EventType // 事件类型（注册/注销）
	Service *Service  // 涉及的服务实例
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
	Type    EventType // 事件类型（注册/注销/过期）
	Service *Service  // 关联的服务实例
}

// Discovery 服务发现核心组件
type Discovery struct {
	storage  Storage
	events   chan Event // 修正：字段名去掉前导点，使用合法标识符
	watchers map[string][]chan Event
	mu       sync.RWMutex
	ctx      context.Context
	cancel   context.CancelFunc
}

// New 创建服务发现实例
func New(storage Storage) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &Discovery{
		storage:  storage,
		events:   make(chan Event, 100),
		watchers: make(map[string][]chan Event),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Register 注册服务（通过IPC调用）
// Register 注册服务（通过IPC调用）
func (d *Discovery) Register(ctx context.Context, s *Service) error {
	s.UpdatedAt = time.Now()
	s.ExpireAt = s.UpdatedAt.Add(30 * time.Second) // 默认30秒租约
	if err := d.storage.Register(ctx, s); err != nil {
		return err
	}
	// 发送注册事件（使用新定义的Event类型和EventRegistered常量）
	d.events <- Event{Type: EventRegistered, Service: s}
	return nil
}

// Watch 监听指定服务的变更事件
func (d *Discovery) Watch(serviceName string) <-chan Event {
	d.mu.Lock()
	defer d.mu.Unlock()
	ch := make(chan Event, 10) // 通道类型为Event
	d.watchers[serviceName] = append(d.watchers[serviceName], ch)
	go func() {
		for {
			select {
			case event := <-d.events:
				if event.Service.Name == serviceName {
					select {
					case ch <- event:
					default:
						// 通道满时丢弃旧事件
					}
				}
			case <-d.ctx.Done():
				close(ch)
				return
			}
		}
	}()
	return ch
}

// Close 关闭服务发现
func (d *Discovery) Close() {
	d.cancel()
	d.storage.(*InMemoryStorage).Close() // 假设使用内存存储
}

// GetServices 根据服务名称获取所有注册的服务实例
// 参数：serviceName - 目标服务名称
// 返回：服务实例列表，错误信息
func (d *Discovery) GetServices(serviceName string) ([]*Service, error) {
	return d.storage.List(d.ctx, serviceName) // 使用Discovery持有的上下文和Storage.List方法
}
