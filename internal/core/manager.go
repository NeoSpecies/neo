package core

import (
	"context"
	"fmt"
	"neo/internal/types"
	"neo/internal/utils"
	"sync"
	"time"
)

// ServiceManager 服务管理器接口
type ServiceManager interface {
	// RegisterService 注册服务
	RegisterService(service Service) error
	// UnregisterService 注销服务
	UnregisterService(name string) error
	// GetService 获取服务
	GetService(name string) (Service, error)
	// ListServices 列出所有服务
	ListServices() []string
	// RouteRequest 路由请求到对应服务
	RouteRequest(ctx context.Context, serviceName string, req types.Request) (types.Response, error)
	// Close 关闭管理器
	Close() error
	// GetStats 获取统计信息
	GetStats() ManagerStats
}

// ManagerStats 管理器统计信息
type ManagerStats struct {
	TotalServices    int
	ActiveServices   int
	TotalRequests    int64
	SuccessRequests  int64
	ErrorRequests    int64
	AverageLatency   time.Duration
	LastRequestTime  time.Time
}

// serviceManager 服务管理器实现
type serviceManager struct {
	services map[string]Service
	metrics  *ManagerMetrics
	logger   utils.Logger
	mu       sync.RWMutex
	closed   bool
}

// ManagerMetrics 管理器指标
type ManagerMetrics struct {
	mu              sync.RWMutex
	totalRequests   int64
	successRequests int64
	errorRequests   int64
	totalLatency    time.Duration
	lastRequestTime time.Time
}

// NewServiceManager 创建服务管理器
func NewServiceManager(logger utils.Logger) ServiceManager {
	if logger == nil {
		logger = utils.DefaultLogger
	}

	return &serviceManager{
		services: make(map[string]Service),
		metrics:  &ManagerMetrics{},
		logger:   logger,
	}
}

// RegisterService 注册服务
func (sm *serviceManager) RegisterService(service Service) error {
	if service == nil {
		return fmt.Errorf("service cannot be nil")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.closed {
		return fmt.Errorf("service manager is closed")
	}

	name := service.Name()
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if _, exists := sm.services[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}

	sm.services[name] = service
	sm.logger.Info("service registered",
		utils.String("service", name),
		utils.Int("totalServices", len(sm.services)))

	return nil
}

// UnregisterService 注销服务
func (sm *serviceManager) UnregisterService(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.closed {
		return fmt.Errorf("service manager is closed")
	}

	service, exists := sm.services[name]
	if !exists {
		return fmt.Errorf("service %s not found", name)
	}

	// 关闭服务
	if err := service.Close(); err != nil {
		sm.logger.Error("failed to close service",
			utils.String("service", name),
			utils.String("error", err.Error()))
	}

	delete(sm.services, name)
	sm.logger.Info("service unregistered",
		utils.String("service", name),
		utils.Int("totalServices", len(sm.services)))

	return nil
}

// GetService 获取服务
func (sm *serviceManager) GetService(name string) (Service, error) {
	if name == "" {
		return nil, fmt.Errorf("service name cannot be empty")
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.closed {
		return nil, fmt.Errorf("service manager is closed")
	}

	service, exists := sm.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// ListServices 列出所有服务
func (sm *serviceManager) ListServices() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	services := make([]string, 0, len(sm.services))
	for name := range sm.services {
		services = append(services, name)
	}

	return services
}

// RouteRequest 路由请求到对应服务
func (sm *serviceManager) RouteRequest(ctx context.Context, serviceName string, req types.Request) (types.Response, error) {
	start := time.Now()
	
	// 更新指标
	sm.metrics.incTotalRequests()
	sm.metrics.updateLastRequestTime(start)

	// 获取服务
	service, err := sm.GetService(serviceName)
	if err != nil {
		sm.metrics.incErrorRequests()
		return types.Response{
			ID:     req.ID,
			Status: 404,
			Error:  fmt.Sprintf("service not found: %v", err),
		}, nil
	}

	// 处理请求
	resp, err := service.HandleRequest(ctx, req)
	
	// 更新指标
	duration := time.Since(start)
	sm.metrics.updateLatency(duration)
	
	if err != nil || resp.Status >= 400 {
		sm.metrics.incErrorRequests()
	} else {
		sm.metrics.incSuccessRequests()
	}

	return resp, err
}

// Close 关闭管理器
func (sm *serviceManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.closed {
		return nil
	}

	sm.closed = true

	// 关闭所有服务
	for name, service := range sm.services {
		if err := service.Close(); err != nil {
			sm.logger.Error("failed to close service",
				utils.String("service", name),
				utils.String("error", err.Error()))
		}
	}

	sm.services = nil
	sm.logger.Info("service manager closed")
	return nil
}

// GetStats 获取统计信息
func (sm *serviceManager) GetStats() ManagerStats {
	sm.mu.RLock()
	totalServices := len(sm.services)
	activeServices := 0
	for _, service := range sm.services {
		// 简单判断服务是否活跃（可以扩展为更复杂的逻辑）
		if service != nil {
			activeServices++
		}
	}
	sm.mu.RUnlock()

	totalReqs, successReqs, errorReqs, avgLatency, lastReq := sm.metrics.getStats()

	return ManagerStats{
		TotalServices:   totalServices,
		ActiveServices:  activeServices,
		TotalRequests:   totalReqs,
		SuccessRequests: successReqs,
		ErrorRequests:   errorReqs,
		AverageLatency:  avgLatency,
		LastRequestTime: lastReq,
	}
}

// ManagerMetrics 方法实现

func (m *ManagerMetrics) incTotalRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalRequests++
}

func (m *ManagerMetrics) incSuccessRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successRequests++
}

func (m *ManagerMetrics) incErrorRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorRequests++
}

func (m *ManagerMetrics) updateLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLatency += latency
}

func (m *ManagerMetrics) updateLastRequestTime(t time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastRequestTime = t
}

func (m *ManagerMetrics) getStats() (total, success, errors int64, avgLatency time.Duration, lastRequest time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	total = m.totalRequests
	success = m.successRequests
	errors = m.errorRequests
	lastRequest = m.lastRequestTime
	
	if m.totalRequests > 0 {
		avgLatency = m.totalLatency / time.Duration(m.totalRequests)
	}
	
	return
}