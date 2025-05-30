package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-ipc/pool"
)

// ServiceResolver 服务解析器
type ServiceResolver struct {
	discovery *ServiceDiscovery
	name      string
	balancer  pool.Balancer
	services  map[string]*ServiceInfo
	mutex     sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// ResolverConfig 解析器配置
type ResolverConfig struct {
	Name         string                  // 服务名称
	LoadBalance  pool.LoadBalanceStrategy // 负载均衡策略
	FilterFunc   FilterFunc              // 服务过滤函数
	RefreshInterval time.Duration        // 刷新间隔
}

// FilterFunc 服务过滤函数
type FilterFunc func(*ServiceInfo) bool

// NewServiceResolver 创建服务解析器
func NewServiceResolver(discovery *ServiceDiscovery, config ResolverConfig) (*ServiceResolver, error) {
	ctx, cancel := context.WithCancel(context.Background())

	resolver := &ServiceResolver{
		discovery: discovery,
		name:      config.Name,
		balancer:  pool.NewBalancer(config.LoadBalance),
		services:  make(map[string]*ServiceInfo),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 初始化服务列表
	if err := resolver.refresh(); err != nil {
		cancel()
		return nil, err
	}

	// 监听服务变更
	updateCh, err := discovery.Watch(config.Name)
	if err != nil {
		cancel()
		return nil, err
	}

	// 处理服务更新
	go resolver.watchUpdates(updateCh, config.FilterFunc)

	// 定期刷新服务列表
	if config.RefreshInterval > 0 {
		go resolver.refreshLoop(config.RefreshInterval, config.FilterFunc)
	}

	return resolver, nil
}

// Resolve 解析服务地址
func (r *ServiceResolver) Resolve() (*ServiceInfo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if len(r.services) == 0 {
		return nil, fmt.Errorf("no available services for %s", r.name)
	}

	// 将服务列表转换为连接列表（用于负载均衡）
	conns := make([]*pool.Connection, 0, len(r.services))
	for _, service := range r.services {
		if service.Status == "healthy" {
			conn := &pool.Connection{
				Stats: &pool.ConnectionStats{
					LastUsedTime: time.Now(),
				},
			}
			conns = append(conns, conn)
		}
	}

	if len(conns) == 0 {
		return nil, fmt.Errorf("no healthy services for %s", r.name)
	}

	// 使用负载均衡器选择服务
	selectedConn := r.balancer.Select(conns)
	if selectedConn == nil {
		return nil, fmt.Errorf("failed to select service for %s", r.name)
	}

	// 找到对应的服务信息
	for _, service := range r.services {
		if service.Status == "healthy" {
			return service, nil
		}
	}

	return nil, fmt.Errorf("no available service found for %s", r.name)
}

// GetServices 获取所有服务
func (r *ServiceResolver) GetServices() []*ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	services := make([]*ServiceInfo, 0, len(r.services))
	for _, service := range r.services {
		services = append(services, service)
	}
	return services
}

// watchUpdates 监听服务更新
func (r *ServiceResolver) watchUpdates(updateCh <-chan *ServiceUpdate, filter FilterFunc) {
	for {
		select {
		case <-r.ctx.Done():
			return
		case update := <-updateCh:
			if update == nil {
				continue
			}

			r.mutex.Lock()
			switch update.Type {
			case ServiceAdded, ServiceModified:
				if filter == nil || filter(update.Service) {
					r.services[update.Service.ID] = update.Service
				}
			case ServiceRemoved:
				delete(r.services, update.Service.ID)
			}
			r.mutex.Unlock()
		}
	}
}

// refreshLoop 定期刷新服务列表
func (r *ServiceResolver) refreshLoop(interval time.Duration, filter FilterFunc) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			if err := r.refresh(); err != nil {
				// 刷新失败，继续使用现有服务列表
				continue
			}

			// 应用过滤器
			if filter != nil {
				r.mutex.Lock()
				for id, service := range r.services {
					if !filter(service) {
						delete(r.services, id)
					}
				}
				r.mutex.Unlock()
			}
		}
	}
}

// refresh 刷新服务列表
func (r *ServiceResolver) refresh() error {
	services, err := r.discovery.GetServices(r.name)
	if err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 更新服务列表
	newServices := make(map[string]*ServiceInfo)
	for _, service := range services {
		newServices[service.ID] = service
	}
	r.services = newServices

	return nil
}

// Close 关闭解析器
func (r *ServiceResolver) Close() error {
	r.cancel()
	return nil
} 