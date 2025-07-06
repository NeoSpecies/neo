// neo/internal/types/metrics_extended.go
package types

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// 协议指标定义 - 使用小写字母开头表示包内私有
var (
	// HTTP协议指标
	httpRequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP请求处理耗时分布",
			Buckets:   prometheus.DefBuckets,
		},
	)

	httpRequestTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "http_request_total",
			Help:      "HTTP请求总数",
		},
	)

	httpRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "http_request_errors_total",
			Help:      "HTTP请求错误总数",
		},
	)

	// TCP协议指标
	tcpRequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "tcp_request_duration_seconds",
			Help:      "TCP请求处理耗时分布",
			Buckets:   prometheus.DefBuckets,
		},
	)

	tcpRequestTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "tcp_request_total",
			Help:      "TCP请求总数",
		},
	)

	tcpRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "tcp_request_errors_total",
			Help:      "TCP请求错误总数",
		},
	)

	// 服务调用指标
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "requests_total",
			Help:      "Total number of requests",
		},
		[]string{"service", "method", "status"},
	)

	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"service", "method", "error_type"},
	)

	latencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "latency_seconds",
			Help:      "Request latency distribution",
			Buckets:   []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5, 1},
		},
		[]string{"service", "method"},
	)

	connectionGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "connections",
			Help:      "Connection pool statistics",
		},
		[]string{"service", "state"},
	)

	messageSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "message_size_bytes",
			Help:      "Message size distribution",
			Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144},
		},
		[]string{"service", "method", "type"},
	)

	healthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "health_score",
			Help:      "Service health score (0-100)",
		},
		[]string{"service", "instance"},
	)
)

// PrometheusCollector 实现 types.MetricsCollector 接口
type PrometheusCollector struct{}

// 实现接口方法
func (p *PrometheusCollector) UpdateConnections(serviceName string, active, idle, total int) {
	connectionGauge.WithLabelValues(serviceName, "active").Set(float64(active))
	connectionGauge.WithLabelValues(serviceName, "idle").Set(float64(idle))
	connectionGauge.WithLabelValues(serviceName, "total").Set(float64(total))
}

func (p *PrometheusCollector) CollectRequest(ctx context.Context, serviceName, method string) time.Time {
	return time.Now()
}

func (p *PrometheusCollector) CollectResponse(ctx context.Context, serviceName, method string, startTime time.Time, err error) {
	duration := time.Since(startTime)
	latencyHistogram.WithLabelValues(serviceName, method).Observe(duration.Seconds())

	status := "success"
	if err != nil {
		status = "error"
		errorCounter.WithLabelValues(serviceName, method, err.Error()).Inc()
	}
	requestCounter.WithLabelValues(serviceName, method, status).Inc()
}
