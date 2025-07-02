package metrics

import (
	"context"
	"log"

	"neo/internal/types"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// 全局变量声明
var (
	globalRegistry = prometheus.NewRegistry()
	once           sync.Once
	defaultMetrics *types.Metrics
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

// 初始化所有指标和默认实例
// 初始化默认指标收集器
func init() {
	// 注册所有协议指标
	metrics := []prometheus.Collector{
		httpRequestDuration,
		httpRequestTotal,
		httpRequestErrors,
		tcpRequestDuration,
		tcpRequestTotal,
		tcpRequestErrors,
		requestCounter,
		errorCounter,
		latencyHistogram,
		connectionGauge,
		messageSizeHistogram,
		healthGauge,
	}

	// 修复：使用MustRegister代替Register，并正确处理多个Collector
	globalRegistry.MustRegister(metrics...)

	// 初始化默认指标实例
	once.Do(func() {
		defaultMetrics = types.NewMetrics(globalRegistry)
	})
	// 添加指标收集器初始化
	defaultCollector := &PrometheusCollector{}
	defaultMetrics.Collector = defaultCollector
}

// GetDefaultMetrics 获取默认指标实例
func GetDefaultMetrics() *types.Metrics {
	return defaultMetrics
}

// StartServer 启动监控服务器
// 修改StartServer函数，通过参数注入配置
func StartServer(cfg *types.MetricsConfig) error {
	if !cfg.Enabled {
		log.Println("监控功能已禁用")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(globalRegistry, promhttp.HandlerOpts{}))

	addr := cfg.PrometheusAddress
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// 将服务器实例保存到默认指标中
	defaultMetrics.Server = server

	go func() {
		log.Printf("Prometheus监控服务器已启动，地址: %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("监控服务器错误: %v", err)
		}
	}()

	return nil
}

// StopServer 停止监控服务器
func StopServer(ctx context.Context) error {
	if defaultMetrics.Server != nil {
		return defaultMetrics.Server.Shutdown(ctx)
	}
	return nil
}

// HTTP指标收集函数
func RecordHTTPRequest(duration float64, success bool) {
	httpRequestDuration.Observe(duration)
	httpRequestTotal.Inc()
	if !success {
		httpRequestErrors.Inc()
	}
}

// TCP指标收集函数
func RecordTCPRequest(duration float64, success bool) {
	tcpRequestDuration.Observe(duration)
	tcpRequestTotal.Inc()
	if !success {
		tcpRequestErrors.Inc()
	}
}

// RecordConnectionRefused 记录连接拒绝事件
func RecordConnectionRefused() {
	connectionGauge.WithLabelValues("tcp", "refused").Inc()
}

// RecordMessageSize 记录消息大小
func RecordMessageSize(service, method, msgType string, size int64) {
	messageSizeHistogram.WithLabelValues(service, method, msgType).Observe(float64(size))
}

// UpdateConnections 更新连接池状态
// PrometheusCollector 实现 types.MetricsCollector 接口
type PrometheusCollector struct{}

// UpdateConnections 实现接口方法
func (p *PrometheusCollector) UpdateConnections(serviceName string, active, idle, total int) {
	connectionGauge.WithLabelValues(serviceName, "active").Set(float64(active))
	connectionGauge.WithLabelValues(serviceName, "idle").Set(float64(idle))
	connectionGauge.WithLabelValues(serviceName, "total").Set(float64(total))
}

// CollectRequest 实现接口方法
func (p *PrometheusCollector) CollectRequest(ctx context.Context, serviceName, method string) time.Time {
	return time.Now()
}

// CollectResponse 实现接口方法
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

// UpdateHealthScore 更新服务健康度
func UpdateHealthScore(service, instance string, score float64) {
	healthGauge.WithLabelValues(service, instance).Set(score)
}

// RecordRequest 记录请求指标
func RecordRequest(serviceName, method, status string) {
	requestCounter.WithLabelValues(serviceName, method, status).Inc()
}

// RecordError 记录错误指标
func RecordError(serviceName, method, errorType string) {
	errorCounter.WithLabelValues(serviceName, method, errorType).Inc()
}

// RecordLatency 记录延迟指标
func RecordLatency(serviceName, method string, duration time.Duration) {
	latencyHistogram.WithLabelValues(serviceName, method).Observe(duration.Seconds())
}
