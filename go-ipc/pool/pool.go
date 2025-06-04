package pool

import (
	"context"
	"errors"
	"fmt"
	"log" // 新增log包导入
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

// Config 统一连接池配置（合并后）
type Config struct {
	// 基础配置（原connection_pool.go字段）
	MaxSize        int           // 最大连接数
	MinSize        int           // 最小连接数
	ConnectTimeout time.Duration // 连接超时时间
	IdleTimeout    time.Duration // 空闲超时时间

	// 扩展配置（原pool.go字段）
	InitialSize         int                 // 初始连接数
	AutoScaling         bool                // 是否启用自动扩缩容
	ScaleUpThreshold    float64             // 扩容阈值（活跃连接比例）
	ScaleDownThreshold  float64             // 缩容阈值（空闲连接比例）
	ScaleStep           int                 // 每次扩缩容步长
	HealthCheck         bool                // 是否启用健康检查
	HealthCheckInterval time.Duration       // 健康检查间隔
	MaxErrorCount       int                 // 最大错误次数
	MaxLatency          time.Duration       // 最大延迟阈值
	LoadBalance         LoadBalanceStrategy // 负载均衡策略（来自balancer.go）
}

// 连接池结构体新增扩缩容阈值配置
type PoolConfig struct {
	MinSize   int // 最小连接数
	MaxSize   int // 最大连接数
	ExpandPct int // 扩容阈值（当前连接数超过MinSize*ExpandPct%时扩容）
	ShrinkPct int // 缩容阈值（当前连接数低于MaxSize*ShrinkPct%时缩容）
	// ... 其他配置
}

// Connection 统一连接结构体（合并后）
type Connection struct {
	conn       net.Conn         // 底层连接
	pool       *ConnectionPool  // 所属连接池
	Stats      *ConnectionStats // 连接统计（来自stats.go）
	lastUsed   time.Time        // 最后使用时间（原connection_pool.go字段）
	lastCheck  time.Time        // 最后检查时间（原pool.go字段）
	inUse      bool             // 是否正在使用（原connection_pool.go字段）
	errorCount int              // 错误次数（原connection_pool.go字段）
	closed     bool             // 是否已关闭（原pool.go字段）
}

// ConnectionPool 统一连接池结构体（合并后）
type ConnectionPool struct {
	mu          sync.RWMutex
	config      Config                   // 合并后的完整配置（含基础+扩展参数）
	factory     func() (net.Conn, error) // 连接创建工厂
	connections []*Connection            // 合并后的连接列表（含统计和健康状态）
	balancer    Balancer                 // 负载均衡器（来自balancer.go）
	closed      bool                     // 关闭状态标记

	// 基础连接池参数（原connection_pool.go字段）
	maxSize int           // 最大连接数
	minSize int           // 最小连接数
	timeout time.Duration // 连接超时时间

	// 监控指标（原pool.go字段）
	metrics struct {
		totalConnections  int64
		activeConnections int64
		idleConnections   int64
		waitingRequests   int64
	}

	// 控制通道（原pool.go字段）
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

// Acquire 获取连接（合并后逻辑）
func (p *ConnectionPool) Acquire(ctx context.Context) (*Connection, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, ErrPoolClosed
	}

	// 1. 使用负载均衡器选择可用连接（来自balancer.go）
	availableConns := p.getAvailableConnections() // 过滤未使用且健康的连接
	if conn := p.balancer.Select(availableConns); conn != nil {
		conn.inUse = true
		conn.lastUsed = time.Now()
		p.metrics.activeConnections++
		return conn, nil
	}

	// 2. 若没有可用连接且未达最大限制，创建新连接（原connection_pool.go逻辑）
	if len(p.connections) < p.maxSize {
		newConn, err := p.factory()
		if err != nil {
			return nil, err
		}
		wrappedConn := &Connection{
			conn:     newConn,
			pool:     p,
			Stats:    NewConnectionStats(),
			lastUsed: time.Now(),
			inUse:    true,
		}
		p.connections = append(p.connections, wrappedConn)
		p.metrics.totalConnections++
		p.metrics.activeConnections++
		return wrappedConn, nil
	}

	// 3. 连接池耗尽，等待可用连接（原pool.go逻辑）
	p.metrics.waitingRequests++
	select {
	case <-ctx.Done():
		p.metrics.waitingRequests--
		return nil, ctx.Err()
	case <-p.waitConn:
		// 等待后重新尝试获取
		return p.Acquire(ctx)
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

// maintain 维护连接池（优化后：使用配置参数）
func (p *ConnectionPool) maintain() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.mu.Lock()
			totalConns := len(p.connections)
			activeConns := int(p.metrics.activeConnections)
			idleConns := totalConns - activeConns

			usageRatio := float64(activeConns) / float64(totalConns)
			scaleUpThreshold := p.config.ScaleUpThreshold
			scaleStep := p.config.ScaleStep
			scaleDownIdleThreshold := p.config.ScaleDownThreshold // 原float64类型

			// 扩容逻辑（修复log未定义）
			if usageRatio > scaleUpThreshold && totalConns < p.config.MaxSize {
				newConns := min(
					scaleStep,
					p.config.MaxSize-totalConns,
				)
				for i := 0; i < newConns; i++ {
					conn, err := p.createConnection()
					if err == nil {
						p.connections = append(p.connections, conn)
						p.metrics.totalConnections++
					}
				}
				log.Printf("扩容 %d 个连接（阈值：%.2f）", newConns, scaleUpThreshold) // 已导入log包
			}

			// 缩容逻辑（修复类型不匹配）
			if idleConns > int(scaleDownIdleThreshold) && totalConns > p.config.MinSize { // 转换为int比较
				removeCount := min(
					idleConns-1,
					totalConns-p.config.MinSize,
				)
				for i := 0; i < removeCount; i++ {
					if len(p.connections) == 0 {
						break
					}
					for j, conn := range p.connections {
						if !conn.inUse {
							conn.conn.Close()
							p.connections = append(p.connections[:j], p.connections[j+1:]...)
							p.metrics.totalConnections--
							removeCount--
							break
						}
					}
				}
				log.Printf("缩容 %d 个连接（阈值：%d）", removeCount, int(scaleDownIdleThreshold)) // 已导入log包
			}

			p.mu.Unlock()
		case <-p.done:
			return
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
// AutoAdjust 自动调整连接池大小（修复后）
func (p *ConnectionPool) AutoAdjust() { // 接收者修正为 *ConnectionPool
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock() // 加锁保证并发安全
		defer p.mu.Unlock()

		total := len(p.connections) // 总连接数从 ConnectionPool 的 connections 字段获取
		if total == 0 {
			continue // 避免除零错误
		}

		// 统计活跃连接数（ActiveRequests > 0 的连接）
		active := 0
		for _, conn := range p.connections {
			if conn.Stats.ActiveRequests > 0 {
				active++
			}
		}

		// 扩容逻辑：活跃比例 > 扩容阈值 且 未达最大连接数
		if float64(active)/float64(total) > p.config.ScaleUpThreshold && total < p.config.MaxSize {
			toAdd := min(
				p.config.ScaleStep,     // 从 config 中获取步长
				p.config.MaxSize-total, // 不超过最大限制
			)
			for i := 0; i < toAdd; i++ {
				conn, err := p.createConnection() // 使用现有 createConnection 方法创建连接
				if err == nil {
					p.connections = append(p.connections, conn) // 添加新连接到连接池
				}
			}
			log.Printf("AutoAdjust: 扩容 %d 个连接（活跃比例: %.2f）", toAdd, float64(active)/float64(total))
		}

		// 缩容逻辑：空闲比例 > 缩容阈值 且 超过最小连接数
		idle := total - active
		if float64(idle)/float64(total) > p.config.ScaleDownThreshold && total > p.config.MinSize {
			toRemove := min(
				p.config.ScaleStep,     // 从 config 中获取步长
				total-p.config.MinSize, // 不低于最小限制
			)
			for i := 0; i < toRemove; i++ {
				idx := p.findLeastUsedIdleConnection() // 使用现有方法查找最久未使用的空闲连接
				if idx >= 0 {
					p.removeConnection(idx) // 使用现有 removeConnection 方法移除连接
				}
			}
			log.Printf("AutoAdjust: 缩容 %d 个连接（空闲比例: %.2f）", toRemove, float64(idle)/float64(total))
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
	// 实现具体的健康检查逻辑（使用 conn 参数）
	// 示例：尝试向连接写入一个心跳包并读取响应
	testData := []byte("heartbeat")
	// 使用 conn.conn（底层 net.Conn）执行写操作
	if _, err := conn.conn.Write(testData); err != nil {
		return errors.New("connection write failed: " + err.Error())
	}
	// 读取响应验证连接状态
	buf := make([]byte, len(testData))
	if _, err := conn.conn.Read(buf); err != nil {
		return fmt.Errorf("connection read failed: %v", err)
	}
	return nil
}

// activateConnection 激活连接
// func (p *ConnectionPool) activateConnection(conn *Connection) {
// 	conn.Stats.ActiveRequests++
// 	conn.Stats.TotalRequests++
// 	conn.Stats.LastUsedTime = time.Now()
// }

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
		leastUsed           = -1
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
