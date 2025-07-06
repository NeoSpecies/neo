/*
 * 描述: 定义系统性能指标收集相关类型和Prometheus指标实现，包括HTTP/TCP协议指标、服务调用指标和连接池统计
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// 初始化函数：注册所有Prometheus指标
func init() {
	prometheus.MustRegister(
		httpRequestDuration,
		httpRequestTotal,
		httpRequestErrors,
		tcpRequestDuration,
		tcpRequestTotal,
		tcpRequestErrors,
		messageSizeHistogram,
		healthGauge,
		requestCounter,
		errorCounter,
		latencyHistogram,
		connectionGauge,
	)
}

// 协议指标定义 - 使用小写字母开头表示包内私有
// 包含HTTP和TCP协议的请求统计、错误计数和耗时分布指标
var (
	// HTTP协议指标
	// +---------------------------+-----------------------------------+
	// | 指标名                    | 描述                              |
	// +---------------------------+-----------------------------------+
	// | http_request_duration     | HTTP请求处理耗时分布              |
	// | http_request_total        | HTTP请求总数                      |
	// | http_request_errors_total | HTTP请求错误总数                  |
	// +---------------------------+-----------------------------------+
	// HTTP请求处理耗时分布（秒）
	// 包含默认分桶: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
	httpRequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "http_request_duration_seconds",
			Help:      "HTTP请求处理耗时分布",
			Buckets:   prometheus.DefBuckets,
		},
	)

	// HTTP请求总数计数器
	// 累计所有HTTP请求的总数量
	httpRequestTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "http_request_total",
			Help:      "HTTP请求总数",
		},
	)

	// HTTP请求错误总数计数器
	// 累计所有HTTP请求发生的错误数量
	httpRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "http_request_errors_total",
			Help:      "HTTP请求错误总数",
		},
	)

	// TCP协议指标
	// +---------------------------+-----------------------------------+
	// | 指标名                    | 描述                              |
	// +---------------------------+-----------------------------------+
	// | tcp_request_duration     | TCP请求处理耗时分布              |
	// | tcp_request_total        | TCP请求总数                      |
	// | tcp_request_errors_total | TCP请求错误总数                  |
	// +---------------------------+-----------------------------------+
	// TCP请求处理耗时分布（秒）
	// 包含默认分桶: 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
	tcpRequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "tcp_request_duration_seconds",
			Help:      "TCP请求处理耗时分布",
			Buckets:   prometheus.DefBuckets,
		},
	)

	// TCP请求总数计数器
	// 累计所有TCP请求的总数量
	tcpRequestTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "tcp_request_total",
			Help:      "TCP请求总数",
		},
	)

	// TCP请求错误总数计数器
	// 累计所有TCP请求发生的错误数量
	tcpRequestErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "tcp_request_errors_total",
			Help:      "TCP请求错误总数",
		},
	)

	// 服务调用指标
	// +---------------------------+-----------------------------------+
	// | 指标名                    | 描述                              |
	// +---------------------------+-----------------------------------+
	// | requests_total            | 请求总数（按服务、方法、状态）    |
	// | errors_total              | 错误总数（按服务、方法、错误类型）|
	// | latency_seconds           | 请求延迟分布（按服务、方法）      |
	// | connections               | 连接池统计（按服务、状态）        |
	// | message_size_bytes        | 消息大小分布（按服务、方法、类型）|
	// | health_score              | 服务健康分数（0-100）             |
	// +---------------------------+
	// 请求总数计数器向量
	// 标签: service(服务名), method(方法名), status(状态: success/error)
	requestCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "requests_total",
			Help:      "Total number of requests",
		},
		[]string{"service", "method", "status"},
	)

	// 错误总数计数器向量
	// 标签: service(服务名), method(方法名), error_type(错误类型)
	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "neo_ipc",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
		[]string{"service", "method", "error_type"},
	)

	// 请求延迟直方图向量（秒）
	// 标签: service(服务名), method(方法名)
	// 分桶: 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1秒
	latencyHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "latency_seconds",
			Help:      "Request latency distribution",
			Buckets:   []float64{.001, .002, .005, .01, .02, .05, .1, .2, .5, 1},
		},
		[]string{"service", "method"},
	)

	// 连接池统计 gauge 向量
	// 标签: service(服务名), state(状态: active/idle/total)
	connectionGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "connections",
			Help:      "Connection pool statistics",
		},
		[]string{"service", "state"},
	)

	// 消息大小直方图向量（字节）
	// 标签: service(服务名), method(方法名), type(类型)
	// 分桶: 64, 256, 1024, 4096, 16384, 65536, 262144字节
	messageSizeHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "neo_ipc",
			Name:      "message_size_bytes",
			Help:      "Message size distribution",
			Buckets:   []float64{64, 256, 1024, 4096, 16384, 65536, 262144},
		},
		[]string{"service", "method", "type"},
	)

	// 服务健康分数 gauge 向量
	// 标签: service(服务名), instance(实例)
	// 值范围: 0-100，100表示健康，0表示不可用
	healthGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "neo_ipc",
			Name:      "health_score",
			Help:      "Service health score (0-100)",
		},
		[]string{"service", "instance"},
	)
)

// PrometheusCollector Prometheus指标收集器
// 实现types.MetricsCollector接口，提供基于Prometheus的指标收集功能
// +---------------------------+-----------------------------------+
// | 方法名                    | 描述                              |
// +---------------------------+-----------------------------------+
// | UpdateConnections         | 更新连接池统计指标                |
// | CollectRequest            | 记录请求开始时间                  |
// | CollectResponse           | 记录请求响应时间和状态            |
// +---------------------------+-----------------------------------+
type PrometheusCollector struct{}

// UpdateConnections 更新连接池统计指标
// 更新指定服务的活跃连接数、空闲连接数和总连接数
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | serviceName    | string           | 服务名称                          |
// | active         | int              | 活跃连接数                        |
// | idle           | int              | 空闲连接数                        |
// | total          | int              | 总连接数                          |
// +----------------+------------------+-----------------------------------+
func (p *PrometheusCollector) UpdateConnections(serviceName string, active, idle, total int) {
	connectionGauge.WithLabelValues(serviceName, "active").Set(float64(active))
	connectionGauge.WithLabelValues(serviceName, "idle").Set(float64(idle))
	connectionGauge.WithLabelValues(serviceName, "total").Set(float64(total))
}

// CollectRequest 记录请求开始时间
// 为指定服务和方法记录请求开始时间并返回当前时间戳
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | ctx            | context.Context  | 请求上下文                        |
// | serviceName    | string           | 服务名称                          |
// | method         | string           | 方法名称                          |
// | 返回值         | time.Time        | 请求开始时间戳                    |
// +----------------+------------------+-----------------------------------+
func (p *PrometheusCollector) CollectRequest(ctx context.Context, serviceName, method string) time.Time {
	return time.Now()
}

// CollectResponse 记录请求响应时间和状态
// 计算请求耗时并更新延迟直方图、请求计数器和错误计数器
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | ctx            | context.Context  | 请求上下文                        |
// | serviceName    | string           | 服务名称                          |
// | method         | string           | 方法名称                          |
// | startTime      | time.Time        | 请求开始时间（CollectRequest返回值）|
// | err            | error            | 请求错误信息（nil表示成功）       |
// +----------------+------------------+-----------------------------------+
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
