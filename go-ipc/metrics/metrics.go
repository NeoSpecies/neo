package metrics

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// 默认指标
	defaultMetrics = newMetrics()
	// 全局注册表
	registry = prometheus.NewRegistry()
)

// 性能指标定义
var (
	// QPS指标
	qpsGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ipc",
			Name:      "qps",
			Help:      "Queries per second",
		},
		[]string{"service", "method"},
	)

	// 延迟分布
	latencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ipc",
			Name:      "latency_seconds",
			Help:      "Request latency distribution",
			Buckets:   []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5, 1},
		},
		[]string{"service", "method"},
	)

	// 错误计数
	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ipc",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"service", "method", "error_type"},
	)

	// 连接池指标
	connectionGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ipc",
			Name:      "connections",
			Help:      "Connection pool statistics",
		},
		[]string{"service", "state"}, // state: active, idle, total
	)

	// 请求计数
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "ipc",
			Name:      "requests_total",
			Help:      "Total number of requests",
		},
		[]string{"service", "method", "status"},
	)

	// 消息大小
	messageSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "ipc",
			Name:      "message_size_bytes",
			Help:      "Message size distribution",
			Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144},
		},
		[]string{"service", "method", "type"}, // type: request, response
	)

	// 服务健康度
	healthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "ipc",
			Name:      "health_score",
			Help:      "Service health score (0-100)",
		},
		[]string{"service", "instance"},
	)
)

// Metrics 监控指标管理
type Metrics struct {
	mu       sync.RWMutex
	registry *prometheus.Registry
	server   *http.Server
}

// newMetrics 创建监控指标管理器
func newMetrics() *Metrics {
	reg := prometheus.NewRegistry()

	// 注册指标
	reg.MustRegister(qpsGauge)
	reg.MustRegister(latencyHistogram)
	reg.MustRegister(errorCounter)
	reg.MustRegister(connectionGauge)
	reg.MustRegister(requestCounter)
	reg.MustRegister(messageSizeHistogram)
	reg.MustRegister(healthGauge)

	return &Metrics{
		registry: reg,
	}
}

// StartServer 启动监控服务器
func (m *Metrics) StartServer(addr string) error {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))

	m.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return m.server.ListenAndServe()
}

// StopServer 停止监控服务器
func (m *Metrics) StopServer(ctx context.Context) error {
	if m.server != nil {
		return m.server.Shutdown(ctx)
	}
	return nil
}

// RecordLatency 记录请求延迟
func RecordLatency(service, method string, duration time.Duration) {
	latencyHistogram.WithLabelValues(service, method).Observe(duration.Seconds())
}

// RecordError 记录错误
func RecordError(service, method, errorType string) {
	errorCounter.WithLabelValues(service, method, errorType).Inc()
}

// RecordRequest 记录请求
func RecordRequest(service, method, status string) {
	requestCounter.WithLabelValues(service, method, status).Inc()
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

// MetricsCollector 指标收集器
type MetricsCollector struct {
	service    string
	method     string
	startTime  time.Time
	messageSize int64
}

// NewMetricsCollector 创建指标收集器
func NewMetricsCollector(service, method string) *MetricsCollector {
	return &MetricsCollector{
		service:   service,
		method:    method,
		startTime: time.Now(),
	}
}

// SetMessageSize 设置消息大小
func (c *MetricsCollector) SetMessageSize(size int64) {
	c.messageSize = size
}

// Done 完成请求记录
func (c *MetricsCollector) Done(err error) {
	duration := time.Since(c.startTime)
	RecordLatency(c.service, c.method, duration)

	if c.messageSize > 0 {
		RecordMessageSize(c.service, c.method, "request", c.messageSize)
	}

	if err != nil {
		RecordError(c.service, c.method, err.Error())
		RecordRequest(c.service, c.method, "error")
	} else {
		RecordRequest(c.service, c.method, "success")
	}
}

// QPSCollector QPS收集器
type QPSCollector struct {
	mu         sync.RWMutex
	service    string
	method     string
	count      int64
	lastUpdate time.Time
}

// NewQPSCollector 创建QPS收集器
func NewQPSCollector(service, method string) *QPSCollector {
	collector := &QPSCollector{
		service:    service,
		method:     method,
		lastUpdate: time.Now(),
	}

	// 启动QPS计算协程
	go collector.calculate()

	return collector
}

// Increment 增加请求计数
func (c *QPSCollector) Increment() {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()
}

// calculate 计算QPS
func (c *QPSCollector) calculate() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		duration := now.Sub(c.lastUpdate).Seconds()
		qps := float64(c.count) / duration
		c.count = 0
		c.lastUpdate = now
		c.mu.Unlock()

		UpdateQPS(c.service, c.method, qps)
	}
}

// HealthCalculator 健康度计算器
type HealthCalculator struct {
	mu             sync.RWMutex
	service        string
	instance       string
	errorCount     int64
	totalRequests  int64
	avgLatency     float64
	lastCalculated time.Time
}

// NewHealthCalculator 创建健康度计算器
func NewHealthCalculator(service, instance string) *HealthCalculator {
	calculator := &HealthCalculator{
		service:        service,
		instance:       instance,
		lastCalculated: time.Now(),
	}

	// 启动健康度计算协程
	go calculator.calculate()

	return calculator
}

// RecordRequest 记录请求
func (c *HealthCalculator) RecordRequest(duration time.Duration, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.totalRequests++
	if err != nil {
		c.errorCount++
	}

	// 计算移动平均延迟
	if c.avgLatency == 0 {
		c.avgLatency = duration.Seconds()
	} else {
		c.avgLatency = (c.avgLatency*0.9 + duration.Seconds()*0.1)
	}
}

// calculate 计算健康度
func (c *HealthCalculator) calculate() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		var score float64 = 100

		// 错误率影响
		if c.totalRequests > 0 {
			errorRate := float64(c.errorCount) / float64(c.totalRequests)
			if errorRate > 0.1 { // 错误率超过10%
				score -= errorRate * 100
			}
		}

		// 延迟影响
		if c.avgLatency > 0.1 { // 平均延迟超过100ms
			score -= (c.avgLatency - 0.1) * 100
		}

		// 重置计数
		c.errorCount = 0
		c.totalRequests = 0
		c.mu.Unlock()

		// 确保分数在0-100范围内
		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}

		UpdateHealth(c.service, c.instance, score)
	}
} 