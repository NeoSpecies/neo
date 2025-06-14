package discovery

import (
	"fmt"
	"sync"
)

type InMemoryStorage struct {
	services map[string]*ServiceInfo // key: serviceID
	updates  chan *ServiceUpdate
	mutex    sync.RWMutex
	ttl      int64
}

func NewInMemoryStorage(ttl int64) *InMemoryStorage {
	return &InMemoryStorage{
		services: make(map[string]*ServiceInfo),
		updates:  make(chan *ServiceUpdate, 100),
		ttl:      ttl,
	}
}

// Register 注册服务
func (m *InMemoryStorage) Register(service *ServiceInfo) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.services[service.ID] = service
	m.updates <- &ServiceUpdate{Type: ServiceAdded, Service: service}
	return nil
}

// Deregister 注销服务
func (m *InMemoryStorage) Deregister(serviceID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if service, exists := m.services[serviceID]; exists {
		delete(m.services, serviceID)
		m.updates <- &ServiceUpdate{Type: ServiceRemoved, Service: service}
		return nil
	}
	return fmt.Errorf("service %s not found", serviceID)
}

// GetService 获取服务信息
func (m *InMemoryStorage) GetService(name, id string) (*ServiceInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	for _, service := range m.services {
		if service.Name == name && service.ID == id {
			return service, nil
		}
	}
	return nil, fmt.Errorf("service %s/%s not found", name, id)
}

// GetServices 获取所有服务
func (m *InMemoryStorage) GetServices(name string) ([]*ServiceInfo, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	services := make([]*ServiceInfo, 0)
	for _, service := range m.services {
		if service.Name == name {
			services = append(services, service)
		}
	}
	return services, nil
}

// Watch 返回服务更新通道
func (m *InMemoryStorage) Watch() chan *ServiceUpdate {
	return m.updates
}

// Close 关闭存储
func (m *InMemoryStorage) Close() error {
	close(m.updates)
	return nil
}
