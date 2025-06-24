package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"neo/internal/config"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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
	Default *Metrics
)

// Metrics 监控指标管理器
type Metrics struct {
	mu       sync.RWMutex
	registry *prometheus.Registry
	server   *http.Server
}

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

	// 初始化默认实例
	Default = NewMetrics()
}

// NewMetrics 创建监控指标管理器
func NewMetrics() *Metrics {
	return &Metrics{
		registry: registry,
	}
}

// StartServer 启动监控服务器
func (m *Metrics) StartServer() error {
	cfg := config.Get().Metrics
	if !cfg.EnablePrometheus {
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	addr := fmt.Sprintf(":%d", cfg.PrometheusPort)
	m.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		if err := m.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("监控服务器错误: %v", err)
		}
	}()

	log.Printf("Prometheus监控服务器已启动，地址: %s", addr)
	return nil
}

// StopServer 停止监控服务器
func (m *Metrics) StopServer(ctx context.Context) error {
	if m.server != nil {
		return m.server.Shutdown(ctx)
	}
	return nil
}

// RecordConnectionRefused 记录连接拒绝事件
func (m *Metrics) RecordConnectionRefused() {
	connectionGauge.WithLabelValues("tcp", "refused").Inc()
}

// RecordError 记录错误
func (m *Metrics) RecordError(service, method, errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	errorCounter.WithLabelValues(service, method, errorType).Inc()
}

// RecordRequest 记录请求
func (m *Metrics) RecordRequest(service, method, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	requestCounter.WithLabelValues(service, method, status).Inc()
}

// RecordLatency 记录请求延迟
func (m *Metrics) RecordLatency(service, method string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	latencyHistogram.WithLabelValues(service, method).Observe(duration.Seconds())
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

// UpdateHealth 更新服务健康度
func UpdateHealth(service, instance string, score float64) {
	healthGauge.WithLabelValues(service, instance).Set(score)
}
