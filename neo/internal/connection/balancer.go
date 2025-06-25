package connection

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"neo/internal/metrics"
	"neo/internal/types"
)

// LoadBalanceStrategy 负载均衡策略
type LoadBalanceStrategy int

const (
	RoundRobin LoadBalanceStrategy = iota // 轮询策略
	Weighted                              // 加权策略
)

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	CollectRequest(ctx context.Context, serviceName, method string) time.Time
	CollectResponse(ctx context.Context, serviceName, method string, startTime time.Time, err error)
}

// Balancer 负载均衡器接口
type Balancer interface {
	// Pick 选择一个连接
	Pick(availableConns []interface{}) (interface{}, error)
	// Release 释放连接
	Release(conn interface{}, err error)
	// Add 添加连接
	Add(conn interface{})
	// Remove 移除连接
	Remove(conn interface{})
	// Len 获取连接数量
	Len() int
	// Close 关闭负载均衡器
	Close()
}

// NewBalancer 根据策略创建负载均衡器
func NewBalancer(strategy LoadBalanceStrategy, serviceName, methodName string, collector MetricsCollector) Balancer {
	switch strategy {
	case RoundRobin:
		return NewRoundRobinBalancer(serviceName, methodName, collector)
	case Weighted:
		return NewWeightedBalancer()
	default:
		return NewRoundRobinBalancer(serviceName, methodName, collector) // 默认轮询策略
	}
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

// NewRoundRobinBalancer 创建新的轮询负载均衡器
func NewRoundRobinBalancer(serviceName, methodName string, collector MetricsCollector) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		connections:      make([]interface{}, 0),
		index:            0,
		serviceName:      serviceName,
		methodName:       methodName,
		metricsCollector: collector,
	}
}

// Pick 轮询选择一个连接
func (r *RoundRobinBalancer) Pick(availableConns []interface{}) (interface{}, error) {
	if availableConns == nil || len(availableConns) == 0 {
		r.mu.Lock()
		defer r.mu.Unlock()
		availableConns = r.connections
	}

	if len(availableConns) == 0 {
		err := errors.New("没有可用连接")
		if r.metricsCollector != nil {
			ctx := context.Background()
			startTime := time.Now()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, err)
		}
		return nil, err
	}

	// 轮询选择连接
	r.mu.Lock()
	conn := availableConns[r.index]
	r.index = (r.index + 1) % len(availableConns)
	r.mu.Unlock()

	return conn, nil
}

// Release 释放连接
func (r *RoundRobinBalancer) Release(conn interface{}, err error) {
	startTime := time.Now()

	if err != nil {
		if r.metricsCollector != nil {
			ctx := context.Background()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, err)
		}
		r.Remove(conn)
	} else {
		if r.metricsCollector != nil {
			ctx := context.Background()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, nil)
		}
	}
}

// Add 添加连接到负载均衡器
func (r *RoundRobinBalancer) Add(conn interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.connections {
		if c == conn {
			return
		}
	}

	r.connections = append(r.connections, conn)
	log.Printf("添加连接到负载均衡器，当前连接数: %d", len(r.connections))
	metrics.UpdateConnections(r.serviceName, len(r.connections), 0, len(r.connections))
}

// Remove 从负载均衡器移除连接
func (r *RoundRobinBalancer) Remove(conn interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, c := range r.connections {
		if c == conn {
			r.connections = append(r.connections[:i], r.connections[i+1:]...)
			log.Printf("从负载均衡器移除连接，当前连接数: %d", len(r.connections))

			if r.index >= len(r.connections) && len(r.connections) > 0 {
				r.index = 0
			}

			metrics.UpdateConnections(r.serviceName, len(r.connections), 0, len(r.connections))
			return
		}
	}
}

// Len 获取连接数量
func (r *RoundRobinBalancer) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.connections)
}

// Close 关闭负载均衡器
func (r *RoundRobinBalancer) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.connections = nil
	r.index = 0
	log.Println("负载均衡器已关闭")
}

// WeightedBalancer 加权负载均衡器
type WeightedBalancer struct {
	// 实现加权负载均衡逻辑
}

// NewWeightedBalancer 创建新的加权负载均衡器
func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{}
}

// Pick 选择一个连接（加权策略）
func (w *WeightedBalancer) Pick(availableConns []interface{}) (interface{}, error) {
	// 实现加权选择逻辑
	return nil, nil
}

// Release 释放连接（加权策略）
func (w *WeightedBalancer) Release(conn interface{}, err error) {
	// 实现释放逻辑
}

// Add 添加连接（加权策略）
func (w *WeightedBalancer) Add(conn interface{}) {
	// 实现添加逻辑
}

// Remove 移除连接（加权策略）
func (w *WeightedBalancer) Remove(conn interface{}) {
	// 实现移除逻辑
}

// Len 获取连接数量（加权策略）
func (w *WeightedBalancer) Len() int {
	return 0
}

// Close 关闭负载均衡器（加权策略）
func (w *WeightedBalancer) Close() {
	// 实现关闭逻辑
}

// Balancer 负载均衡器接口
// Select 实现types.Balancer接口
func (r *RoundRobinBalancer) Select(connections []*types.Connection) (*types.Connection, error) {
	if len(connections) == 0 {
		err := errors.New("没有可用连接")
		if r.metricsCollector != nil {
			// 使用metricsCollector代替metrics.Default
			ctx := context.Background()
			startTime := time.Now()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, err)
		}
		return nil, err
	}

	// 轮询选择连接
	r.mu.Lock()
	conn := connections[r.index]
	r.index = (r.index + 1) % len(connections)
	r.mu.Unlock()

	return conn, nil
}
