package conn

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// PoolStats 连接池统计信息
type PoolStats struct {
	TotalConnections   int
	ActiveConnections  int
	IdleConnections    int
	WaitingRequests    int
	TotalRequests      int64
	TotalHits          int64
	TotalMisses        int64
	TotalTimeouts      int64
	TotalErrors        int64
	AverageWaitTime    time.Duration
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxSize             int           // 最大连接数
	MinSize             int           // 最小连接数
	MaxIdleTime         time.Duration // 最大空闲时间
	ConnectionTimeout   time.Duration // 连接超时时间
	HealthCheckInterval time.Duration // 健康检查间隔
	MaxRetries          int           // 最大重试次数
}

// DefaultPoolConfig 默认配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxSize:             100,
		MinSize:             10,
		MaxIdleTime:         5 * time.Minute,
		ConnectionTimeout:   30 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
	}
}

// ConnectionPool 连接池接口
type ConnectionPool interface {
	// Get 获取连接
	Get(ctx context.Context, addr string) (Connection, error)
	// Put 归还连接
	Put(conn Connection) error
	// Close 关闭连接池
	Close() error
	// Stats 获取统计信息
	Stats() PoolStats
}

// pooledConnection 池化连接包装器
type pooledConnection struct {
	Connection
	pool      *connectionPool
	addr      string
	createdAt time.Time
	lastUsed  time.Time
	inUse     bool
}

// connectionPool 连接池实现
type connectionPool struct {
	config      *PoolConfig
	connections map[string][]*pooledConnection // addr -> connections
	mu          sync.RWMutex
	waitQueue   map[string][]chan *pooledConnection
	stats       PoolStats
	statsMu     sync.RWMutex
	dialFunc    func(ctx context.Context, addr string) (Connection, error)
	closed      bool
	closeCh     chan struct{}
	wg          sync.WaitGroup
}

// NewConnectionPool 创建连接池
func NewConnectionPool(config *PoolConfig, dialFunc func(ctx context.Context, addr string) (Connection, error)) ConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	if dialFunc == nil {
		dialFunc = DefaultDialFunc
	}

	pool := &connectionPool{
		config:      config,
		connections: make(map[string][]*pooledConnection),
		waitQueue:   make(map[string][]chan *pooledConnection),
		dialFunc:    dialFunc,
		closeCh:     make(chan struct{}),
	}

	// 启动健康检查
	pool.startHealthChecker()

	return pool
}

// DefaultDialFunc 默认连接创建函数
func DefaultDialFunc(ctx context.Context, addr string) (Connection, error) {
	conn, err := net.DialTimeout("tcp", addr, 30*time.Second)
	if err != nil {
		return nil, err
	}
	
	// 生成连接ID
	id := fmt.Sprintf("conn-%d", time.Now().UnixNano())
	
	return NewTCPConnection(conn, id, 30*time.Second, 30*time.Second), nil
}

// Get 获取连接
func (p *connectionPool) Get(ctx context.Context, addr string) (Connection, error) {
	p.mu.Lock()
	
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("connection pool is closed")
	}

	// 更新统计
	p.updateStats(func(s *PoolStats) {
		s.TotalRequests++
	})

	// 查找可用连接
	connections := p.connections[addr]
	for i, conn := range connections {
		if !conn.inUse && conn.IsHealthy() {
			conn.inUse = true
			conn.lastUsed = time.Now()
			
			// 命中统计
			p.updateStats(func(s *PoolStats) {
				s.TotalHits++
				s.ActiveConnections++
				s.IdleConnections--
			})
			
			p.mu.Unlock()
			return conn, nil
		}
		
		// 清理不健康的连接
		if !conn.IsHealthy() && !conn.inUse {
			p.connections[addr] = append(connections[:i], connections[i+1:]...)
			conn.Close()
			p.updateStats(func(s *PoolStats) {
				s.TotalConnections--
				s.IdleConnections--
			})
		}
	}

	// 检查是否达到最大连接数
	totalConns := 0
	for _, conns := range p.connections {
		totalConns += len(conns)
	}

	if totalConns >= p.config.MaxSize {
		// 需要等待
		waitCh := make(chan *pooledConnection, 1)
		if p.waitQueue[addr] == nil {
			p.waitQueue[addr] = []chan *pooledConnection{}
		}
		p.waitQueue[addr] = append(p.waitQueue[addr], waitCh)
		
		p.updateStats(func(s *PoolStats) {
			s.WaitingRequests++
		})
		
		p.mu.Unlock()

		// 等待可用连接
		select {
		case conn := <-waitCh:
			if conn != nil {
				return conn, nil
			}
			return nil, fmt.Errorf("failed to get connection from wait queue")
		case <-ctx.Done():
			// 从等待队列中移除
			p.mu.Lock()
			p.removeFromWaitQueue(addr, waitCh)
			p.mu.Unlock()
			
			p.updateStats(func(s *PoolStats) {
				s.TotalTimeouts++
			})
			
			return nil, ctx.Err()
		}
	}

	// 创建新连接
	p.updateStats(func(s *PoolStats) {
		s.TotalMisses++
	})
	
	p.mu.Unlock()

	// 在锁外创建连接
	newConn, err := p.dialFunc(ctx, addr)
	if err != nil {
		p.updateStats(func(s *PoolStats) {
			s.TotalErrors++
		})
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// 包装连接
	pooledConn := &pooledConnection{
		Connection: newConn,
		pool:       p,
		addr:       addr,
		createdAt:  time.Now(),
		lastUsed:   time.Now(),
		inUse:      true,
	}

	// 添加到池中
	p.mu.Lock()
	if p.connections[addr] == nil {
		p.connections[addr] = []*pooledConnection{}
	}
	p.connections[addr] = append(p.connections[addr], pooledConn)
	
	p.updateStats(func(s *PoolStats) {
		s.TotalConnections++
		s.ActiveConnections++
	})
	
	p.mu.Unlock()

	return pooledConn, nil
}

// Put 归还连接
func (p *connectionPool) Put(conn Connection) error {
	pooledConn, ok := conn.(*pooledConnection)
	if !ok {
		return fmt.Errorf("invalid connection type")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		conn.Close()
		return fmt.Errorf("connection pool is closed")
	}

	// 检查连接健康状态
	if !pooledConn.IsHealthy() {
		// 不健康的连接直接关闭
		p.removeConnection(pooledConn)
		pooledConn.Close()
		return nil
	}

	// 检查是否有等待的请求
	if waitQueue, exists := p.waitQueue[pooledConn.addr]; exists && len(waitQueue) > 0 {
		waitCh := waitQueue[0]
		p.waitQueue[pooledConn.addr] = waitQueue[1:]
		
		// 将连接交给等待者
		select {
		case waitCh <- pooledConn:
			p.updateStats(func(s *PoolStats) {
				s.WaitingRequests--
			})
			return nil
		default:
		}
	}

	// 标记为未使用
	pooledConn.inUse = false
	pooledConn.lastUsed = time.Now()
	
	p.updateStats(func(s *PoolStats) {
		s.ActiveConnections--
		s.IdleConnections++
	})

	return nil
}

// Close 关闭连接池
func (p *connectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	close(p.closeCh)
	
	// 关闭所有连接
	for _, conns := range p.connections {
		for _, conn := range conns {
			conn.Close()
		}
	}
	
	// 通知所有等待者
	for _, waitQueue := range p.waitQueue {
		for _, waitCh := range waitQueue {
			close(waitCh)
		}
	}
	
	p.connections = nil
	p.waitQueue = nil
	
	p.mu.Unlock()
	
	// 等待后台任务完成
	p.wg.Wait()
	
	return nil
}

// Stats 获取统计信息
func (p *connectionPool) Stats() PoolStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()
	return p.stats
}

// startHealthChecker 启动健康检查器
func (p *connectionPool) startHealthChecker() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		
		ticker := time.NewTicker(p.config.HealthCheckInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				p.performHealthCheck()
			case <-p.closeCh:
				return
			}
		}
	}()
}

// performHealthCheck 执行健康检查
func (p *connectionPool) performHealthCheck() {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	now := time.Now()
	
	for addr, conns := range p.connections {
		validConns := make([]*pooledConnection, 0, len(conns))
		
		for _, conn := range conns {
			// 跳过正在使用的连接
			if conn.inUse {
				validConns = append(validConns, conn)
				continue
			}
			
			// 检查空闲时间
			if now.Sub(conn.lastUsed) > p.config.MaxIdleTime {
				conn.Close()
				p.updateStats(func(s *PoolStats) {
					s.TotalConnections--
					s.IdleConnections--
				})
				continue
			}
			
			// 检查健康状态
			if !conn.IsHealthy() {
				conn.Close()
				p.updateStats(func(s *PoolStats) {
					s.TotalConnections--
					s.IdleConnections--
				})
				continue
			}
			
			validConns = append(validConns, conn)
		}
		
		// 更新连接列表
		if len(validConns) > 0 {
			p.connections[addr] = validConns
		} else {
			delete(p.connections, addr)
		}
		
		// 确保最小连接数
		if len(validConns) < p.config.MinSize {
			go p.ensureMinConnections(addr, p.config.MinSize-len(validConns))
		}
	}
}

// ensureMinConnections 确保最小连接数
func (p *connectionPool) ensureMinConnections(addr string, count int) {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectionTimeout)
	defer cancel()
	
	for i := 0; i < count; i++ {
		conn, err := p.dialFunc(ctx, addr)
		if err != nil {
			continue
		}
		
		pooledConn := &pooledConnection{
			Connection: conn,
			pool:       p,
			addr:       addr,
			createdAt:  time.Now(),
			lastUsed:   time.Now(),
			inUse:      false,
		}
		
		p.mu.Lock()
		if p.closed {
			p.mu.Unlock()
			conn.Close()
			break
		}
		
		if p.connections[addr] == nil {
			p.connections[addr] = []*pooledConnection{}
		}
		p.connections[addr] = append(p.connections[addr], pooledConn)
		
		p.updateStats(func(s *PoolStats) {
			s.TotalConnections++
			s.IdleConnections++
		})
		
		p.mu.Unlock()
	}
}

// removeConnection 从池中移除连接
func (p *connectionPool) removeConnection(conn *pooledConnection) {
	conns := p.connections[conn.addr]
	for i, c := range conns {
		if c == conn {
			p.connections[conn.addr] = append(conns[:i], conns[i+1:]...)
			
			p.updateStats(func(s *PoolStats) {
				s.TotalConnections--
				if conn.inUse {
					s.ActiveConnections--
				} else {
					s.IdleConnections--
				}
			})
			
			if len(p.connections[conn.addr]) == 0 {
				delete(p.connections, conn.addr)
			}
			break
		}
	}
}

// removeFromWaitQueue 从等待队列中移除
func (p *connectionPool) removeFromWaitQueue(addr string, ch chan *pooledConnection) {
	waitQueue := p.waitQueue[addr]
	for i, waitCh := range waitQueue {
		if waitCh == ch {
			p.waitQueue[addr] = append(waitQueue[:i], waitQueue[i+1:]...)
			close(ch)
			
			p.updateStats(func(s *PoolStats) {
				s.WaitingRequests--
			})
			
			if len(p.waitQueue[addr]) == 0 {
				delete(p.waitQueue, addr)
			}
			break
		}
	}
}

// updateStats 更新统计信息
func (p *connectionPool) updateStats(fn func(*PoolStats)) {
	p.statsMu.Lock()
	defer p.statsMu.Unlock()
	fn(&p.stats)
}