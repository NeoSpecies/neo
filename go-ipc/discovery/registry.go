package discovery

import (
	"context"
	"encoding/hex"
	"math/rand"
	"sync"
	"time"
)

// RegistryConfig 注册配置
type RegistryConfig struct {
	Name        string            // 服务名称
	Address     string            // 服务地址
	Port        int               // 服务端口
	Version     string            // 服务版本
	Metadata    map[string]string // 服务元数据
	HealthCheck bool              // 是否启用健康检查
}

// ServiceRegistry 服务注册器
type ServiceRegistry struct {
	discovery  *ServiceDiscovery
	service    *ServiceInfo
	config     RegistryConfig  // 添加config字段
	healthChan chan struct{}   // 添加健康检查通道
	ctx        context.Context // 添加上下文
	mutex      sync.Mutex      // 添加互斥锁
}

// generateID 生成服务实例唯一ID
// generateID 生成服务实例唯一ID
func generateID() string { // 添加func关键字
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + ":" + time.Now().Format("150405")
}

// NewServiceRegistry 创建服务注册器
func NewServiceRegistry(config RegistryConfig) (*ServiceRegistry, error) {
	// 修复GetInstance调用，移除参数
	discovery := GetInstance()

	service := &ServiceInfo{
		Name:      config.Name,
		ID:        generateID(), // 使用生成的ID
		Address:   config.Address,
		Port:      config.Port,
		Version:   config.Version,
		Metadata:  config.Metadata,
		Status:    "healthy",
		StartTime: time.Now(),
	}

	// 注册服务
	if err := discovery.Register(service); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 修复结构体初始化
	registry := &ServiceRegistry{
		discovery:  discovery,
		service:    service,
		config:     config,
		healthChan: make(chan struct{}),
		ctx:        ctx,
	}

	// 暂时注释掉健康检查相关代码，或实现该方法
	// 启动健康检查
	// if config.HealthCheck {
	//     registry.startHealthCheck()
	// }

	return registry, nil
}

// Deregister 注销服务
func (r *ServiceRegistry) Deregister() error {
	return r.discovery.Deregister(r.service)
}

// UpdateStatus 更新服务状态
func (r *ServiceRegistry) UpdateStatus(status string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.service.Status = status
	return r.discovery.Register(r.service) // 重新注册以更新状态
}

// UpdateMetadata 更新服务元数据
func (r *ServiceRegistry) UpdateMetadata(metadata map[string]string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.service.Metadata = metadata
	return r.discovery.Register(r.service) // 重新注册以更新元数据
}

// ReportHealth 报告健康状态
func (r *ServiceRegistry) ReportHealth() {
	select {
	case r.healthChan <- struct{}{}:
	default:
	}
}
