package discovery

import (
	"context"
	"go-ipc/config" // 添加配置包导入
	"sync"
	"time"
)

const (
	// DefaultTTL 服务注册的默认TTL
	DefaultTTL = 10 // seconds

	// DefaultRefreshInterval 服务刷新间隔
	DefaultRefreshInterval = 3 * time.Second
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name       string            `json:"name"`        // 服务名称
	ID         string            `json:"id"`          // 服务实例ID
	Address    string            `json:"address"`     // 服务地址
	Port       int               `json:"port"`        // 服务端口
	Version    string            `json:"version"`     // 服务版本
	Metadata   map[string]string `json:"metadata"`    // 服务元数据
	Status     string            `json:"status"`      // 服务状态
	StartTime  time.Time         `json:"start_time"`  // 启动时间
	ExpireTime time.Time         `json:"expire_time"` // 服务过期时间
}

// ServiceUpdate 服务更新事件
type ServiceUpdate struct {
	Type    UpdateType   // 更新类型
	Service *ServiceInfo // 服务信息
}

// UpdateType 更新类型
type UpdateType int

const (
	// ServiceAdded 服务添加
	ServiceAdded UpdateType = iota
	// ServiceRemoved 服务移除
	ServiceRemoved
	// ServiceModified 服务修改
	ServiceModified
)

// ServiceDiscovery 服务发现
// 新增存储接口抽象
type Storage interface {
	Register(service *ServiceInfo) error
	Deregister(serviceID string) error
	GetService(name, id string) (*ServiceInfo, error)
	GetServices(name string) ([]*ServiceInfo, error)
	Watch() chan *ServiceUpdate
	Close() error
}

// 修改ServiceDiscovery结构
type ServiceDiscovery struct {
	storage    Storage // 存储接口
	serviceKey string  // 服务键前缀
	ttl        int64   // 租约TTL
	mutex      sync.RWMutex
	watchers   map[string][]chan *ServiceUpdate
	ctx        context.Context
	cancel     context.CancelFunc
}

var (
	instance *ServiceDiscovery
	once     sync.Once
)

// GetInstance 获取服务发现单例实例
func GetInstance() *ServiceDiscovery {
	once.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		cfg := config.GetDiscoveryConfig()
		var storage Storage

		// 根据配置选择存储类型 - 暂时只保留内存存储实现
		storage = NewInMemoryStorage(int64(cfg.TTL)) // 添加类型转换

		instance = &ServiceDiscovery{
			storage:    storage,
			serviceKey: cfg.ServiceKey,
			ttl:        int64(cfg.TTL),
			watchers:   make(map[string][]chan *ServiceUpdate),
			ctx:        ctx,
			cancel:     cancel,
		}

		// 启动服务发现
		go instance.watch()
	})

	return instance
}

// Register 注册服务
// 修改Register方法
func (sd *ServiceDiscovery) Register(service *ServiceInfo) error {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()

	// 添加服务过期时间
	service.ExpireTime = time.Now().Add(time.Duration(sd.ttl) * time.Second)
	return sd.storage.Register(service)
}

// Deregister 注销服务
func (sd *ServiceDiscovery) Deregister(service *ServiceInfo) error {
	return sd.storage.Deregister(service.ID)
}

// GetService 获取服务信息
func (sd *ServiceDiscovery) GetService(name, id string) (*ServiceInfo, error) {
	return sd.storage.GetService(name, id)
}

// GetServices 获取所有服务
func (sd *ServiceDiscovery) GetServices(name string) ([]*ServiceInfo, error) {
	return sd.storage.GetServices(name)
}

// watch 监听服务变更
func (sd *ServiceDiscovery) watch() {
	// 实现监听逻辑
}
