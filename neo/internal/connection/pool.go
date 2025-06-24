package connection

import (
	"context"
	"errors"
	"fmt"
	"neo/internal/config"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

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
	KeepAliveInterval time.Duration // 保持连接间隔（新增）

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
	LoadBalance         LoadBalanceStrategy // 负载均衡策略
	MaxRetryCount       int                 // 连接创建最大重试次数
	RetryInterval       time.Duration       // 连接重试间隔
}

// PoolConfig 连接池配置参数
type PoolConfig struct {
	MinSize   int // 最小连接数
	MaxSize   int // 最大连接数
	ExpandPct int // 扩容阈值（当前连接数超过MinSize*ExpandPct%时扩容）
	ShrinkPct int // 缩容阈值（当前连接数低于MaxSize*ShrinkPct%时缩容）
}

// Connection 统一连接结构体
type Connection struct {
	Conn       net.Conn           // 底层连接（已修复：改为导出字段）
	pool       *TCPConnectionPool // 所属连接池
	Stats      *ConnectionStats   // 连接统计 (使用stats.go中的定义)
	lastUsed   time.Time          // 最后使用时间
	lastCheck  time.Time          // 最后检查时间
	inUse      bool               // 是否正在使用
	errorCount int                // 错误次数
	closed     bool               // 是否已关闭
}

// TCPConnectionPool 连接池结构体
type TCPConnectionPool struct {
	mu          sync.RWMutex
	config      Config                   // 合并后的完整配置
	factory     func() (net.Conn, error) // 连接创建工厂
	connections []*Connection            // 合并后的连接列表
	balancer    Balancer                 // 负载均衡器
	closed      bool                     // 关闭状态标记

	// 基础连接池参数
	maxSize int           // 最大连接数
	minSize int           // 最小连接数
	timeout time.Duration // 连接超时时间

	// 监控指标
	metrics struct {
		totalConnections  int64
		activeConnections int64
		idleConnections   int64
		waitingRequests   int64
	}

	// 控制通道
	done     chan struct{}
	waitConn chan struct{} // 等待可用连接的通道
}

// NewTCPConnectionPool 创建连接池
func NewTCPConnectionPool(factory func() (net.Conn, error)) (*TCPConnectionPool, error) {
	cfg := config.Get().Pool
	config := Config{
		MaxSize:           cfg.MaxSize,
		MinSize:           cfg.MinSize,
		InitialSize:       cfg.InitialSize,
		IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
		KeepAliveInterval: time.Duration(cfg.KeepAliveInterval) * time.Second, // 新增：加载保持连接间隔配置
		MaxRetryCount:     3,
		RetryInterval:     100 * time.Millisecond,
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	// 创建指标收集器（实际使用时需要正确导入并初始化）
	var metricsCollector MetricsCollector

	pool := &TCPConnectionPool{
		config:  config,
		factory: factory,
		// 创建负载均衡器，传入策略和指标收集器
		balancer: NewBalancer(config.LoadBalance, "tcp", "connection", metricsCollector),
		done:     make(chan struct{}),
		waitConn: make(chan struct{}, config.MaxSize),
	}

	// 初始化连接
	for i := 0; i < config.InitialSize; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			pool.Close()
			return nil, err
		}
		pool.connections = append(pool.connections, conn)
	}

	// 启动维护协程
	if config.AutoScaling || config.HealthCheck {
		go pool.maintain()
	}

	return pool, nil
}

// validateConfig 验证配置
func validateConfig(config Config) error {
	if config.MinSize < 0 {
		return errors.New("minimum size must be non-negative")
	}
	if config.MaxSize < config.MinSize {
		return errors.New("maximum size must be greater than minimum size")
	}
	if config.InitialSize < config.MinSize || config.InitialSize > config.MaxSize {
		return errors.New("initial size must be between minimum and maximum size")
	}
	return nil
}

// createConnection 创建新连接（内部未导出方法）
func (p *TCPConnectionPool) createConnection() (*Connection, error) {
	var conn net.Conn
	var err error

	// 使用配置的重试参数
	for i := 0; i < p.config.MaxRetryCount; i++ {
		conn, err = p.factory()
		if err == nil {
			return &Connection{
				Conn:      conn, // 已修复：使用导出字段Conn
				pool:      p,
				Stats:     NewConnectionStats(), // 使用stats.go中的构造函数
				lastCheck: time.Now(),
			}, nil
		}
		if i < p.config.MaxRetryCount-1 {
			time.Sleep(p.config.RetryInterval)
		}
	}
	return nil, fmt.Errorf("达到最大重试次数 %d: %v", p.config.MaxRetryCount, err)
}

// CreateConnection 创建新连接（导出方法供外部调用）
func (p *TCPConnectionPool) CreateConnection() (*Connection, error) {
	return p.createConnection()
}

// Acquire 获取连接
func (p *TCPConnectionPool) Acquire(ctx context.Context) (*Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// 使用负载均衡器选择可用连接
	availableConns := p.getAvailableConnections()
	// 将[]*Connection转换为[]interface{}
	connInterfaces := make([]interface{}, len(availableConns))
	for i, c := range availableConns {
		connInterfaces[i] = c
	}
	// 正确处理Pick方法的两个返回值
	conn, err := p.balancer.Pick(connInterfaces)
	if err == nil && conn != nil {
		// 类型断言转换回*Connection
		if typedConn, ok := conn.(*Connection); ok {
			// 标记连接为使用中
			typedConn.inUse = true
			typedConn.lastUsed = time.Now()
			atomic.AddInt64(&p.metrics.activeConnections, 1)
			return typedConn, nil
		}
	}

	// 连接池耗尽，无法创建新连接
	if len(p.connections) >= p.config.MaxSize {
		return nil, ErrPoolExhausted
	}

	// 创建新连接
	newConn, err := p.createConnection()
	if err != nil {
		return nil, err
	}

	newConn.inUse = true
	newConn.lastUsed = time.Now()
	p.connections = append(p.connections, newConn)
	atomic.AddInt64(&p.metrics.totalConnections, 1)
	atomic.AddInt64(&p.metrics.activeConnections, 1)

	return newConn, nil
}

// Release 释放连接
func (p *TCPConnectionPool) Release(conn *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn.closed {
		return
	}

	conn.inUse = false
	conn.lastUsed = time.Now()

	// 通知等待的请求
	if p.metrics.waitingRequests > 0 {
		select {
		case p.waitConn <- struct{}{}:
		default:
		}
	}
}

// Close 关闭连接池
func (p *TCPConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.done)

	for _, conn := range p.connections {
		conn.Conn.Close() // 已修复：使用导出字段Conn
		conn.closed = true
	}

	p.connections = nil
	return nil
}

// maintain 维护连接池（自动扩缩容、健康检查等）
func (p *TCPConnectionPool) maintain() {
	if p.config.HealthCheckInterval == 0 {
		p.config.HealthCheckInterval = 30 * time.Second
	}
	ticker := time.NewTicker(p.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 执行健康检查（当配置启用时）
			if p.config.HealthCheck {
				p.performHealthCheck()
			}
			// 清理空闲连接（当配置启用自动扩缩容时）
			if p.config.AutoScaling {
				p.cleanIdleConnections()
			}
		case <-p.done:
			return
		}
	}
}

// performHealthCheck 执行健康检查
func (p *TCPConnectionPool) performHealthCheck() {
	now := time.Now()
	for i := 0; i < len(p.connections); i++ {
		conn := p.connections[i]
		if now.Sub(conn.lastCheck) < p.config.HealthCheckInterval {
			continue
		}

		// 检查错误次数
		if conn.errorCount > p.config.MaxErrorCount {
			p.removeConnection(i)
			i--
			continue
		}

		// 执行健康检查
		if err := p.checkConnection(conn); err != nil {
			p.removeConnection(i)
			i--
			continue
		}

		conn.lastCheck = now
	}
}

// cleanIdleConnections 清理空闲连接
func (p *TCPConnectionPool) cleanIdleConnections() {
	now := time.Now()
	for i := 0; i < len(p.connections); i++ {
		conn := p.connections[i]
		if !conn.inUse &&
			now.Sub(conn.lastUsed) > p.config.IdleTimeout &&
			len(p.connections) > p.config.MinSize {
			p.removeConnection(i)
			i--
		}
	}
}

// removeConnection 移除连接
func (p *TCPConnectionPool) removeConnection(index int) {
	conn := p.connections[index]
	conn.Conn.Close() // 已修复：使用导出字段Conn
	conn.closed = true
	p.connections = append(p.connections[:index], p.connections[index+1:]...)
}

// checkConnection 检查连接健康状态
func (p *TCPConnectionPool) checkConnection(conn *Connection) error {
	// 实现具体的健康检查逻辑
	testData := []byte("heartbeat")
	if _, err := conn.Conn.Write(testData); err != nil { // 已修复：使用导出字段Conn
		return errors.New("connection write failed: " + err.Error())
	}
	// 设置读取超时
	conn.Conn.SetReadDeadline(time.Now().Add(1 * time.Second)) // 已修复：使用导出字段Conn
	buf := make([]byte, len(testData))
	if _, err := conn.Conn.Read(buf); err != nil { // 已修复：使用导出字段Conn
		return fmt.Errorf("connection read failed: %v", err)
	}
	// 重置读取超时
	conn.Conn.SetReadDeadline(time.Time{}) // 已修复：使用导出字段Conn
	return nil
}

// getAvailableConnections 获取可用连接列表
func (p *TCPConnectionPool) getAvailableConnections() []*Connection {
	available := make([]*Connection, 0, len(p.connections))
	for _, conn := range p.connections {
		if !conn.closed && !conn.inUse {
			available = append(available, conn)
		}
	}
	return available
}

// GetStats 获取连接池统计信息
func (p *TCPConnectionPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_connections"] = len(p.connections)
	stats["waiting_requests"] = p.metrics.waitingRequests

	active := 0
	errorCount := 0
	for _, conn := range p.connections {
		if conn.inUse {
			active++
		}
		errorCount += conn.errorCount
	}

	stats["active_connections"] = active
	stats["idle_connections"] = len(p.connections) - active
	stats["total_errors"] = errorCount

	return stats
}
