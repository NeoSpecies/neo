package discovery

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-ipc/config"
	"go-ipc/pool"
)

// ResolverConfig 解析器配置
type ResolverConfig struct {
	Name            string                   // 服务名称
	LoadBalance     pool.LoadBalanceStrategy // 负载均衡策略
	FilterFunc      FilterFunc               // 服务过滤函数
	RefreshInterval time.Duration            // 刷新间隔
}

// FilterFunc 服务过滤函数
type FilterFunc func(*ServiceInfo) bool

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

// NewServiceResolver 创建服务解析器
func NewServiceResolver(resolverConfig ResolverConfig) (*ServiceResolver, error) {
	sd := GetInstance()

	// 从全局配置获取刷新间隔
	cfg := config.GetDiscoveryConfig()
	if resolverConfig.RefreshInterval == 0 {
		resolverConfig.RefreshInterval = time.Duration(cfg.RefreshInterval) * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	resolver := &ServiceResolver{
		discovery: sd,
		name:      resolverConfig.Name,
		balancer:  pool.NewBalancer(resolverConfig.LoadBalance),
		services:  make(map[string]*ServiceInfo),
		ctx:       ctx,
		cancel:    cancel,
	}

	// 初始化服务列表
	if err := resolver.refresh(); err != nil {
		cancel()
		return nil, err
	}

	// 启动服务监听协程
	go func() {
		sd.watch()
	}()

	// 定期刷新服务列表
	if resolverConfig.RefreshInterval > 0 {
		go resolver.refreshLoop(resolverConfig.RefreshInterval, resolverConfig.FilterFunc)
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
// 修改refresh方法
func (r *ServiceResolver) refresh() error {
	services, err := r.discovery.GetServices(r.name)
	if err != nil {
		return err
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	// 过滤过期服务
	now := time.Now()
	r.services = make(map[string]*ServiceInfo)
	for _, service := range services {
		if service.ExpireTime.After(now) && service.Status == "healthy" {
			r.services[service.ID] = service
		}
	}

	return nil
}
