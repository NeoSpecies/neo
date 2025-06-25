package transport

import (
	"context"
	"time"

	"neo/internal/metrics"
)

// MetricsCollector 处理服务调用的指标收集
type MetricsCollector struct {
	// 可以添加配置字段，如指标前缀、采样率等
}

// NewMetricsCollector 创建新的指标收集器实例
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// CollectRequest 记录请求开始时的指标
func (m *MetricsCollector) CollectRequest(ctx context.Context, serviceName, method string) time.Time {
	// 记录请求总数，状态为"started"
	metrics.RecordRequest(serviceName, method, "started")
	return time.Now()
}

// CollectResponse 记录请求完成时的指标
func (m *MetricsCollector) CollectResponse(ctx context.Context, serviceName, method string, startTime time.Time, err error) {
	// 计算请求持续时间
	duration := time.Since(startTime)
	status := "success"
	errorType := ""

	// 如果有错误，记录错误指标
	if err != nil {
		status = "error"
		errorType = err.Error()
		metrics.RecordError(serviceName, method, errorType)
	}

	// 更新请求状态为完成
	metrics.RecordRequest(serviceName, method, status)
	// 记录延迟指标
	metrics.RecordLatency(serviceName, method, duration)
}
