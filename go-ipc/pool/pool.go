package pool

import (
	"context"
	"errors"
	"net"
	"sync"
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

// Config 连接池配置
type Config struct {
	// 基础配置
	InitialSize    int           // 初始连接数
	MinSize        int           // 最小连接数
	MaxSize        int           // 最大连接数
	ConnectTimeout time.Duration // 连接超时
	IdleTimeout    time.Duration // 空闲超时

	// 自动扩缩容配置
	AutoScaling     bool    // 是否启用自动扩缩容
	ScaleUpThreshold   float64 // 扩容阈值（活跃连接比例）
	ScaleDownThreshold float64 // 缩容阈值（空闲连接比例）
	ScaleStep         int     // 每次扩缩容步长

	// 健康检查配置
	HealthCheck         bool          // 是否启用健康检查
	HealthCheckInterval time.Duration // 健康检查间隔
	MaxErrorCount       int           // 最大错误次数
	MaxLatency         time.Duration  // 最大延迟阈值

	// 负载均衡配置
	LoadBalance LoadBalanceStrategy // 负载均衡策略
}

// Connection 连接包装
type Connection struct {
	conn      net.Conn         // 底层连接
	pool      *ConnectionPool  // 所属连接池
	Stats     *ConnectionStats // 连接统计
	lastCheck time.Time        // 最后检查时间
	closed    bool            // 是否已关闭
}

// ConnectionPool 连接池
type ConnectionPool struct {
	mu          sync.RWMutex
	config      Config
	factory     func() (net.Conn, error)
	connections []*Connection
	balancer    Balancer
	closed      bool

	// 监控指标
	metrics struct {
		totalConnections int64
		activeConnections int64
		idleConnections int64
		waitingRequests int64
	}

	// 控制通道
	done     chan struct{}
	waitConn chan struct{} // 等待可用连接的通道
}

// NewConnectionPool 创建连接池
func NewConnectionPool(config Config, factory func() (net.Conn, error)) (*ConnectionPool, error) {
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	pool := &ConnectionPool{
		config:   config,
		factory:  factory,
		balancer: NewBalancer(config.LoadBalance),
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

// createConnection 创建新连接
func (p *ConnectionPool) createConnection() (*Connection, error) {
	conn, err := p.factory()
	if err != nil {
		return nil, err
	}

	return &Connection{
		conn:      conn,
		pool:      p,
		Stats:     NewConnectionStats(),
		lastCheck: time.Now(),
	}, nil
}

// Acquire 获取连接
func (p *ConnectionPool) Acquire(ctx context.Context) (*Connection, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, ErrPoolClosed
	}

	// 使用负载均衡器选择连接
	if conn := p.balancer.Select(p.getAvailableConnections()); conn != nil {
		p.activateConnection(conn)
		p.mu.Unlock()
		return conn, nil
	}

	// 检查是否可以创建新连接
	if len(p.connections) < p.config.MaxSize {
		conn, err := p.createConnection()
		if err != nil {
			p.mu.Unlock()
			return nil, err
		}
		p.connections = append(p.connections, conn)
		p.activateConnection(conn)
		p.mu.Unlock()
		return conn, nil
	}

	// 等待可用连接
	p.metrics.waitingRequests++
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		p.mu.Lock()
		p.metrics.waitingRequests--
		p.mu.Unlock()
		return nil, ctx.Err()
	case <-p.waitConn:
		p.mu.Lock()
		p.metrics.waitingRequests--
		if conn := p.balancer.Select(p.getAvailableConnections()); conn != nil {
			p.activateConnection(conn)
			p.mu.Unlock()
			return conn, nil
		}
		p.mu.Unlock()
		return nil, ErrPoolExhausted
	}
}

// Release 释放连接
func (p *ConnectionPool) Release(conn *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if conn.closed {
		return
	}

	conn.Stats.ActiveRequests--
	conn.Stats.LastUsedTime = time.Now()

	// 通知等待的请求
	if p.metrics.waitingRequests > 0 {
		select {
		case p.waitConn <- struct{}{}:
		default:
		}
	}
}

// Close 关闭连接池
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.done)

	for _, conn := range p.connections {
		conn.conn.Close()
		conn.closed = true
	}

	p.connections = nil
	return nil
}

// maintain 维护连接池
func (p *ConnectionPool) maintain() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.mu.Lock()

			// 健康检查
			if p.config.HealthCheck {
				p.performHealthCheck()
			}

			// 自动扩缩容
			if p.config.AutoScaling {
				p.adjustPoolSize()
			}

			// 清理空闲连接
			p.cleanIdleConnections()

			p.mu.Unlock()
		}
	}
}

// performHealthCheck 执行健康检查
func (p *ConnectionPool) performHealthCheck() {
	now := time.Now()
	for i := 0; i < len(p.connections); i++ {
		conn := p.connections[i]
		if now.Sub(conn.lastCheck) < p.config.HealthCheckInterval {
			continue
		}

		// 检查错误次数
		if conn.Stats.ErrorCount > int64(p.config.MaxErrorCount) {
			p.removeConnection(i)
			i--
			continue
		}

		// 检查平均延迟
		if avgLatency := conn.Stats.LatencyStats.Average(); avgLatency > p.config.MaxLatency {
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

// adjustPoolSize 调整连接池大小
func (p *ConnectionPool) adjustPoolSize() {
	total := len(p.connections)
	active := 0
	for _, conn := range p.connections {
		if conn.Stats.ActiveRequests > 0 {
			active++
		}
	}

	// 计算使用率
	activeRatio := float64(active) / float64(total)
	idleRatio := float64(total-active) / float64(total)

	// 扩容
	if activeRatio >= p.config.ScaleUpThreshold && total < p.config.MaxSize {
		toAdd := min(p.config.ScaleStep, p.config.MaxSize-total)
		for i := 0; i < toAdd; i++ {
			if conn, err := p.createConnection(); err == nil {
				p.connections = append(p.connections, conn)
			}
		}
	}

	// 缩容
	if idleRatio >= p.config.ScaleDownThreshold && total > p.config.MinSize {
		toRemove := min(p.config.ScaleStep, total-p.config.MinSize)
		for i := 0; i < toRemove; i++ {
			// 移除最久未使用的空闲连接
			if idx := p.findLeastUsedIdleConnection(); idx >= 0 {
				p.removeConnection(idx)
			}
		}
	}
}

// cleanIdleConnections 清理空闲连接
func (p *ConnectionPool) cleanIdleConnections() {
	now := time.Now()
	for i := 0; i < len(p.connections); i++ {
		conn := p.connections[i]
		if conn.Stats.ActiveRequests == 0 &&
			now.Sub(conn.Stats.LastUsedTime) > p.config.IdleTimeout &&
			len(p.connections) > p.config.MinSize {
			p.removeConnection(i)
			i--
		}
	}
}

// removeConnection 移除连接
func (p *ConnectionPool) removeConnection(index int) {
	conn := p.connections[index]
	conn.conn.Close()
	conn.closed = true
	p.connections = append(p.connections[:index], p.connections[index+1:]...)
}

// checkConnection 检查连接健康状态
func (p *ConnectionPool) checkConnection(conn *Connection) error {
	// 这里可以实现具体的健康检查逻辑
	// 例如：发送心跳包、检查连接状态等
	return nil
}

// activateConnection 激活连接
func (p *ConnectionPool) activateConnection(conn *Connection) {
	conn.Stats.ActiveRequests++
	conn.Stats.TotalRequests++
	conn.Stats.LastUsedTime = time.Now()
}

// getAvailableConnections 获取可用连接列表
func (p *ConnectionPool) getAvailableConnections() []*Connection {
	available := make([]*Connection, 0, len(p.connections))
	for _, conn := range p.connections {
		if !conn.closed && conn.Stats.ActiveRequests == 0 {
			available = append(available, conn)
		}
	}
	return available
}

// findLeastUsedIdleConnection 查找最少使用的空闲连接
func (p *ConnectionPool) findLeastUsedIdleConnection() int {
	var (
		leastUsed     = -1
		leastRequests int64 = 1<<63 - 1
	)

	for i, conn := range p.connections {
		if conn.Stats.ActiveRequests == 0 && conn.Stats.TotalRequests < leastRequests {
			leastUsed = i
			leastRequests = conn.Stats.TotalRequests
		}
	}

	return leastUsed
}

// GetStats 获取连接池统计信息
func (p *ConnectionPool) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_connections"] = len(p.connections)
	stats["waiting_requests"] = p.metrics.waitingRequests

	active := 0
	errors := int64(0)
	for _, conn := range p.connections {
		if conn.Stats.ActiveRequests > 0 {
			active++
		}
		errors += conn.Stats.ErrorCount
	}

	stats["active_connections"] = active
	stats["idle_connections"] = len(p.connections) - active
	stats["total_errors"] = errors

	return stats
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
} 