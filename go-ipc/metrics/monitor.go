package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

// Monitor 监控管理器
type Monitor struct {
	metrics   *Metrics
	qps       map[string]*QPSCollector
	health    map[string]*HealthCalculator
	tracer    trace.Tracer
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// MonitorConfig 监控配置
type MonitorConfig struct {
	ServiceName    string
	MetricsAddr   string
	JaegerURL     string
	EnableTracing bool
}

// NewMonitor 创建监控管理器
func NewMonitor(config MonitorConfig) (*Monitor, error) {
	ctx, cancel := context.WithCancel(context.Background())

	m := &Monitor{
		metrics: newMetrics(),
		qps:     make(map[string]*QPSCollector),
		health:  make(map[string]*HealthCalculator),
		ctx:     ctx,
		cancel:  cancel,
	}

	// 启动指标服务器
	if config.MetricsAddr != "" {
		go func() {
			if err := m.metrics.StartServer(config.MetricsAddr); err != nil && err != http.ErrServerClosed {
				log.Printf("Metrics server error: %v", err)
			}
		}()
	}

	// 初始化追踪
	if config.EnableTracing && config.JaegerURL != "" {
		if err := m.initTracing(config.ServiceName, config.JaegerURL); err != nil {
			return nil, fmt.Errorf("failed to initialize tracing: %v", err)
		}
	}

	return m, nil
}

// initTracing 初始化分布式追踪
func (m *Monitor) initTracing(serviceName, jaegerURL string) error {
	// 创建Jaeger导出器
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(jaegerURL)))
	if err != nil {
		return err
	}

	// 创建资源信息
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return err
	}

	// 创建追踪器提供者
	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	m.tracer = tp.Tracer(serviceName)
	return nil
}

// StartRequest 开始请求追踪
func (m *Monitor) StartRequest(ctx context.Context, service, method string) (context.Context, *RequestMonitor) {
	var span trace.Span
	if m.tracer != nil {
		ctx, span = m.tracer.Start(ctx, fmt.Sprintf("%s.%s", service, method))
	}

	return ctx, &RequestMonitor{
		monitor:  m,
		service:  service,
		method:   method,
		metrics:  NewMetricsCollector(service, method),
		span:     span,
		startTime: time.Now(),
	}
}

// GetQPSCollector 获取QPS收集器
func (m *Monitor) GetQPSCollector(service, method string) *QPSCollector {
	key := fmt.Sprintf("%s.%s", service, method)
	
	m.mu.Lock()
	defer m.mu.Unlock()

	if collector, ok := m.qps[key]; ok {
		return collector
	}

	collector := NewQPSCollector(service, method)
	m.qps[key] = collector
	return collector
}

// GetHealthCalculator 获取健康度计算器
func (m *Monitor) GetHealthCalculator(service, instance string) *HealthCalculator {
	key := fmt.Sprintf("%s.%s", service, instance)
	
	m.mu.Lock()
	defer m.mu.Unlock()

	if calculator, ok := m.health[key]; ok {
		return calculator
	}

	calculator := NewHealthCalculator(service, instance)
	m.health[key] = calculator
	return calculator
}

// Close 关闭监控
func (m *Monitor) Close() error {
	m.cancel()
	return m.metrics.StopServer(context.Background())
}

// RequestMonitor 请求监控器
type RequestMonitor struct {
	monitor   *Monitor
	service   string
	method    string
	metrics   *MetricsCollector
	span      trace.Span
	startTime time.Time
}

// AddAttribute 添加追踪属性
func (r *RequestMonitor) AddAttribute(key string, value interface{}) {
	if r.span != nil {
		switch v := value.(type) {
		case string:
			r.span.SetAttributes(attribute.String(key, v))
		case int64:
			r.span.SetAttributes(attribute.Int64(key, v))
		case float64:
			r.span.SetAttributes(attribute.Float64(key, v))
		case bool:
			r.span.SetAttributes(attribute.Bool(key, v))
		}
	}
}

// SetMessageSize 设置消息大小
func (r *RequestMonitor) SetMessageSize(size int64) {
	r.metrics.SetMessageSize(size)
	if r.span != nil {
		r.span.SetAttributes(attribute.Int64("message.size", size))
	}
}

// End 结束请求监控
func (r *RequestMonitor) End(err error) {
	duration := time.Since(r.startTime)

	// 记录指标
	r.metrics.Done(err)

	// 更新QPS
	r.monitor.GetQPSCollector(r.service, r.method).Increment()

	// 更新健康度
	r.monitor.GetHealthCalculator(r.service, "").RecordRequest(duration, err)

	// 结束追踪
	if r.span != nil {
		if err != nil {
			r.span.RecordError(err)
		}
		r.span.End()
	}
}

// RecordEvent 记录事件
func (r *RequestMonitor) RecordEvent(name string, attributes map[string]interface{}) {
	if r.span != nil {
		attrs := make([]attribute.KeyValue, 0, len(attributes))
		for k, v := range attributes {
			switch val := v.(type) {
			case string:
				attrs = append(attrs, attribute.String(k, val))
			case int64:
				attrs = append(attrs, attribute.Int64(k, val))
			case float64:
				attrs = append(attrs, attribute.Float64(k, val))
			case bool:
				attrs = append(attrs, attribute.Bool(k, val))
			}
		}
		r.span.AddEvent(name, trace.WithAttributes(attrs...))
	}
} 