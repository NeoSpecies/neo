package core

import (
	"context"
	"fmt"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/types"
	"neo/internal/utils"
	"sync"
	"time"
)

// Service 服务接口
type Service interface {
	Name() string
	HandleRequest(ctx context.Context, req types.Request) (types.Response, error)
	Middleware() []Middleware
	Close() error
}

// Middleware 中间件接口
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc 处理函数
type HandlerFunc func(ctx context.Context, req types.Request) (types.Response, error)

// ServiceOptions 服务选项
type ServiceOptions struct {
	Name        string
	Transport   transport.Transport
	Registry    registry.ServiceRegistry
	Middlewares []Middleware
	Timeout     time.Duration
	Logger      utils.Logger
	AsyncIPC    AsyncIPCClient // 添加AsyncIPC接口
}

// AsyncIPCClient IPC客户端接口
type AsyncIPCClient interface {
	ForwardRequest(ctx context.Context, serviceName string, method string, data []byte) ([]byte, error)
}

// serviceImpl 服务实现
type serviceImpl struct {
	name        string
	transport   transport.Transport
	registry    registry.ServiceRegistry
	middlewares []Middleware
	timeout     time.Duration
	logger      utils.Logger
	asyncIPC    AsyncIPCClient
	handler     HandlerFunc
	mu          sync.RWMutex
	closed      bool
}

// NewService 创建新服务实例
func NewService(opts ServiceOptions) Service {
	if opts.Name == "" {
		opts.Name = "default-service"
	}
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.Logger == nil {
		opts.Logger = utils.DefaultLogger
	}

	service := &serviceImpl{
		name:        opts.Name,
		transport:   opts.Transport,
		registry:    opts.Registry,
		middlewares: opts.Middlewares,
		timeout:     opts.Timeout,
		logger:      opts.Logger,
		asyncIPC:    opts.AsyncIPC,
	}

	// 构建中间件链
	service.buildMiddlewareChain()

	return service
}

// Name 返回服务名称
func (s *serviceImpl) Name() string {
	return s.name
}

// HandleRequest 处理请求
func (s *serviceImpl) HandleRequest(ctx context.Context, req types.Request) (types.Response, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return types.Response{}, fmt.Errorf("service is closed")
	}
	handler := s.handler
	s.mu.RUnlock()

	// 如果没有设置处理器，使用默认处理逻辑
	if handler == nil {
		handler = s.defaultHandler
	}

	// 应用超时
	if s.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	// 记录请求开始
	startTime := time.Now()
	s.logger.Info("handling request",
		utils.String("service", s.name),
		utils.String("requestID", req.ID),
		utils.String("targetService", req.Service),
		utils.String("method", req.Method))

	// 执行处理器
	resp, err := handler(ctx, req)

	// 记录处理结果
	duration := time.Since(startTime)
	if err != nil {
		s.logger.Error("request handling failed",
			utils.String("service", s.name),
			utils.String("requestID", req.ID),
			utils.String("error", err.Error()),
			utils.String("duration", duration.String()))
	} else {
		s.logger.Info("request handled successfully",
			utils.String("service", s.name),
			utils.String("requestID", req.ID),
			utils.Int("status", resp.Status),
			utils.String("duration", duration.String()))
	}

	return resp, err
}

// Middleware 返回中间件列表
func (s *serviceImpl) Middleware() []Middleware {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回副本避免外部修改
	result := make([]Middleware, len(s.middlewares))
	copy(result, s.middlewares)
	return result
}

// Close 关闭服务
func (s *serviceImpl) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	s.logger.Info("service closed", utils.String("service", s.name))
	return nil
}

// SetHandler 设置自定义处理器
func (s *serviceImpl) SetHandler(handler HandlerFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handler = handler
}

// serviceImpl 类型需要导出才能在测试中使用
type ServiceImpl = serviceImpl

// buildMiddlewareChain 构建中间件链
func (s *serviceImpl) buildMiddlewareChain() {
	if len(s.middlewares) == 0 {
		return
	}

	// 从最后一个中间件开始构建链
	handler := s.defaultHandler
	for i := len(s.middlewares) - 1; i >= 0; i-- {
		handler = s.middlewares[i](handler)
	}
	s.handler = handler
}

// defaultHandler 默认处理器，实现服务发现和请求转发
func (s *serviceImpl) defaultHandler(ctx context.Context, req types.Request) (types.Response, error) {
	// 调试：打印接收到的请求
	fmt.Printf("defaultHandler: Received request for service '%s', method '%s'\n", req.Service, req.Method)
	
	// 1. 通过注册中心发现目标服务
	if s.registry == nil {
		fmt.Printf("defaultHandler: Registry not configured\n")
		return types.Response{
			ID:     req.ID,
			Status: 500,
			Error:  "registry not configured",
		}, nil
	}

	fmt.Printf("defaultHandler: Discovering service '%s'\n", req.Service)
	
	// 调试：列出所有已注册的服务
	allServices, err := s.registry.ListServices(ctx)
	if err != nil {
		fmt.Printf("defaultHandler: Failed to list services: %v\n", err)
	} else {
		fmt.Printf("defaultHandler: All registered services (%d): %v\n", len(allServices), allServices)
	}
	
	instances, err := s.registry.Discover(ctx, req.Service)
	if err != nil {
		fmt.Printf("defaultHandler: Service discovery failed: %v\n", err)
		return types.Response{
			ID:     req.ID,
			Status: 500,
			Error:  fmt.Sprintf("service discovery failed: %v", err),
		}, nil
	}

	fmt.Printf("defaultHandler: Found %d instances for service '%s'\n", len(instances), req.Service)
	if len(instances) == 0 {
		fmt.Printf("defaultHandler: No service instances found for '%s'\n", req.Service)
		return types.Response{
			ID:     req.ID,
			Status: 404,
			Error:  "no service instances found",
		}, nil
	}

	// 2. 选择服务实例（简单使用第一个）
	_ = instances[0] // TODO: 实现负载均衡算法选择实例
	
	// 3. 通过AsyncIPC发送请求
	fmt.Printf("defaultHandler: AsyncIPC status: %v\n", s.asyncIPC != nil)
	if s.asyncIPC == nil {
		fmt.Printf("defaultHandler: AsyncIPC not configured\n")
		return types.Response{
			ID:     req.ID,
			Status: 500,
			Error:  "AsyncIPC not configured",
		}, nil
	}

	// 直接使用AsyncIPC转发请求
	fmt.Printf("defaultHandler: Calling AsyncIPC.ForwardRequest for service '%s', method '%s'\n", req.Service, req.Method)
	respData, err := s.asyncIPC.ForwardRequest(ctx, req.Service, req.Method, req.Body)
	if err != nil {
		return types.Response{
			ID:     req.ID,
			Status: 503,
			Error:  fmt.Sprintf("IPC call failed: %v", err),
		}, nil
	}

	return types.Response{
		ID:     req.ID,
		Status: 200,
		Body:   respData,
	}, nil
}

// 预定义的中间件

// LoggingMiddleware 日志中间件
func LoggingMiddleware(logger utils.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req types.Request) (types.Response, error) {
			start := time.Now()
			
			logger.Info("request started",
				utils.String("requestID", req.ID),
				utils.String("service", req.Service),
				utils.String("method", req.Method))

			resp, err := next(ctx, req)
			
			duration := time.Since(start)
			if err != nil {
				logger.Error("request failed",
					utils.String("requestID", req.ID),
					utils.String("error", err.Error()),
					utils.String("duration", duration.String()))
			} else {
				logger.Info("request completed",
					utils.String("requestID", req.ID),
					utils.Int("status", resp.Status),
					utils.String("duration", duration.String()))
			}

			return resp, err
		}
	}
}

// TimeoutMiddleware 超时中间件
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req types.Request) (types.Response, error) {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			done := make(chan struct{})
			var resp types.Response
			var err error

			go func() {
				resp, err = next(ctx, req)
				close(done)
			}()

			select {
			case <-done:
				return resp, err
			case <-ctx.Done():
				return types.Response{
					ID:     req.ID,
					Status: 408,
					Error:  "request timeout",
				}, nil
			}
		}
	}
}

// RecoveryMiddleware 恢复中间件，捕获panic
func RecoveryMiddleware(logger utils.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req types.Request) (resp types.Response, err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic recovered",
						utils.String("requestID", req.ID),
						utils.String("panic", fmt.Sprintf("%v", r)))
					
					resp = types.Response{
						ID:     req.ID,
						Status: 500,
						Error:  "internal server error",
					}
					err = nil
				}
			}()

			return next(ctx, req)
		}
	}
}

// MetricsMiddleware 指标中间件
func MetricsMiddleware(metrics *ServiceMetrics) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req types.Request) (types.Response, error) {
			start := time.Now()
			metrics.IncRequestCount()

			resp, err := next(ctx, req)

			duration := time.Since(start)
			metrics.UpdateLatency(duration)
			
			if err != nil || resp.Status >= 400 {
				metrics.IncErrorCount()
			} else {
				metrics.IncSuccessCount()
			}

			return resp, err
		}
	}
}

// ServiceMetrics 服务指标
type ServiceMetrics struct {
	mu            sync.RWMutex
	requestCount  int64
	successCount  int64
	errorCount    int64
	totalLatency  time.Duration
	requestLatest time.Time
}

// IncRequestCount 增加请求计数
func (m *ServiceMetrics) IncRequestCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCount++
	m.requestLatest = time.Now()
}

// IncSuccessCount 增加成功计数
func (m *ServiceMetrics) IncSuccessCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.successCount++
}

// IncErrorCount 增加错误计数
func (m *ServiceMetrics) IncErrorCount() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount++
}

// UpdateLatency 更新延迟
func (m *ServiceMetrics) UpdateLatency(latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalLatency += latency
}

// GetStats 获取统计信息
func (m *ServiceMetrics) GetStats() (requestCount, successCount, errorCount int64, avgLatency time.Duration, lastRequest time.Time) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	requestCount = m.requestCount
	successCount = m.successCount
	errorCount = m.errorCount
	lastRequest = m.requestLatest
	
	if m.requestCount > 0 {
		avgLatency = m.totalLatency / time.Duration(m.requestCount)
	}
	
	return
}