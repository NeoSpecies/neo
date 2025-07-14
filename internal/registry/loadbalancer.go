package registry

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// Select 选择一个服务实例
	Select(instances []*ServiceInstance) (*ServiceInstance, error)
	// Name 获取负载均衡器名称
	Name() string
}

// RandomLoadBalancer 随机负载均衡器
type RandomLoadBalancer struct {
	rand *rand.Rand
	mu   sync.Mutex
}

// NewRandomLoadBalancer 创建随机负载均衡器
func NewRandomLoadBalancer() LoadBalancer {
	return &RandomLoadBalancer{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select 随机选择一个实例
func (r *RandomLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	return instances[r.rand.Intn(len(instances))], nil
}

// Name 获取名称
func (r *RandomLoadBalancer) Name() string {
	return "random"
}

// RoundRobinLoadBalancer 轮询负载均衡器
type RoundRobinLoadBalancer struct {
	counter uint64
}

// NewRoundRobinLoadBalancer 创建轮询负载均衡器
func NewRoundRobinLoadBalancer() LoadBalancer {
	return &RoundRobinLoadBalancer{}
}

// Select 轮询选择实例
func (r *RoundRobinLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}
	
	count := atomic.AddUint64(&r.counter, 1)
	return instances[(count-1)%uint64(len(instances))], nil
}

// Name 获取名称
func (r *RoundRobinLoadBalancer) Name() string {
	return "round-robin"
}

// WeightedRoundRobinLoadBalancer 加权轮询负载均衡器
type WeightedRoundRobinLoadBalancer struct {
	mu              sync.Mutex
	currentWeights  map[string]int
	effectiveWeights map[string]int
}

// NewWeightedRoundRobinLoadBalancer 创建加权轮询负载均衡器
func NewWeightedRoundRobinLoadBalancer() LoadBalancer {
	return &WeightedRoundRobinLoadBalancer{
		currentWeights:   make(map[string]int),
		effectiveWeights: make(map[string]int),
	}
}

// Select 基于权重选择实例
func (w *WeightedRoundRobinLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}
	
	w.mu.Lock()
	defer w.mu.Unlock()
	
	// 单实例直接返回
	if len(instances) == 1 {
		return instances[0], nil
	}
	
	totalWeight := 0
	var selected *ServiceInstance
	
	// 计算总权重并选择实例
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = 1
		}
		
		// 初始化权重
		if _, exists := w.effectiveWeights[instance.ID]; !exists {
			w.effectiveWeights[instance.ID] = weight
		}
		if _, exists := w.currentWeights[instance.ID]; !exists {
			w.currentWeights[instance.ID] = 0
		}
		
		// 增加当前权重
		w.currentWeights[instance.ID] += w.effectiveWeights[instance.ID]
		totalWeight += w.effectiveWeights[instance.ID]
		
		// 选择当前权重最大的实例
		if selected == nil || w.currentWeights[instance.ID] > w.currentWeights[selected.ID] {
			selected = instance
		}
	}
	
	// 减少选中实例的当前权重
	if selected != nil {
		w.currentWeights[selected.ID] -= totalWeight
	}
	
	// 清理不存在的实例权重
	w.cleanupWeights(instances)
	
	return selected, nil
}

// cleanupWeights 清理不存在的实例权重
func (w *WeightedRoundRobinLoadBalancer) cleanupWeights(instances []*ServiceInstance) {
	instanceMap := make(map[string]bool)
	for _, instance := range instances {
		instanceMap[instance.ID] = true
	}
	
	for id := range w.currentWeights {
		if !instanceMap[id] {
			delete(w.currentWeights, id)
			delete(w.effectiveWeights, id)
		}
	}
}

// Name 获取名称
func (w *WeightedRoundRobinLoadBalancer) Name() string {
	return "weighted-round-robin"
}

// LeastConnectionLoadBalancer 最少连接数负载均衡器
type LeastConnectionLoadBalancer struct {
	mu          sync.RWMutex
	connections map[string]int64
}

// NewLeastConnectionLoadBalancer 创建最少连接数负载均衡器
func NewLeastConnectionLoadBalancer() LoadBalancer {
	return &LeastConnectionLoadBalancer{
		connections: make(map[string]int64),
	}
}

// Select 选择连接数最少的实例
func (l *LeastConnectionLoadBalancer) Select(instances []*ServiceInstance) (*ServiceInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("no available instances")
	}
	
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	var selected *ServiceInstance
	minConnections := int64(^uint64(0) >> 1) // MaxInt64
	
	for _, instance := range instances {
		connections := l.connections[instance.ID]
		if connections < minConnections {
			minConnections = connections
			selected = instance
		}
	}
	
	if selected == nil && len(instances) > 0 {
		selected = instances[0]
	}
	
	return selected, nil
}

// Name 获取名称
func (l *LeastConnectionLoadBalancer) Name() string {
	return "least-connection"
}

// AddConnection 增加连接数
func (l *LeastConnectionLoadBalancer) AddConnection(instanceID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.connections[instanceID]++
}

// RemoveConnection 减少连接数
func (l *LeastConnectionLoadBalancer) RemoveConnection(instanceID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.connections[instanceID] > 0 {
		l.connections[instanceID]--
	}
}

// LoadBalancerFactory 负载均衡器工厂
func NewLoadBalancer(algorithm string) (LoadBalancer, error) {
	switch algorithm {
	case "random":
		return NewRandomLoadBalancer(), nil
	case "round-robin":
		return NewRoundRobinLoadBalancer(), nil
	case "weighted-round-robin":
		return NewWeightedRoundRobinLoadBalancer(), nil
	case "least-connection":
		return NewLeastConnectionLoadBalancer(), nil
	default:
		return nil, fmt.Errorf("unknown load balancer algorithm: %s", algorithm)
	}
}