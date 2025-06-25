package metrics

import (
	"context"
	"log"
	"neo/internal/config"
	"neo/internal/types"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// 全局指标注册表
	registry = prometheus.NewRegistry()

	// QPS指标
	qpsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "qps",
			Help:      "Queries per second",
		},
		[]string{"service", "method"},
	)

	// 延迟分布
	latencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "latency_seconds",
			Help:      "Request latency distribution",
			Buckets:   []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5, 1},
		},
		[]string{"service", "method"},
	)

	// 错误计数
	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"service", "method", "error_type"},
	)

	// 连接池指标
	connectionGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "connections",
			Help:      "Connection pool statistics",
		},
		[]string{"service", "state"},
	)

	// 请求计数
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "requests_total",
			Help:      "Total number of requests",
		},
		[]string{"service", "method", "status"},
	)

	// 消息大小
	messageSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "message_size_bytes",
			Help:      "Message size distribution",
			Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144},
		},
		[]string{"service", "method", "type"},
	)

	// 服务健康度
	healthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "health_score",
			Help:      "Service health score (0-100)",
		},
		[]string{"service", "instance"},
	)

	// Default 全局默认指标实例
)

// 初始化指标
func init() {
	// 注册所有指标
	registry.MustRegister(qpsGauge)
	registry.MustRegister(latencyHistogram)
	registry.MustRegister(errorCounter)
	registry.MustRegister(connectionGauge)
	registry.MustRegister(requestCounter)
	registry.MustRegister(messageSizeHistogram)
	registry.MustRegister(healthGauge)

	// 修复：传入全局registry参数
	Default = NewMetrics(registry)
}

// StartServer 启动监控服务器
// StartServer 启动监控服务器
func StartServer(m *types.Metrics) error {
    // 使用全局配置
    cfg := config.GetGlobalConfig()

    // 检查指标是否启用 - 修复：使用正确的配置路径
    if !cfg.Metrics.Enabled {
        return nil
    }

    mux := http.NewServeMux()
    mux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))

    // 使用配置中的完整地址
    addr := cfg.Metrics.PrometheusAddress
    m.Server = &http.Server{
        Addr:    addr,
        Handler: mux,
    }

    go func() {
        if err := m.Server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Printf("监控服务器错误: %v", err)
        }
    }()

    log.Printf("Prometheus监控服务器已启动，地址: %s", addr)
    return nil
}

// StopServer 停止监控服务器
func StopServer(m *types.Metrics, ctx context.Context) error {
	if m.Server != nil {
		return m.Server.Shutdown(ctx)
	}
	return nil
}

// RecordConnectionRefused 记录连接拒绝事件 - 修复：改为函数形式
func RecordConnectionRefused(m *types.Metrics) {
	connectionGauge.WithLabelValues("tcp", "refused").Inc()
}

// RecordMessageSize 记录消息大小
func RecordMessageSize(service, method, msgType string, size int64) {
	messageSizeHistogram.WithLabelValues(service, method, msgType).Observe(float64(size))
}

// UpdateQPS 更新QPS
func UpdateQPS(service, method string, qps float64) {
	qpsGauge.WithLabelValues(service, method).Set(qps)
}

// UpdateConnections 更新连接池状态
func UpdateConnections(service string, active, idle, total int) {
	connectionGauge.WithLabelValues(service, "active").Set(float64(active))
	connectionGauge.WithLabelValues(service, "idle").Set(float64(idle))
	connectionGauge.WithLabelValues(service, "total").Set(float64(total))
}

// UpdateHealthScore 更新服务健康度
func UpdateHealthScore(service, instance string, score float64) {
	healthGauge.WithLabelValues(service, instance).Set(score)
}

// Default 全局默认指标实例
var Default *types.Metrics

// 初始化指标
func init() {
	// 注册所有指标
	registry.MustRegister(qpsGauge)
	registry.MustRegister(latencyHistogram)
	registry.MustRegister(errorCounter)
	registry.MustRegister(connectionGauge)
	registry.MustRegister(requestCounter)
	registry.MustRegister(messageSizeHistogram)
	registry.MustRegister(healthGauge)

	// 修复：传入全局registry参数
	Default = NewMetrics(registry)
}

// InitDefaultMetrics 初始化默认指标实例
func InitDefaultMetrics() {
	Default = NewMetrics(prometheus.NewRegistry()) // 确保此处调用带参数的NewMetrics
}

// NewMetrics 创建新的指标实例
func NewMetrics(registry *prometheus.Registry) *types.Metrics {
	m := &types.Metrics{
		Registry: registry,
	}

	// 注册指标
	promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{Name: "requests_total", Help: "Total number of requests"},
		[]string{"service", "method", "status"},
	)

	promauto.With(registry).NewCounterVec(
		prometheus.CounterOpts{Name: "errors_total", Help: "Total number of errors"},
		[]string{"service", "method", "error_type"},
	)

	promauto.With(registry).NewHistogramVec(
		prometheus.HistogramOpts{Name: "request_latency_seconds", Help: "Request latency in seconds"},
		[]string{"service", "method"},
	)

	return m
}

// RecordRequest 记录请求指标
func RecordRequest(m *types.Metrics, serviceName, method, status string) {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	// 使用promauto从注册表获取或创建指标
	metric := promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{Name: "requests_total", Help: "Total number of requests"},
		[]string{"service", "method", "status"},
	)
	metric.WithLabelValues(serviceName, method, status).Inc()
}

// RecordError 记录错误指标
func RecordError(m *types.Metrics, serviceName, method, errorType string) {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	metric := promauto.With(m.Registry).NewCounterVec(
		prometheus.CounterOpts{Name: "errors_total", Help: "Total number of errors"},
		[]string{"service", "method", "error_type"},
	)
	metric.WithLabelValues(serviceName, method, errorType).Inc()
}

// RecordLatency 记录延迟指标
func RecordLatency(m *types.Metrics, serviceName, method string, duration time.Duration) {
	m.Mu.RLock()
	defer m.Mu.RUnlock()

	metric := promauto.With(m.Registry).NewHistogramVec(
		prometheus.HistogramOpts{Name: "request_latency_seconds", Help: "Request latency in seconds"},
		[]string{"service", "method"},
	)
	metric.WithLabelValues(serviceName, method).Observe(duration.Seconds())
}
