package discovery

import (
	"context"
	"fmt"
	"neo/internal/types"
	"time"
)

// Service 服务元数据
type Service struct {
	ID        string            `json:"id"`         // 全局唯一ID
	Name      string            `json:"name"`       // 服务名称
	Address   string            `json:"address"`    // IPC地址（如unix:///tmp/service.sock）
	Port      int               `json:"port"`       // 服务端口号
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
	storage  types.Storage
	events   chan types.Event
	watchers map[string][]chan types.Event
	ctx      context.Context
	cancel   context.CancelFunc
}

// New 创建服务发现实例
func New(storage types.Storage) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &Discovery{
		storage:  storage,
	events:   make(chan types.Event, 100),
	watchers: make(map[string][]chan types.Event),
	ctx:      ctx,
	cancel:   cancel,
	}
}

// DiscoveryService 包装结构体用于实现接口
type DiscoveryService struct {
	*types.Discovery
}

// Register 注册服务
func (d *DiscoveryService) Register(ctx context.Context, s *types.Service) error {
	fmt.Printf("[DEBUG] 收到服务注册请求: ID=%s, Name=%s, Address=%s:%d\n", s.ID, s.Name, s.Address, s.Port)
	fmt.Printf("[DEBUG] 注册元数据: %+v\n", s.Metadata)

	s.UpdatedAt = time.Now()
	s.ExpireAt = s.UpdatedAt.Add(30 * time.Second)
	if err := d.Storage.Register(ctx, s); err != nil {
		fmt.Printf("[ERROR] 服务注册失败: %v\n", err)
		return err
	}

	fmt.Printf("[DEBUG] 服务注册成功: ID=%s, 过期时间=%v\n", s.ID, s.ExpireAt)
	d.Events <- types.Event{Type: types.EventRegistered, Service: s}
	return nil
}

// Watch 监听服务事件
func (d *DiscoveryService) Watch(serviceName string) <-chan types.Event {
	d.Mu.Lock()
	defer d.Mu.Unlock()
	ch := make(chan types.Event, 10)
	d.Watchers[serviceName] = append(d.Watchers[serviceName], ch)
	fmt.Printf("[DEBUG] 新增服务监听器: 服务名称=%s, 监听器数量=%d\n", serviceName, len(d.Watchers[serviceName]))

	go func() {
		for {
			select {
			case event := <-d.Events:
				if event.Service.Name == serviceName {
					fmt.Printf("[DEBUG] 服务事件触发: 类型=%v, 服务名称=%s, ID=%s\n", event.Type, event.Service.Name, event.Service.ID)
					select {
					case ch <- event:
					default:
						fmt.Printf("[WARN] 事件通道已满，丢弃事件: %v\n", event.Type)
					}
				}
			case <-d.Ctx.Done():
				close(ch)
				fmt.Printf("[DEBUG] 监听器已关闭: 服务名称=%s\n", serviceName)
				return
			}
		}
	}()
	return ch
}

// GetServices 获取服务列表
func (d *DiscoveryService) GetServices(serviceName string) ([]*types.Service, error) {
	return d.Storage.List(d.Ctx, serviceName)
}
