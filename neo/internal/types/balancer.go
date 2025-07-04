package types

import (
	"context"
	"errors"
	"sync"
	"time"
)

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	LoadBalanceRoundRobin LoadBalanceStrategy = iota // 轮询策略
	LoadBalanceWeighted                              // 加权策略
)

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	CollectRequest(ctx context.Context, serviceName, method string) time.Time
	CollectResponse(ctx context.Context, serviceName, method string, startTime time.Time, err error)
	UpdateConnections(serviceName string, current, added, removed int)
}

// Balancer 负载均衡器接口
type Balancer interface {
	Pick(availableConns []interface{}) (interface{}, error)
	Release(conn interface{}, err error)
	Add(conn interface{})
	Remove(conn interface{})
	Len() int
	Close()
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	connections      []interface{}
	index            int
	mu               sync.Mutex
	serviceName      string
	methodName       string
	metricsCollector MetricsCollector
}

// WeightedBalancer 加权负载均衡器
type WeightedBalancer struct {
	connections []interface{}
	weights     []int
	index       int
	mu          sync.Mutex
}

// NewWeightedBalancer 创建新的加权负载均衡器
func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{
		connections: make([]interface{}, 0),
		weights:     make([]int, 0),
	}
}

// Pick 加权选择一个连接
func (w *WeightedBalancer) Pick(availableConns []interface{}) (interface{}, error) {
	// 实现加权负载均衡算法
	w.mu.Lock()
	defer w.mu.Unlock()

	// 简单实现：返回第一个连接
	if len(availableConns) > 0 {
		return availableConns[0], nil
	}
	return nil, errors.New("没有可用连接")
}

// Release 释放连接
func (w *WeightedBalancer) Release(conn interface{}, err error) {
	// 实现释放逻辑
}

// Add 添加连接到负载均衡器
func (w *WeightedBalancer) Add(conn interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 默认权重为1
	w.connections = append(w.connections, conn)
	w.weights = append(w.weights, 1)
}

// Remove 从负载均衡器移除连接
func (w *WeightedBalancer) Remove(conn interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 实现移除逻辑
}

// Len 获取连接数量
func (w *WeightedBalancer) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.connections)
}

// Close 关闭负载均衡器
func (w *WeightedBalancer) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connections = nil
	w.weights = nil
}
