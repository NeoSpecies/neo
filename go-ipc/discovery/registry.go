package discovery

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ServiceRegistry 服务注册器
type ServiceRegistry struct {
	discovery  *ServiceDiscovery
	service    *ServiceInfo
	healthChan chan struct{}
	mutex      sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

// RegistryConfig 注册配置
type RegistryConfig struct {
	Name        string            // 服务名称
	Address     string            // 服务地址
	Port        int              // 服务端口
	Version     string            // 服务版本
	Metadata    map[string]string // 服务元数据
	HealthCheck bool              // 是否启用健康检查
}

// NewServiceRegistry 创建服务注册器
func NewServiceRegistry(discovery *ServiceDiscovery, config RegistryConfig) (*ServiceRegistry, error) {
	// 生成服务实例ID
	hostname, _ := os.Hostname()
	id := fmt.Sprintf("%s-%s", hostname, uuid.New().String())

	// 创建服务信息
	service := &ServiceInfo{
		Name:      config.Name,
		ID:        id,
		Address:   config.Address,
		Port:      config.Port,
		Version:   config.Version,
		Metadata:  config.Metadata,
		Status:    "starting",
		StartTime: time.Now(),
	}

	ctx, cancel := context.WithCancel(context.Background())

	registry := &ServiceRegistry{
		discovery:  discovery,
		service:    service,
		healthChan: make(chan struct{}, 1),
		ctx:        ctx,
		cancel:     cancel,
	}

	// 注册服务
	if err := registry.register(); err != nil {
		cancel()
		return nil, err
	}

	// 启动健康检查
	if config.HealthCheck {
		go registry.healthCheck()
	}

	return registry, nil
}

// register 注册服务
func (r *ServiceRegistry) register() error {
	return r.discovery.Register(r.service)
}

// Deregister 注销服务
func (r *ServiceRegistry) Deregister() error {
	r.cancel()
	return r.discovery.Deregister(r.service)
}

// UpdateStatus 更新服务状态
func (r *ServiceRegistry) UpdateStatus(status string) error {
	r.mutex.Lock()
	r.service.Status = status
	r.mutex.Unlock()

	return r.register()
}

// UpdateMetadata 更新服务元数据
func (r *ServiceRegistry) UpdateMetadata(metadata map[string]string) error {
	r.mutex.Lock()
	r.service.Metadata = metadata
	r.mutex.Unlock()

	return r.register()
}

// ReportHealth 报告健康状态
func (r *ServiceRegistry) ReportHealth() {
	select {
	case r.healthChan <- struct{}{}:
	default:
	}
}

// healthCheck 健康检查
func (r *ServiceRegistry) healthCheck() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(DefaultTTL * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-r.healthChan:
			// 收到健康报告，重置超时
			if !timeout.Stop() {
				<-timeout.C
			}
			timeout.Reset(DefaultTTL * time.Second)
		case <-timeout.C:
			// 超时，服务不健康
			log.Printf("Service %s health check timeout", r.service.ID)
			r.UpdateStatus("unhealthy")
		case <-ticker.C:
			// 定期检查服务可用性
			if err := r.checkServiceAvailability(); err != nil {
				log.Printf("Service %s availability check failed: %v", r.service.ID, err)
				r.UpdateStatus("unavailable")
			} else {
				r.UpdateStatus("healthy")
			}
		}
	}
}

// checkServiceAvailability 检查服务可用性
func (r *ServiceRegistry) checkServiceAvailability() error {
	// 尝试连接服务地址
	address := fmt.Sprintf("%s:%d", r.service.Address, r.service.Port)
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// GetServiceInfo 获取服务信息
func (r *ServiceRegistry) GetServiceInfo() *ServiceInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.service
} 