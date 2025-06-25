package discovery

import (
	"context"
	"errors"
	"log"
	"neo/internal/types"
	"sync"
	"time"
)

// InMemoryStorage 内存存储实现
type InMemoryStorage struct {
	services map[string]*types.Service // key: serviceID
	mu       sync.RWMutex
}

// NewInMemoryStorage 创建内存存储
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		services: make(map[string]*types.Service),
	}
}

// Register 注册服务
func (s *InMemoryStorage) Register(ctx context.Context, service *types.Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[service.ID] = service
	// 新增：打印注册的服务信息
	log.Printf("服务已注册: ID=%s, Name=%s, Address=%s:%d, Status=%s, ExpiresAt=%v",
		service.ID, service.Name, service.Address, service.Port, service.Status, service.ExpireAt)
	return nil
}

// Get 获取单个服务
func (s *InMemoryStorage) Get(ctx context.Context, serviceID string) (*types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, ok := s.services[serviceID]
	if !ok {
		return nil, errors.New("service not found")
	}
	return service, nil
}

// List 按名称列表服务
func (s *InMemoryStorage) List(ctx context.Context, serviceName string) ([]*types.Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*types.Service
	for _, service := range s.services {
		if service.Name == serviceName && service.ExpireAt.After(time.Now()) {
			result = append(result, service)
			log.Printf("Registered service: %+v\n", service)
		}
	}
	return result, nil
}

// Renew 续租服务
func (s *InMemoryStorage) Renew(ctx context.Context, serviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	service, ok := s.services[serviceID]
	if !ok {
		return errors.New("service not found")
	}
	service.ExpireAt = time.Now().Add(30 * time.Second)
	return nil
}

// Close 关闭存储
func (s *InMemoryStorage) Close() {
	s.mu.Lock()
	s.services = nil
	s.mu.Unlock()
}

// Deregister 注销服务
func (s *InMemoryStorage) Deregister(ctx context.Context, serviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceID]; !ok {
		return errors.New("service not found")
	}
	delete(s.services, serviceID)
	log.Printf("服务已注销: ID=%s", serviceID) // 添加注销日志
	return nil
}
