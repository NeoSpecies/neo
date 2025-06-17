package discovery

import (
	"context"
	"errors"
	"sync"
	"time"
)

// InMemoryStorage 内存存储实现
type InMemoryStorage struct {
	services map[string]*Service // key: serviceID
	mu       sync.RWMutex
}

// NewInMemoryStorage 创建内存存储
func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		services: make(map[string]*Service),
	}
}

// Register 注册服务
func (s *InMemoryStorage) Register(ctx context.Context, service *Service) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.services[service.ID] = service
	return nil
}

// Deregister 注销服务
func (s *InMemoryStorage) Deregister(ctx context.Context, serviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceID]; !ok {
		return errors.New("service not found")
	}
	delete(s.services, serviceID)
	return nil
}

// Get 获取单个服务
func (s *InMemoryStorage) Get(ctx context.Context, serviceID string) (*Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	service, ok := s.services[serviceID]
	if !ok {
		return nil, errors.New("service not found")
	}
	return service, nil
}

// List 按名称列表服务
func (s *InMemoryStorage) List(ctx context.Context, serviceName string) ([]*Service, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Service
	for _, service := range s.services {
		if service.Name == serviceName && service.ExpireAt.After(time.Now()) {
			result = append(result, service)
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
