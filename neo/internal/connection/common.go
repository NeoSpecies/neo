package connection

import (
	"errors"
	"net"
	"sync"
	"time"

	"neo/internal/metrics"
	"neo/internal/types"
)

// 带统计功能的连接包装器
type StatsConnection struct {
	net.Conn
	stats *types.ConnectionStats
	mu    sync.Mutex
}

// 创建新的带统计功能的连接
func NewStatsConnection(conn net.Conn) *StatsConnection {
	return &StatsConnection{
		Conn:  conn,
		stats: types.NewConnectionStats(),
	}
}

// 读取数据
func (c *StatsConnection) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		c.mu.Lock()
		c.stats.BytesRead += uint64(n) // 已与stats.go中的uint64类型匹配
		c.stats.LastActive = time.Now()
		c.mu.Unlock()
		metrics.RecordMessageSize("connection", "read", "bytes", int64(n))
	}
	return n, err
}

// 写入数据
func (c *StatsConnection) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.mu.Lock()
		c.stats.BytesWritten += uint64(n) // 已与stats.go中的uint64类型匹配
		c.stats.LastActive = time.Now()
		c.mu.Unlock()
		metrics.RecordMessageSize("connection", "write", "bytes", int64(n))
	}
	return n, err
}

// 获取连接统计信息
func (s *StatsConnection) Stats() *types.ConnectionStats {
	return s.stats
}

// 带超时控制的连接
type TimeoutConnection struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// 创建新的带超时控制的连接
func NewTimeoutConnection(conn net.Conn, readTimeout, writeTimeout time.Duration) *TimeoutConnection {
	return &TimeoutConnection{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// 读取数据
func (c *TimeoutConnection) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Read(b)
}

// 写入数据
func (c *TimeoutConnection) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(b)
}

// 设置读取超时
func (c *TimeoutConnection) SetReadTimeout(timeout time.Duration) {
	c.readTimeout = timeout
}

// 设置写入超时
func (c *TimeoutConnection) SetWriteTimeout(timeout time.Duration) {
	c.writeTimeout = timeout
}

// 连接池接口
type ConnectionPool interface {
	Get() (net.Conn, error)
	Put(conn net.Conn)
	Close()
}

// 基础连接池实现
type BasicConnectionPool struct {
	pool     chan net.Conn
	mu       sync.Mutex
	closed   bool
	createFn func() (net.Conn, error)
}

// 创建新的基础连接池
func NewBasicConnectionPool(size int, createFn func() (net.Conn, error)) (*BasicConnectionPool, error) {
	if size <= 0 {
		return nil, errors.New("pool size must be positive")
	}
	if createFn == nil {
		return nil, errors.New("create function must not be nil")
	}
	return &BasicConnectionPool{
		pool:     make(chan net.Conn, size),
		createFn: createFn,
	}, nil
}

// 获取连接
func (p *BasicConnectionPool) Get() (net.Conn, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil, errors.New("pool is closed")
	}

	// 尝试从池中获取连接
	select {
	case conn := <-p.pool:
		return conn, nil
	default:
		// 池中无连接，创建新连接
		return p.createFn()
	}
}

// 归还连接
func (p *BasicConnectionPool) Put(conn net.Conn) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		conn.Close()
		return
	}

	select {
	case p.pool <- conn:
		// 连接放入池中
	default:
		// 池已满，关闭连接
		conn.Close()
	}
}

// 关闭连接池
func (p *BasicConnectionPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	close(p.pool)

	// 关闭所有连接
	for conn := range p.pool {
		conn.Close()
	}
}
