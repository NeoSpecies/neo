package discovery

import (
	"context"
	"fmt"
	types "neo/internal/types"
	"time"
)

// Deprecated: 请使用 types.Service (计划于v2.0移除)
type Service = types.Service

// Deprecated: 请使用 types.Storage (计划于v2.0移除)
type Storage = types.Storage

// Deprecated: 请使用 types.ServiceEvent (计划于v2.0移除)
type ServiceEvent = types.ServiceEvent

// Deprecated: 请使用 types.EventType (计划于v2.0移除)
type EventType = types.EventType

// Deprecated: 请使用 types.Discovery (计划于v2.0移除)
type Discovery = types.Discovery

// New 创建服务发现实例
func New(storage types.Storage) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())
	return &Discovery{
		Storage:  storage,
		Events:   make(chan types.Event, 100),
		Watchers: make(map[string][]chan types.Event),
		Ctx:      ctx,
		Cancel:   cancel,
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
