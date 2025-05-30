package pool

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LoadBalanceStrategy 负载均衡策略类型
type LoadBalanceStrategy string

const (
	// RoundRobin 轮询策略
	RoundRobin LoadBalanceStrategy = "round_robin"
	// LeastConnections 最少连接策略
	LeastConnections LoadBalanceStrategy = "least_connections"
	// Random 随机策略
	Random LoadBalanceStrategy = "random"
	// WeightedRoundRobin 加权轮询策略
	WeightedRoundRobin LoadBalanceStrategy = "weighted_round_robin"
)

// Balancer 负载均衡器接口
type Balancer interface {
	// Select 选择一个连接
	Select(conns []*Connection) *Connection
	// UpdateStats 更新连接统计信息
	UpdateStats(conn *Connection, latency time.Duration, err error)
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	counter uint64
}

func (b *RoundRobinBalancer) Select(conns []*Connection) *Connection {
	if len(conns) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&b.counter, 1) % uint64(len(conns))
	return conns[idx]
}

func (b *RoundRobinBalancer) UpdateStats(conn *Connection, latency time.Duration, err error) {
	// 轮询策略不需要统计信息
}

// LeastConnectionsBalancer 最少连接负载均衡器
type LeastConnectionsBalancer struct {
	mu sync.RWMutex
}

func (b *LeastConnectionsBalancer) Select(conns []*Connection) *Connection {
	if len(conns) == 0 {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	var selected *Connection
	minActive := int64(^uint64(0) >> 1)

	for _, conn := range conns {
		if conn.Stats.ActiveRequests < minActive {
			selected = conn
			minActive = conn.Stats.ActiveRequests
		}
	}

	return selected
}

func (b *LeastConnectionsBalancer) UpdateStats(conn *Connection, latency time.Duration, err error) {
	if err != nil {
		atomic.AddInt64(&conn.Stats.ErrorCount, 1)
	}
	conn.Stats.LatencyStats.Add(latency)
}

// RandomBalancer 随机负载均衡器
type RandomBalancer struct {
	rand *rand.Rand
	mu   sync.Mutex
}

func NewRandomBalancer() *RandomBalancer {
	return &RandomBalancer{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (b *RandomBalancer) Select(conns []*Connection) *Connection {
	if len(conns) == 0 {
		return nil
	}

	b.mu.Lock()
	idx := b.rand.Intn(len(conns))
	b.mu.Unlock()

	return conns[idx]
}

func (b *RandomBalancer) UpdateStats(conn *Connection, latency time.Duration, err error) {
	// 随机策略不需要统计信息
}

// WeightedRoundRobinBalancer 加权轮询负载均衡器
type WeightedRoundRobinBalancer struct {
	mu            sync.RWMutex
	weights       map[*Connection]int
	currentWeight int
	maxWeight     int
}

func NewWeightedRoundRobinBalancer() *WeightedRoundRobinBalancer {
	return &WeightedRoundRobinBalancer{
		weights: make(map[*Connection]int),
	}
}

func (b *WeightedRoundRobinBalancer) Select(conns []*Connection) *Connection {
	if len(conns) == 0 {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// 初始化或更新权重
	b.updateWeights(conns)

	// 选择权重最高的连接
	var selected *Connection
	maxWeight := 0

	for _, conn := range conns {
		weight := b.weights[conn]
		if weight > maxWeight {
			selected = conn
			maxWeight = weight
		}
	}

	// 更新当前权重
	if selected != nil {
		b.weights[selected] -= b.maxWeight
	}

	return selected
}

func (b *WeightedRoundRobinBalancer) updateWeights(conns []*Connection) {
	// 根据连接的性能统计更新权重
	for _, conn := range conns {
		if _, exists := b.weights[conn]; !exists {
			weight := b.calculateWeight(conn)
			b.weights[conn] = weight
			if weight > b.maxWeight {
				b.maxWeight = weight
			}
		}
	}

	// 清理已断开的连接
	for conn := range b.weights {
		found := false
		for _, activeConn := range conns {
			if conn == activeConn {
				found = true
				break
			}
		}
		if !found {
			delete(b.weights, conn)
		}
	}
}

func (b *WeightedRoundRobinBalancer) calculateWeight(conn *Connection) int {
	// 基础权重
	weight := 100

	// 根据错误率调整权重
	if conn.Stats.TotalRequests > 0 {
		errorRate := float64(conn.Stats.ErrorCount) / float64(conn.Stats.TotalRequests)
		if errorRate > 0.1 { // 错误率超过10%
			weight -= int(errorRate * 100)
		}
	}

	// 根据延迟调整权重
	avgLatency := conn.Stats.LatencyStats.Average()
	if avgLatency > time.Millisecond*100 { // 平均延迟超过100ms
		weight -= int(avgLatency.Milliseconds() / 100)
	}

	// 确保权重在合理范围内
	if weight < 1 {
		weight = 1
	}
	if weight > 100 {
		weight = 100
	}

	return weight
}

func (b *WeightedRoundRobinBalancer) UpdateStats(conn *Connection, latency time.Duration, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err != nil {
		atomic.AddInt64(&conn.Stats.ErrorCount, 1)
	}
	atomic.AddInt64(&conn.Stats.TotalRequests, 1)
	conn.Stats.LatencyStats.Add(latency)

	// 更新连接权重
	b.weights[conn] = b.calculateWeight(conn)
}

// NewBalancer 创建负载均衡器
func NewBalancer(strategy LoadBalanceStrategy) Balancer {
	switch strategy {
	case RoundRobin:
		return &RoundRobinBalancer{}
	case LeastConnections:
		return &LeastConnectionsBalancer{}
	case Random:
		return NewRandomBalancer()
	case WeightedRoundRobin:
		return NewWeightedRoundRobinBalancer()
	default:
		return &RoundRobinBalancer{} // 默认使用轮询策略
	}
} 