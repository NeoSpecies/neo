/*
 * 描述: 定义负载均衡相关的类型和接口，包括轮询和加权负载均衡策略的实现
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"context"
	"errors"
	"sync"
	"time"
)

// LoadBalanceStrategy 负载均衡策略枚举
// 定义系统支持的负载均衡算法类型
// +---------------------+-----------------------------------+
// | 常量名              | 描述                              |
// +---------------------+-----------------------------------+
// | LoadBalanceRoundRobin | 轮询策略：按顺序依次选择连接        |
// | LoadBalanceWeighted  | 加权策略：根据权重值选择连接        |
// +---------------------+-----------------------------------+
type LoadBalanceStrategy int

const (
	LoadBalanceRoundRobin LoadBalanceStrategy = iota // 轮询策略
	LoadBalanceWeighted                              // 加权策略
)

// MetricsCollector 指标收集器接口
// 用于收集负载均衡过程中的性能指标和连接状态
// +--------------------------------+------------------------------------------------+
// | 方法名                         | 描述                                           |
// +--------------------------------+------------------------------------------------+
// | CollectRequest                 | 记录请求开始时间，返回当前时间戳                |
// | CollectResponse                | 记录请求响应时间，计算耗时并处理错误信息        |
// | UpdateConnections              | 更新连接状态统计，包括当前连接数、新增和移除数量 |
// +--------------------------------+------------------------------------------------+
type MetricsCollector interface {
	CollectRequest(ctx context.Context, serviceName, method string) time.Time
	CollectResponse(ctx context.Context, serviceName, method string, startTime time.Time, err error)
	UpdateConnections(serviceName string, current, added, removed int)
}

// Balancer 负载均衡器接口
// 定义负载均衡器的基本操作规范
// +----------------+----------------------------------------+
// | 方法名         | 描述                                   |
// +----------------+----------------------------------------+
// | Pick           | 从可用连接中选择一个连接               |
// | Release        | 释放连接并记录使用情况                 |
// | Add            | 添加新连接到负载均衡器                 |
// | Remove         | 从负载均衡器中移除指定连接             |
// | Len            | 获取当前连接总数                       |
// | Close          | 关闭负载均衡器并释放所有资源           |
// +----------------+----------------------------------------+
type Balancer interface {
	Pick(availableConns []interface{}) (interface{}, error)
	Release(conn interface{}, err error)
	Add(conn interface{})
	Remove(conn interface{})
	Len() int
	Close()
}

// RoundRobinBalancer 轮询负载均衡器
// 实现按顺序循环选择连接的负载均衡策略
// +-------------------+-----------------------------------+
// | 字段名            | 描述                              |
// +-------------------+-----------------------------------+
// | connections       | 存储所有可用连接的切片            |
// | index             | 当前选择的连接索引                |
// | mu                | 保证并发安全的互斥锁              |
// | serviceName       | 服务名称                          |
// | methodName        | 方法名称                          |
// | metricsCollector  | 指标收集器实例                    |
// +-------------------+-----------------------------------+
type RoundRobinBalancer struct {
	connections      []interface{}
	index            int
	mu               sync.Mutex
	serviceName      string
	methodName       string
	metricsCollector MetricsCollector
}

// Pick 轮询选择一个连接
// 从可用连接中按顺序选择下一个连接
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | availableConns | 可用连接列表                      |
// +----------------+-----------------------------------+
// | 返回值         | 选中的连接和可能的错误            |
// +----------------+-----------------------------------+
func (r *RoundRobinBalancer) Pick(availableConns []interface{}) (interface{}, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(availableConns) == 0 {
		return nil, errors.New("no available connections")
	}

	// 使用当前索引并递增
	conn := availableConns[r.index]
	r.index = (r.index + 1) % len(availableConns)
	return conn, nil
}

// WeightedBalancer 加权负载均衡器
// 实现基于权重的连接选择策略
// +-------------------+-----------------------------------+
// | 字段名            | 描述                              |
// +-------------------+-----------------------------------+
// | connections       | 存储所有可用连接的切片            |
// | weights           | 存储对应连接的权重值              |
// | index             | 当前选择的连接索引                |
// | mu                | 保证并发安全的互斥锁              |
// +-------------------+-----------------------------------+
type WeightedBalancer struct {
	connections []interface{}
	weights     []int
	index       int
	mu          sync.Mutex
}

// NewWeightedBalancer 创建新的加权负载均衡器
// 返回一个初始化后的WeightedBalancer实例
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | *WeightedBalancer | 初始化后的加权负载均衡器实例      |
// +----------------+-----------------------------------+
func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{
		connections: make([]interface{}, 0),
		weights:     make([]int, 0),
	}
}

// Pick 加权选择一个连接
// 根据权重算法从可用连接中选择最合适的连接
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | availableConns | 可用连接列表                      |
// | 返回值         | 选中的连接和可能的错误            |
// +----------------+-----------------------------------+
func (w *WeightedBalancer) Pick(availableConns []interface{}) (interface{}, error) {
	// 实现加权负载均衡算法
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(availableConns) == 0 {
		return nil, errors.New("没有可用连接")
	}

	// 简单加权轮询实现
	totalWeight := 0
	for _, weight := range w.weights {
		totalWeight += weight
	}

	// 使用index选择下一个连接
	w.index = (w.index + 1) % totalWeight
	currentWeight := w.index

	for i, conn := range availableConns {
		currentWeight -= w.weights[i]
		if currentWeight < 0 {
			return conn, nil
		}
	}

	//  fallback to first connection
	return availableConns[0], nil
}

// Release 释放连接
// 释放使用完毕的连接，可在此添加连接状态重置逻辑
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | conn           | 要释放的连接                      |
// | err            | 连接使用过程中产生的错误          |
// +----------------+-----------------------------------+
func (w *WeightedBalancer) Release(conn interface{}, err error) {
	// 实现释放逻辑
}

// Add 添加连接到负载均衡器
// 将新连接添加到负载均衡器，并设置默认权重为1
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | conn           | 要添加的连接                      |
// +----------------+-----------------------------------+
func (w *WeightedBalancer) Add(conn interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 默认权重为1
	w.connections = append(w.connections, conn)
	w.weights = append(w.weights, 1)
}

// Remove 从负载均衡器移除连接
// 从负载均衡器中移除指定连接
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | conn           | 要移除的连接                      |
// +----------------+-----------------------------------+
func (w *WeightedBalancer) Remove(conn interface{}) {
	w.mu.Lock()
	defer w.mu.Unlock()
	// 实现移除逻辑
}

// Len 获取连接数量
// 返回当前负载均衡器中的连接总数
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | int            | 连接总数                          |
// +----------------+-----------------------------------+
func (w *WeightedBalancer) Len() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.connections)
}

// Close 关闭负载均衡器
// 清空所有连接和权重数据，释放资源
func (w *WeightedBalancer) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.connections = nil
	w.weights = nil
}
