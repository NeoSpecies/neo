package types

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// 连接池错误定义
var (
	// ErrPoolClosed 连接池已关闭
	ErrPoolClosed = errors.New("connection pool is closed")
	// ErrPoolExhausted 连接池耗尽
	ErrPoolExhausted = errors.New("connection pool exhausted")
	// ErrConnectionUnhealthy 连接不健康
	ErrConnectionUnhealthy = errors.New("connection is unhealthy")
)

// Config 统一连接池配置
type Config struct {
	// 基础配置
	MaxSize           int           // 最大连接数
	MinSize           int           // 最小连接数
	ConnectTimeout    time.Duration // 连接超时时间
	IdleTimeout       time.Duration // 空闲超时时间
	KeepAliveInterval time.Duration // 保持连接间隔

	// 扩展配置
	InitialSize         int                 // 初始连接数
	AutoScaling         bool                // 是否启用自动扩缩容
	ScaleUpThreshold    float64             // 扩容阈值（活跃连接比例）
	ScaleDownThreshold  float64             // 缩容阈值（空闲连接比例）
	ScaleStep           int                 // 每次扩缩容步长
	HealthCheck         bool                // 是否启用健康检查
	HealthCheckInterval time.Duration       // 健康检查间隔
	MaxErrorCount       int                 // 最大错误次数
	MaxLatency          time.Duration       // 最大延迟阈值
	LoadBalance         LoadBalanceStrategy // 负载均衡策略（修改此行）
	MaxRetryCount       int                 // 连接创建最大重试次数
	RetryInterval       time.Duration       // 连接重试间隔
}

// ScalingConfig 自动扩缩容配置
type ScalingConfig struct {
	MinSize   int // 最小连接数
	MaxSize   int // 最大连接数
	ExpandPct int // 扩容阈值百分比
	ShrinkPct int // 缩容阈值百分比
}

// Connection 统一连接结构体
type Connection struct {
	Conn       net.Conn           // 底层连接
	Pool       *TCPConnectionPool // 所属连接池
	Stats      *ConnectionStats   // 连接统计
	LastUsed   time.Time          // 最后使用时间
	LastCheck  time.Time          // 最后检查时间
	InUse      bool               // 是否正在使用
	ErrorCount int                // 错误次数
	Closed     bool               // 是否已关闭
}

// TCPConnectionPool 连接池结构体
type TCPConnectionPool struct {
	MaxSize           int                      `json:"max_size"`
	MinSize           int                      `json:"min_size"`
	InitialSize       int                      `json:"initial_size"`
	IdleTimeout       time.Duration            `json:"idle_timeout"`
	KeepAliveInterval time.Duration            `json:"keep_alive_interval"`
	Config            Config                   `json:"config"` // 修改为Config类型
	Factory           func() (net.Conn, error) `json:"-"`
	Balancer          Balancer                 `json:"balancer"`
	Metrics           *Metrics                 `json:"metrics"` // 确保Metrics引用的是types.Metrics
	Done              chan struct{}            `json:"-"`
	WaitConn          chan struct{}            `json:"-"`
	Connections       []*Connection            `json:"connections"`
	Mu                *sync.RWMutex            `json:"-"` // 修改为RWMutex
	Closed            bool                     `json:"closed"`
}

// ConnectionPoolMetrics 连接池指标
type ConnectionPoolMetrics struct {
	ActiveConnections  int64 // 活跃连接数
	TotalConnections   int64 // 总连接数
	WaitingRequests    int64 // 等待请求数
	ConnectionErrors   int64 // 连接错误数
	ConnectionTimeouts int64 // 连接超时数
}

// ConnectionStats 连接统计信息
// 扩展原有定义，增加BytesRead和BytesWritten字段
type ConnectionStats struct {
	CreatedAt    time.Time // 创建时间
	LastUsed     time.Time // 最后使用时间
	UsageCount   int64     // 使用次数
	ReadBytes    int64     // 读取字节数
	WrittenBytes int64     // 写入字节数
	ErrorCount   int64     // 错误次数
	LastError    error     // 最后错误
	BytesRead    uint64    // 新增：读取字节数
	BytesWritten uint64    // 新增：写入字节数
	LastActive   time.Time // 新增：最后活动时间
}

// LatencyStats 延迟统计
type LatencyStats struct {
	mu         sync.RWMutex
	count      int64
	sum        time.Duration
	min        time.Duration
	max        time.Duration
	buckets    []int64   // 延迟分布桶
	boundaries []float64 // 桶边界（毫秒）
	windowSize int       // 滑动窗口大小
	samples    []float64 // 最近的样本
	currentPos int       // 当前样本位置
}

// MessageCallback 定义了消息回调函数类型，接收字节切片并返回处理后的字节切片和可能的错误
// 具体实现通常在使用该回调的模块中，如处理网络消息的处理器里
type MessageCallback func([]byte) ([]byte, error)

// NewLatencyStats 创建延迟统计
func NewLatencyStats() *LatencyStats {
	return &LatencyStats{
		min:        time.Duration(1<<63 - 1),
		boundaries: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000}, // 毫秒
		buckets:    make([]int64, 10),                                // 9个边界产生10个桶
		windowSize: 1000,                                             // 保存最近1000个样本
		samples:    make([]float64, 1000),
	}
}

// Add 添加一个延迟样本
func (s *LatencyStats) Add(latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.count++
	s.sum += latency

	// 更新最小最大值
	if latency < s.min {
		s.min = latency
	}
	if latency > s.max {
		s.max = latency
	}

	// 记录样本到滑动窗口
	s.samples[s.currentPos] = float64(latency.Milliseconds())
	s.currentPos = (s.currentPos + 1) % s.windowSize

	// 更新延迟分布桶
	ms := float64(latency.Milliseconds())
	for i, boundary := range s.boundaries {
		if ms <= boundary {
			s.buckets[i]++
			return
		}
	}
	s.buckets[len(s.buckets)-1]++ // 超过最大边界的放入最后一个桶
}

// Register 注册回调函数并设置超时清理
func (m *CallbackManager) Register(msgID string, cb Callback, timeout time.Duration) {
	m.callbacks.Lock()
	defer m.callbacks.Unlock()

	m.registry[msgID] = cb

	// 超时自动清理
	time.AfterFunc(timeout, func() {
		m.callbacks.Lock()
		defer m.callbacks.Unlock()
		delete(m.registry, msgID)
	})
}

// HandleResponse 处理响应并触发回调
func (m *CallbackManager) HandleResponse(msgID string, result interface{}, err error) {
	m.callbacks.RLock()
	cb, exists := m.registry[msgID]
	m.callbacks.RUnlock()

	if exists {
		cb(result, err)
		m.callbacks.Lock()
		delete(m.registry, msgID)
		m.callbacks.Unlock()
	}
}

// NewConnectionStats 创建新的连接统计信息
func NewConnectionStats() *ConnectionStats {
	return &ConnectionStats{
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
}

// CallbackManager 管理连接相关事件的回调函数
type CallbackManager struct {
	callbacks sync.RWMutex
	registry  map[string]Callback
}

// NewCallbackManager 创建回调管理器实例
func NewCallbackManager() *CallbackManager {
	return &CallbackManager{
		registry: make(map[string]Callback),
	}
}

// 确保只保留重命名后的结构体及方法
type ConnectionServerConfig struct {
	MaxConnections    int
	MaxMsgSize        int
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	WorkerCount       int
	ConnectionTimeout time.Duration
	Address           string
}

// 确保方法接收者类型正确
func (c *ConnectionServerConfig) GetMaxConnections() int {
	return c.MaxConnections
}

func (c *ConnectionServerConfig) GetConnectionTimeout() time.Duration {
	return c.ConnectionTimeout
}

func (c *ConnectionServerConfig) GetHandlerConfig() interface{} {
	return nil
}

// 修复 line 178: missing return
func (p *BasicConnectionPool) Get() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// 从连接池获取连接
	select {
	case conn := <-p.pool:
		return conn, nil
	default:
		// 池为空，创建新连接
		if p.createFn != nil {
			return p.createFn()
		}
		return nil, ErrPoolExhausted
	}
}

func (p *BasicConnectionPool) Put(conn net.Conn) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return conn.Close()
	}

	select {
	case p.pool <- conn:
		return nil
	default:
		// 池已满，关闭连接
		return conn.Close()
	}
}

func (p *BasicConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	close(p.pool)
	p.closed = true

	// 关闭所有连接
	for conn := range p.pool {
		conn.Close()
	}

	return nil
}

func (s *StatsConnection) Stats() *ConnectionStats {
	return s.stats
}

func (c *ConnectionServerConfig) GetAddress() string {
	return c.Address // 直接返回结构体字段，避免循环导入
}

// 修复 line 256: undefined: strconv
func (c *ConnectionServerConfig) GetMaxMsgSize() string {
	return strconv.Itoa(c.MaxMsgSize) // 添加 strconv. 包前缀
}

// NewRoundRobinBalancer 创建新的轮询负载均衡器
func NewRoundRobinBalancer(serviceName, methodName string, collector MetricsCollector) *RoundRobinBalancer {
	return &RoundRobinBalancer{
		connections:      make([]interface{}, 0),
		index:            0,
		serviceName:      serviceName,
		methodName:       methodName,
		metricsCollector: collector,
	}
}

// Pick 轮询选择一个连接
func (r *RoundRobinBalancer) Pick(availableConns []interface{}) (interface{}, error) {
	// 修复S1009: 移除多余的nil检查，直接检查长度
	if len(availableConns) == 0 {
		r.mu.Lock()
		defer r.mu.Unlock()
		availableConns = r.connections
	}

	if len(availableConns) == 0 {
		err := errors.New("没有可用连接")
		if r.metricsCollector != nil {
			ctx := context.Background()
			startTime := time.Now()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, err)
		}
		return nil, err
	}

	// 轮询选择连接
	(r.mu).Lock()
	conn := availableConns[r.index]
	r.index = (r.index + 1) % len(availableConns)
	(r.mu).Unlock()

	return conn, nil
}

// Release 释放连接
func (r *RoundRobinBalancer) Release(conn interface{}, err error) {
	startTime := time.Now()

	if err != nil {
		if r.metricsCollector != nil {
			ctx := context.Background()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, err)
		}
		r.Remove(conn)
	} else {
		if r.metricsCollector != nil {
			ctx := context.Background()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, nil)
		}
	}
}

// Add 添加连接到负载均衡器
// 添加 添加连接到负载均衡器
func (r *RoundRobinBalancer) Add(conn interface{}) {
	(r.mu).Lock()
	defer (r.mu).Unlock()

	for _, c := range r.connections {
		if c == conn {
			return
		}
	}

	r.connections = append(r.connections, conn)
	log.Printf("添加连接到负载均衡器，当前连接数: %d", len(r.connections))

	// 使用接口收集指标而非直接调用 metrics 包
	if r.metricsCollector != nil {
		// 移除未使用的ctx和startTime变量
		r.metricsCollector.UpdateConnections(r.serviceName, len(r.connections), 0, len(r.connections))
	}
}

// Remove 从负载均衡器移除连接
func (r *RoundRobinBalancer) Remove(conn interface{}) {
	(r.mu).Lock()
	defer (r.mu).Unlock()

	for i, c := range r.connections {
		if c == conn {
			r.connections = append(r.connections[:i], r.connections[i+1:]...)
			log.Printf("从负载均衡器移除连接，当前连接数: %d", len(r.connections))

			if r.index >= len(r.connections) && len(r.connections) > 0 {
				r.index = 0
			}

			// 使用接口收集指标而非直接调用 metrics 包
			if r.metricsCollector != nil {
				// 移除未使用的ctx和startTime变量
				r.metricsCollector.UpdateConnections(r.serviceName, len(r.connections), 0, len(r.connections))
			}
			return
		}
	}
}

// Len 获取连接数量
func (r *RoundRobinBalancer) Len() int {
	(r.mu).Lock()
	defer (r.mu).Unlock()
	return len(r.connections)
}

// Close 关闭负载均衡器
func (r *RoundRobinBalancer) Close() {
	(r.mu).Lock()
	defer (r.mu).Unlock()

	r.connections = nil
	r.index = 0
	log.Println("负载均衡器已关闭")
}

// Select 实现Balancer接口
func (r *RoundRobinBalancer) Select(connections []*Connection) (*Connection, error) {
	if len(connections) == 0 {
		err := errors.New("没有可用连接")
		if r.metricsCollector != nil {
			// 使用metricsCollector代替metrics.Default
			ctx := context.Background()
			startTime := time.Now()
			r.metricsCollector.CollectResponse(ctx, r.serviceName, r.methodName, startTime, err)
		}
		return nil, err
	}

	// 轮询选择连接
	r.mu.Lock()
	conn := connections[r.index]
	r.index = (r.index + 1) % len(connections)
	r.mu.Unlock()

	return conn, nil
}

// Metrics 指标收集器包装
type Metrics struct {
	Registry  *prometheus.Registry
	Server    *http.Server
	Collector MetricsCollector // 添加Collector字段
}

// NewMetrics 创建新的指标实例
func NewMetrics(registry *prometheus.Registry) *Metrics {
	return &Metrics{
		Registry: registry,
	}
}

// 回调函数类型定义
// 确保此定义唯一存在，task_types.go中不应再有相同定义
type Callback func(result interface{}, err error)

// ConnectionHandler TCP连接处理器
type ConnectionHandler struct {
	Config         *Config            // 导出字段
	ConnectionPool *TCPConnectionPool // 导出字段
}

// HandleConnection 处理新建立的TCP连接
func (h *ConnectionHandler) HandleConnection(conn net.Conn) error {
	// 将net.Conn转换为*net.TCPConn以访问TCP特定方法
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("无效的连接类型，期望*net.TCPConn")
	}

	// 配置TCP连接参数
	if h.Config.ConnectTimeout > 0 {
		bufferSize := int(h.Config.ConnectTimeout.Milliseconds())
		if bufferSize > 0 {
			if err := tcpConn.SetReadBuffer(bufferSize); err != nil {
				log.Printf("设置读取缓冲区大小失败: %v", err)
			}
			if err := tcpConn.SetWriteBuffer(bufferSize); err != nil {
				log.Printf("设置写入缓冲区大小失败: %v", err)
			}
		}
	}

	// 设置TCP保持连接
	if err := tcpConn.SetKeepAlive(true); err != nil {
		log.Printf("设置TCP保持连接失败: %v", err)
	} else {
		// 设置保持连接间隔
		if h.Config.KeepAliveInterval > 0 {
			if err := tcpConn.SetKeepAlivePeriod(h.Config.KeepAliveInterval); err != nil {
				log.Printf("设置TCP保持连接间隔失败: %v", err)
			}
		}
	}

	// 设置连接超时
	if h.Config.IdleTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(h.Config.IdleTimeout)); err != nil {
			return fmt.Errorf("设置读超时失败: %v", err)
		}
	}

	if h.Config.IdleTimeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(h.Config.IdleTimeout)); err != nil {
			return fmt.Errorf("设置写超时失败: %v", err)
		}
	}

	// 创建带统计功能的连接 - 使用连接池的Connection类型
	// 从连接池获取一个连接包装器
	// 将 connection.Acquire 改为直接调用池的方法
	poolConn, err := h.ConnectionPool.Acquire()
	if err != nil {
		return fmt.Errorf("创建连接池连接失败: %v", err)
	}
	// 将原始连接替换为配置好的TCP连接
	poolConn.Conn = tcpConn
	poolConn.Stats = NewConnectionStats() // 修改为使用types包的构造函数

	// 添加到连接池 - 将 connection.Release 改为直接调用池的方法
	poolConn.Release(nil)
	defer func() {
		// 从连接池移除并关闭连接
		conn.Close()
	}()

	// 消息处理循环
	codec := NewCodec(conn, conn)
	for {
		// 读取消息
		msgFrame, err := codec.ReadIPCMessage()
		if err != nil {
			// 检查是否为超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时错误处理 - 继续等待新数据
				continue
			}
			return h.handleConnectionError(err)
		}

		// 成功读取后重置读超时
		if h.Config.IdleTimeout > 0 {
			if readErr := conn.SetReadDeadline(time.Now().Add(h.Config.IdleTimeout)); readErr != nil {
				return fmt.Errorf("重置读超时失败: %v", readErr)
			}
		}

		// 处理消息
		responseFrame, err := h.processMessageFrame(msgFrame)
		if err != nil {
			return h.handleConnectionError(err)
		}

		// 发送响应（如果有）
		if responseFrame != nil {
			if err := codec.WriteIPCMessage(responseFrame); err != nil {
				return h.handleConnectionError(err)
			}
		}
	}
}

// processMessageFrame 处理接收到的消息帧
func (h *ConnectionHandler) processMessageFrame(frame *MessageFrame) (*MessageFrame, error) {
	// 实现消息处理逻辑
	return &MessageFrame{
		Type:    string(MessageTypeResponse), // 显式转换为string
		Payload: []byte("已处理: " + string(frame.Payload)),
	}, nil
}

// handleConnectionError 处理连接错误
func (h *ConnectionHandler) handleConnectionError(err error) error {
	log.Printf("连接错误: %v", err)
	return fmt.Errorf("连接错误: %v", err)
}

// 添加Acquire方法到TCPConnectionPool
func (p *TCPConnectionPool) Acquire() (*Connection, error) {
	if p.Closed {
		return nil, ErrPoolClosed
	}

	for {
		p.Mu.RLock()
		defer p.Mu.RUnlock()

		// 使用负载均衡器选择连接
		if p.Balancer != nil {
			availableConns := make([]interface{}, len(p.Connections))
			for i, conn := range p.Connections {
				availableConns[i] = conn
			}
			result, err := p.Balancer.Pick(availableConns)
			if err != nil {
				return nil, err
			}
			conn, ok := result.(*Connection)
			if !ok {
				return nil, errors.New("invalid connection type returned by balancer")
			}
			return conn, nil
		}

		// 无负载均衡器时的连接选择逻辑
		for _, conn := range p.Connections {
			if !conn.InUse && !conn.Closed && time.Since(conn.LastUsed) < p.IdleTimeout {
				conn.InUse = true
				return conn, nil
			}
		}

		return nil, ErrPoolExhausted
	}
}

// 添加Release方法到Connection
func (c *Connection) Release(err error) {
	if c.Pool.Closed {
		c.Conn.Close()
		return
	}

	// 释放连接逻辑（从原connection.Release迁移）
	c.LastUsed = time.Now()
	c.InUse = false
	// ... 原有释放连接逻辑 ...
}

// NewTCPConnectionPool 创建新的 TCP 连接池
func NewTCPConnectionPool(config Config, factory func() (net.Conn, error), balancer Balancer, metrics *Metrics) *TCPConnectionPool {
	return &TCPConnectionPool{
		MaxSize:           config.MaxSize,
		MinSize:           config.MinSize,
		InitialSize:       config.InitialSize,
		IdleTimeout:       config.IdleTimeout,
		KeepAliveInterval: config.KeepAliveInterval,
		Config:            config,
		Factory:           factory,
		Balancer:          balancer,
		Metrics:           metrics,
		Done:              make(chan struct{}),
		WaitConn:          make(chan struct{}),
		Connections:       make([]*Connection, 0),
		Mu:                &sync.RWMutex{},
		Closed:            false,
	}
}
