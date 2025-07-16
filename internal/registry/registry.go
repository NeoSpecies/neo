package registry

import (
	"context"
	"fmt"
	"neo/internal/utils"
	"sync"
	"time"
)

// ServiceRegistry 服务注册中心接口
type ServiceRegistry interface {
	// Register 注册服务实例
	Register(ctx context.Context, instance *ServiceInstance) error
	// Deregister 注销服务实例
	Deregister(ctx context.Context, instanceID string) error
	// Discover 发现服务实例
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
	// HealthCheck 健康检查
	HealthCheck(ctx context.Context, instanceID string) error
	// UpdateInstance 更新服务实例
	UpdateInstance(ctx context.Context, instance *ServiceInstance) error
	// GetInstance 获取指定实例
	GetInstance(ctx context.Context, instanceID string) (*ServiceInstance, error)
	// ListServices 列出所有服务
	ListServices(ctx context.Context) ([]string, error)
}

// inMemoryRegistry 内存实现的服务注册中心
type inMemoryRegistry struct {
	mu                  sync.RWMutex
	services            map[string]map[string]*ServiceInstance // serviceName -> instanceID -> instance
	instances           map[string]*ServiceInstance            // instanceID -> instance
	watchers            map[string][]chan ServiceEvent        // serviceName -> watchers
	logger              utils.Logger
	healthCheckFunc     HealthCheckFunc
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	cleanupInterval     time.Duration // 清理间隔
	instanceExpiry      time.Duration // 实例过期时间
	healthCheckInterval time.Duration // 健康检查间隔
}

// HealthCheckFunc 健康检查函数类型
type HealthCheckFunc func(ctx context.Context, instance *ServiceInstance) error

// RegistryOption 注册中心选项
type RegistryOption func(*inMemoryRegistry)

// WithLogger 设置日志器
func WithLogger(logger utils.Logger) RegistryOption {
	return func(r *inMemoryRegistry) {
		r.logger = logger
	}
}

// WithHealthCheckFunc 设置健康检查函数
func WithHealthCheckFunc(fn HealthCheckFunc) RegistryOption {
	return func(r *inMemoryRegistry) {
		r.healthCheckFunc = fn
	}
}

// RegistryConfig 注册中心配置
type RegistryConfig struct {
	CleanupInterval     time.Duration
	InstanceExpiry      time.Duration
	HealthCheckInterval time.Duration
}

// WithConfig 设置配置
func WithConfig(config RegistryConfig) RegistryOption {
	return func(r *inMemoryRegistry) {
		if config.CleanupInterval > 0 {
			r.cleanupInterval = config.CleanupInterval
		}
		if config.InstanceExpiry > 0 {
			r.instanceExpiry = config.InstanceExpiry
		}
		if config.HealthCheckInterval > 0 {
			r.healthCheckInterval = config.HealthCheckInterval
		}
	}
}

// NewServiceRegistry 创建新的服务注册中心实例
func NewServiceRegistry(opts ...RegistryOption) ServiceRegistry {
	r := &inMemoryRegistry{
		services:            make(map[string]map[string]*ServiceInstance),
		instances:           make(map[string]*ServiceInstance),
		watchers:            make(map[string][]chan ServiceEvent),
		logger:              utils.DefaultLogger,
		stopCh:              make(chan struct{}),
		cleanupInterval:     10 * time.Second, // 默认值
		instanceExpiry:      5 * time.Minute,  // 默认值
		healthCheckInterval: 30 * time.Second, // 默认值
	}
	
	for _, opt := range opts {
		opt(r)
	}
	
	// 启动健康检查
	r.startHealthChecker()
	
	return r
}

// Register 注册服务实例
func (r *inMemoryRegistry) Register(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("instance cannot be nil")
	}
	
	if instance.ID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	
	if instance.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	
	if instance.Address == "" {
		return fmt.Errorf("service address cannot be empty")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 设置注册时间
	instance.RegisterTime = time.Now()
	instance.LastHeartbeat = time.Now()
	instance.Status = StatusHealthy
	
	// 初始化服务映射
	if _, exists := r.services[instance.Name]; !exists {
		r.services[instance.Name] = make(map[string]*ServiceInstance)
	}
	
	// 检查是否已存在
	oldInstance, exists := r.instances[instance.ID]
	
	// 保存实例
	instanceCopy := instance.Clone()
	r.services[instance.Name][instance.ID] = instanceCopy
	r.instances[instance.ID] = instanceCopy
	
	// 发送事件
	event := ServiceEvent{
		Type:      EventRegister,
		Service:   instance.Name,
		Instance:  instanceCopy,
		Timestamp: time.Now(),
	}
	
	if exists {
		event.Type = EventUpdate
		event.OldInstance = oldInstance
	}
	
	r.notifyWatchers(instance.Name, event)
	
	r.logger.Info("service instance registered",
		utils.String("service", instance.Name),
		utils.String("instanceID", instance.ID),
		utils.String("address", instance.Address))
	
	return nil
}

// Deregister 注销服务实例
func (r *inMemoryRegistry) Deregister(ctx context.Context, instanceID string) error {
	if instanceID == "" {
		return fmt.Errorf("instance ID cannot be empty")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	instance, exists := r.instances[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	
	// 删除实例
	delete(r.instances, instanceID)
	delete(r.services[instance.Name], instanceID)
	
	// 清理空的服务映射
	if len(r.services[instance.Name]) == 0 {
		delete(r.services, instance.Name)
	}
	
	// 发送事件
	r.notifyWatchers(instance.Name, ServiceEvent{
		Type:      EventDeregister,
		Service:   instance.Name,
		Instance:  instance,
		Timestamp: time.Now(),
	})
	
	r.logger.Info("service instance deregistered",
		utils.String("service", instance.Name),
		utils.String("instanceID", instanceID))
	
	return nil
}

// Discover 发现服务实例
func (r *inMemoryRegistry) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	instances, exists := r.services[serviceName]
	if !exists {
		return []*ServiceInstance{}, nil
	}
	
	// 返回健康的实例
	result := make([]*ServiceInstance, 0, len(instances))
	for _, instance := range instances {
		if instance.IsHealthy() {
			result = append(result, instance.Clone())
		}
	}
	
	return result, nil
}

// Watch 监听服务变化
func (r *inMemoryRegistry) Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 创建事件通道
	eventCh := make(chan ServiceEvent, 10)
	r.watchers[serviceName] = append(r.watchers[serviceName], eventCh)
	
	// 启动goroutine处理context取消
	go func() {
		<-ctx.Done()
		r.mu.Lock()
		defer r.mu.Unlock()
		
		// 移除watcher
		watchers := r.watchers[serviceName]
		for i, ch := range watchers {
			if ch == eventCh {
				r.watchers[serviceName] = append(watchers[:i], watchers[i+1:]...)
				break
			}
		}
		
		close(eventCh)
	}()
	
	return eventCh, nil
}

// HealthCheck 健康检查
func (r *inMemoryRegistry) HealthCheck(ctx context.Context, instanceID string) error {
	r.mu.RLock()
	instance, exists := r.instances[instanceID]
	r.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	
	// 更新心跳时间
	r.mu.Lock()
	instance.LastHeartbeat = time.Now()
	r.mu.Unlock()
	
	// 执行健康检查
	if r.healthCheckFunc != nil {
		if err := r.healthCheckFunc(ctx, instance); err != nil {
			r.updateInstanceStatus(instance, StatusUnhealthy)
			return err
		}
	}
	
	r.updateInstanceStatus(instance, StatusHealthy)
	return nil
}

// UpdateInstance 更新服务实例
func (r *inMemoryRegistry) UpdateInstance(ctx context.Context, instance *ServiceInstance) error {
	if instance == nil || instance.ID == "" {
		return fmt.Errorf("invalid instance")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	oldInstance, exists := r.instances[instance.ID]
	if !exists {
		return fmt.Errorf("instance not found: %s", instance.ID)
	}
	
	// 保留某些字段
	instance.RegisterTime = oldInstance.RegisterTime
	instance.LastHeartbeat = time.Now()
	
	// 更新实例
	instanceCopy := instance.Clone()
	r.instances[instance.ID] = instanceCopy
	r.services[instance.Name][instance.ID] = instanceCopy
	
	// 发送更新事件
	r.notifyWatchers(instance.Name, ServiceEvent{
		Type:        EventUpdate,
		Service:     instance.Name,
		Instance:    instanceCopy,
		OldInstance: oldInstance,
		Timestamp:   time.Now(),
	})
	
	return nil
}

// GetInstance 获取指定实例
func (r *inMemoryRegistry) GetInstance(ctx context.Context, instanceID string) (*ServiceInstance, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("instance ID cannot be empty")
	}
	
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	instance, exists := r.instances[instanceID]
	if !exists {
		return nil, fmt.Errorf("instance not found: %s", instanceID)
	}
	
	return instance.Clone(), nil
}

// ListServices 列出所有服务
func (r *inMemoryRegistry) ListServices(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}
	
	return services, nil
}

// notifyWatchers 通知监听者
func (r *inMemoryRegistry) notifyWatchers(serviceName string, event ServiceEvent) {
	watchers := r.watchers[serviceName]
	for _, ch := range watchers {
		select {
		case ch <- event:
		default:
			// 通道满了，跳过
			r.logger.Warn("watcher channel full, dropping event",
				utils.String("service", serviceName),
				utils.String("event", event.Type.String()))
		}
	}
}

// updateInstanceStatus 更新实例状态
func (r *inMemoryRegistry) updateInstanceStatus(instance *ServiceInstance, status ServiceStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if instance.Status != status {
		oldStatus := instance.Status
		instance.Status = status
		
		// 发送健康状态变化事件
		r.notifyWatchers(instance.Name, ServiceEvent{
			Type:      EventHealthChange,
			Service:   instance.Name,
			Instance:  instance.Clone(),
			Timestamp: time.Now(),
		})
		
		r.logger.Info("instance status changed",
			utils.String("instanceID", instance.ID),
			utils.String("oldStatus", fmt.Sprintf("%d", oldStatus)),
			utils.String("newStatus", fmt.Sprintf("%d", status)))
	}
}

// startHealthChecker 启动健康检查器
func (r *inMemoryRegistry) startHealthChecker() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		
		ticker := time.NewTicker(r.cleanupInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				r.checkInstanceHealth()
			case <-r.stopCh:
				return
			}
		}
	}()
}

// checkInstanceHealth 检查实例健康状态
func (r *inMemoryRegistry) checkInstanceHealth() {
	r.mu.RLock()
	instances := make([]*ServiceInstance, 0, len(r.instances))
	for _, instance := range r.instances {
		instances = append(instances, instance)
	}
	r.mu.RUnlock()
	
	now := time.Now()
	for _, instance := range instances {
		// 检查心跳超时
		if now.Sub(instance.LastHeartbeat) > r.instanceExpiry {
			r.updateInstanceStatus(instance, StatusUnhealthy)
		}
	}
}

// Close 关闭注册中心
func (r *inMemoryRegistry) Close() error {
	close(r.stopCh)
	r.wg.Wait()
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// 关闭所有watcher通道
	for _, watchers := range r.watchers {
		for _, ch := range watchers {
			close(ch)
		}
	}
	
	return nil
}