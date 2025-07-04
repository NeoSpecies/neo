package connection

import (
	"neo/internal/types"
)

// 类型别名迁移（保留兼容）
// Deprecated: 请使用 types.LoadBalanceStrategy
type LoadBalanceStrategy = types.LoadBalanceStrategy

// Deprecated: 请使用 types.MetricsCollector
type MetricsCollector = types.MetricsCollector

// Deprecated: 请使用 types.Balancer
type Balancer = types.Balancer

// Deprecated: 请使用 types.RoundRobinBalancer
type RoundRobinBalancer = types.RoundRobinBalancer

// Deprecated: 请使用 types.WeightedBalancer
type WeightedBalancer = types.WeightedBalancer

func NewRoundRobinBalancer(serviceName, methodName string, collector MetricsCollector) *RoundRobinBalancer {
	return types.NewRoundRobinBalancer(serviceName, methodName, collector)
}

// NewBalancer 根据策略创建负载均衡器
func NewBalancer(strategy types.LoadBalanceStrategy, serviceName, methodName string, collector types.MetricsCollector) types.Balancer {
	switch strategy {
	case types.LoadBalanceRoundRobin:
		return types.NewRoundRobinBalancer(serviceName, methodName, collector)
	case types.LoadBalanceWeighted:
		return types.NewWeightedBalancer() // 使用types包的构造函数
	default:
		return types.NewRoundRobinBalancer(serviceName, methodName, collector)
	}
}

// NewWeightedBalancer 创建新的加权负载均衡器
func NewWeightedBalancer() *WeightedBalancer {
	return &WeightedBalancer{}
}
