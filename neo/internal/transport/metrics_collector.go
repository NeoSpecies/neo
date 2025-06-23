package transport

import (
	"neo/internal/metrics"
	"time"
)

// MetricsCollector 专注于监控指标收集
type MetricsCollector struct {
	// 移除metrics字段，使用全局metrics包函数
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// RecordRequest 记录请求指标
// 修复：使用全局metrics包的RecordRequest函数
func (c *MetricsCollector) RecordRequest(method string, status string) {
	metrics.RecordRequest("ipc", method, status)
}

// RecordLatency 记录延迟指标
// 修复：使用全局metrics包的RecordLatency函数
func (c *MetricsCollector) RecordLatency(method string, duration time.Duration) {
	metrics.RecordLatency("ipc", method, duration)
}
