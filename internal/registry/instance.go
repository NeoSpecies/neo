package registry

import (
	"fmt"
	"time"
)

// ServiceStatus 服务状态
type ServiceStatus int

const (
	// StatusHealthy 健康状态
	StatusHealthy ServiceStatus = iota
	// StatusUnhealthy 不健康状态
	StatusUnhealthy
	// StatusUnknown 未知状态
	StatusUnknown
)

// HealthCheck 健康检查配置
type HealthCheck struct {
	// Interval 健康检查间隔
	Interval time.Duration
	// Timeout 健康检查超时时间
	Timeout time.Duration
	// MaxRetries 最大重试次数
	MaxRetries int
	// Path 健康检查路径（HTTP）
	Path string
}

// ServiceInstance 服务实例
type ServiceInstance struct {
	// ID 实例唯一标识
	ID string
	// Name 服务名称
	Name string
	// Address 服务地址
	Address string
	// Port 服务端口
	Port int
	// Metadata 元数据
	Metadata map[string]string
	// HealthCheck 健康检查配置
	HealthCheck HealthCheck
	// RegisterTime 注册时间
	RegisterTime time.Time
	// LastHeartbeat 最后心跳时间
	LastHeartbeat time.Time
	// Status 服务状态
	Status ServiceStatus
	// Version 服务版本
	Version string
	// Weight 权重（用于负载均衡）
	Weight int
}

// GetFullAddress 获取完整地址
func (s *ServiceInstance) GetFullAddress() string {
	if s.Port > 0 {
		return fmt.Sprintf("%s:%d", s.Address, s.Port)
	}
	return s.Address
}

// IsHealthy 判断是否健康
func (s *ServiceInstance) IsHealthy() bool {
	return s.Status == StatusHealthy
}

// Clone 克隆服务实例
func (s *ServiceInstance) Clone() *ServiceInstance {
	metadata := make(map[string]string, len(s.Metadata))
	for k, v := range s.Metadata {
		metadata[k] = v
	}
	
	return &ServiceInstance{
		ID:            s.ID,
		Name:          s.Name,
		Address:       s.Address,
		Port:          s.Port,
		Metadata:      metadata,
		HealthCheck:   s.HealthCheck,
		RegisterTime:  s.RegisterTime,
		LastHeartbeat: s.LastHeartbeat,
		Status:        s.Status,
		Version:       s.Version,
		Weight:        s.Weight,
	}
}