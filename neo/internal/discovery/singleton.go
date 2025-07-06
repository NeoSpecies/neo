package discovery

import (
	"sync"
)

var (
	discoveryInstance *DiscoveryService
	once              sync.Once
)

// GetDiscoveryService 获取单例服务发现实例
func GetDiscoveryService() *DiscoveryService {
	once.Do(func() {
		// 初始化内存存储
		storage := NewInMemoryStorage()
		// 创建服务发现实例
		discovery := New(storage)
		discoveryInstance = &DiscoveryService{discovery}
	})
	return discoveryInstance
}
