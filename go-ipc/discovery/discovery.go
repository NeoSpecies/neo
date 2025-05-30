package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

const (
	// DefaultTTL 服务注册的默认TTL
	DefaultTTL = 10 // seconds

	// DefaultRefreshInterval 服务刷新间隔
	DefaultRefreshInterval = 3 * time.Second
)

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name      string            `json:"name"`       // 服务名称
	ID        string            `json:"id"`         // 服务实例ID
	Address   string            `json:"address"`    // 服务地址
	Port      int              `json:"port"`       // 服务端口
	Version   string            `json:"version"`    // 服务版本
	Metadata  map[string]string `json:"metadata"`   // 服务元数据
	Status    string            `json:"status"`     // 服务状态
	StartTime time.Time         `json:"start_time"` // 启动时间
}

// ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	client     *clientv3.Client  // etcd客户端
	serviceKey string            // 服务键前缀
	ttl        int64            // 租约TTL
	leaseID    clientv3.LeaseID // 租约ID
	mutex      sync.RWMutex
	services   map[string]*ServiceInfo // 本地服务缓存
	watchers   map[string][]chan *ServiceUpdate // 服务变更通知
	ctx        context.Context
	cancel     context.CancelFunc
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

// NewServiceDiscovery 创建服务发现实例
func NewServiceDiscovery(endpoints []string, serviceKey string) (*ServiceDiscovery, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	sd := &ServiceDiscovery{
		client:     client,
		serviceKey: serviceKey,
		ttl:        DefaultTTL,
		services:   make(map[string]*ServiceInfo),
		watchers:   make(map[string][]chan *ServiceUpdate),
		ctx:        ctx,
		cancel:     cancel,
	}

	// 启动服务发现
	go sd.watch()

	return sd, nil
}

// Register 注册服务
func (sd *ServiceDiscovery) Register(service *ServiceInfo) error {
	// 创建租约
	lease, err := sd.client.Grant(sd.ctx, sd.ttl)
	if err != nil {
		return err
	}

	// 服务数据序列化
	data, err := json.Marshal(service)
	if err != nil {
		return err
	}

	// 注册服务
	key := fmt.Sprintf("%s/%s/%s", sd.serviceKey, service.Name, service.ID)
	_, err = sd.client.Put(sd.ctx, key, string(data), clientv3.WithLease(lease.ID))
	if err != nil {
		return err
	}

	sd.leaseID = lease.ID

	// 自动续约
	keepAliveCh, err := sd.client.KeepAlive(sd.ctx, lease.ID)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case <-sd.ctx.Done():
				return
			case resp := <-keepAliveCh:
				if resp == nil {
					// 续约失败，重新注册
					if err := sd.Register(service); err != nil {
						log.Printf("Service re-register failed: %v", err)
					}
					return
				}
			}
		}
	}()

	return nil
}

// Deregister 注销服务
func (sd *ServiceDiscovery) Deregister(service *ServiceInfo) error {
	key := fmt.Sprintf("%s/%s/%s", sd.serviceKey, service.Name, service.ID)
	_, err := sd.client.Delete(sd.ctx, key)
	if err != nil {
		return err
	}

	// 撤销租约
	if sd.leaseID != 0 {
		_, err = sd.client.Revoke(sd.ctx, sd.leaseID)
	}

	return err
}

// GetService 获取服务信息
func (sd *ServiceDiscovery) GetService(name, id string) (*ServiceInfo, error) {
	key := fmt.Sprintf("%s/%s/%s", sd.serviceKey, name, id)
	resp, err := sd.client.Get(sd.ctx, key)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, fmt.Errorf("service not found: %s", key)
	}

	var service ServiceInfo
	if err := json.Unmarshal(resp.Kvs[0].Value, &service); err != nil {
		return nil, err
	}

	return &service, nil
}

// GetServices 获取所有服务
func (sd *ServiceDiscovery) GetServices(name string) ([]*ServiceInfo, error) {
	prefix := fmt.Sprintf("%s/%s/", sd.serviceKey, name)
	resp, err := sd.client.Get(sd.ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	services := make([]*ServiceInfo, 0, len(resp.Kvs))
	for _, kv := range resp.Kvs {
		var service ServiceInfo
		if err := json.Unmarshal(kv.Value, &service); err != nil {
			continue
		}
		services = append(services, &service)
	}

	return services, nil
}

// Watch 监听服务变更
func (sd *ServiceDiscovery) Watch(name string) (<-chan *ServiceUpdate, error) {
	sd.mutex.Lock()
	defer sd.mutex.Unlock()

	ch := make(chan *ServiceUpdate, 10)
	if _, ok := sd.watchers[name]; !ok {
		sd.watchers[name] = make([]chan *ServiceUpdate, 0)
	}
	sd.watchers[name] = append(sd.watchers[name], ch)

	return ch, nil
}

// watch 监听服务变更并通知
func (sd *ServiceDiscovery) watch() {
	prefix := sd.serviceKey + "/"
	rch := sd.client.Watch(sd.ctx, prefix, clientv3.WithPrefix())

	for {
		select {
		case <-sd.ctx.Done():
			return
		case wresp := <-rch:
			for _, ev := range wresp.Events {
				var (
					service ServiceInfo
					typ     UpdateType
				)

				switch ev.Type {
				case clientv3.EventTypePut:
					if err := json.Unmarshal(ev.Kv.Value, &service); err != nil {
						continue
					}
					if ev.IsCreate() {
						typ = ServiceAdded
					} else {
						typ = ServiceModified
					}
				case clientv3.EventTypeDelete:
					// 解析服务名称
					key := string(ev.Kv.Key)
					parts := strings.Split(key, "/")
					if len(parts) < 3 {
						continue
					}
					service.Name = parts[len(parts)-2]
					service.ID = parts[len(parts)-1]
					typ = ServiceRemoved
				}

				// 通知所有监听者
				sd.mutex.RLock()
				if watchers, ok := sd.watchers[service.Name]; ok {
					for _, ch := range watchers {
						select {
						case ch <- &ServiceUpdate{Type: typ, Service: &service}:
						default:
							// 通道已满，跳过
						}
					}
				}
				sd.mutex.RUnlock()
			}
		}
	}
}

// Close 关闭服务发现
func (sd *ServiceDiscovery) Close() error {
	sd.cancel()
	
	// 关闭所有监听通道
	sd.mutex.Lock()
	for _, watchers := range sd.watchers {
		for _, ch := range watchers {
			close(ch)
		}
	}
	sd.watchers = nil
	sd.mutex.Unlock()

	return sd.client.Close()
} 