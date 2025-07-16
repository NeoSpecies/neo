package conn

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// Connection 连接接口
type Connection interface {
	// Send 发送数据
	Send(ctx context.Context, data []byte) error
	// Receive 接收数据
	Receive(ctx context.Context) ([]byte, error)
	// Close 关闭连接
	Close() error
	// IsHealthy 检查连接健康状态
	IsHealthy() bool
	// RemoteAddr 获取远程地址
	RemoteAddr() string
	// LocalAddr 获取本地地址
	LocalAddr() string
	// ID 获取连接ID
	ID() string
}

// TCPConnection TCP连接实现
type TCPConnection struct {
	id                    string
	conn                  net.Conn
	readTimeout           time.Duration
	writeTimeout          time.Duration
	lastActivity          time.Time
	mu                    sync.RWMutex
	closed                bool
	healthChecker         HealthChecker
	activityCheckInterval time.Duration
}

// HealthChecker 健康检查接口
type HealthChecker interface {
	Check(conn Connection) error
}

// DefaultHealthChecker 默认健康检查器
type DefaultHealthChecker struct{}

// Check 执行健康检查
func (d *DefaultHealthChecker) Check(conn Connection) error {
	// 简单的ping检查
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	if err := conn.Send(ctx, []byte("PING")); err != nil {
		return err
	}
	
	data, err := conn.Receive(ctx)
	if err != nil {
		return err
	}
	
	if string(data) != "PONG" {
		return fmt.Errorf("invalid health check response: %s", string(data))
	}
	
	return nil
}

// NewTCPConnection 创建TCP连接
func NewTCPConnection(conn net.Conn, id string, readTimeout, writeTimeout time.Duration) *TCPConnection {
	return &TCPConnection{
		id:                    id,
		conn:                  conn,
		readTimeout:           readTimeout,
		writeTimeout:          writeTimeout,
		lastActivity:          time.Now(),
		healthChecker:         &DefaultHealthChecker{},
		activityCheckInterval: 30 * time.Second, // 默认值
	}
}

// Send 发送数据
func (c *TCPConnection) Send(ctx context.Context, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return fmt.Errorf("connection is closed")
	}
	
	// 设置写超时
	deadline := time.Now().Add(c.writeTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	c.conn.SetWriteDeadline(deadline)
	
	// 写入数据长度（4字节）
	lengthBuf := make([]byte, 4)
	lengthBuf[0] = byte(len(data) >> 24)
	lengthBuf[1] = byte(len(data) >> 16)
	lengthBuf[2] = byte(len(data) >> 8)
	lengthBuf[3] = byte(len(data))
	
	if _, err := c.conn.Write(lengthBuf); err != nil {
		return fmt.Errorf("failed to write length: %w", err)
	}
	
	// 写入数据
	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	
	c.lastActivity = time.Now()
	return nil
}

// Receive 接收数据
func (c *TCPConnection) Receive(ctx context.Context) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil, fmt.Errorf("connection is closed")
	}
	
	// 设置读超时
	deadline := time.Now().Add(c.readTimeout)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	c.conn.SetReadDeadline(deadline)
	
	// 读取长度（4字节）
	lengthBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.conn, lengthBuf); err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}
	
	length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
	if length <= 0 || length > 10*1024*1024 { // 最大10MB
		return nil, fmt.Errorf("invalid message length: %d", length)
	}
	
	// 读取数据
	data := make([]byte, length)
	if _, err := io.ReadFull(c.conn, data); err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	
	c.lastActivity = time.Now()
	return data, nil
}

// Close 关闭连接
func (c *TCPConnection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if c.closed {
		return nil
	}
	
	c.closed = true
	return c.conn.Close()
}

// IsHealthy 检查连接健康状态
func (c *TCPConnection) IsHealthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if c.closed {
		return false
	}
	
	// 检查最后活动时间
	if time.Since(c.lastActivity) > c.activityCheckInterval {
		// 执行健康检查
		if c.healthChecker != nil {
			if err := c.healthChecker.Check(c); err != nil {
				return false
			}
		}
	}
	
	return true
}

// RemoteAddr 获取远程地址
func (c *TCPConnection) RemoteAddr() string {
	if c.conn != nil {
		return c.conn.RemoteAddr().String()
	}
	return ""
}

// LocalAddr 获取本地地址
func (c *TCPConnection) LocalAddr() string {
	if c.conn != nil {
		return c.conn.LocalAddr().String()
	}
	return ""
}

// ID 获取连接ID
func (c *TCPConnection) ID() string {
	return c.id
}

// LastActivity 获取最后活动时间
func (c *TCPConnection) LastActivity() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastActivity
}

// SetHealthChecker 设置健康检查器
func (c *TCPConnection) SetHealthChecker(checker HealthChecker) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthChecker = checker
}

// SetActivityCheckInterval 设置活动检查间隔
func (c *TCPConnection) SetActivityCheckInterval(interval time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.activityCheckInterval = interval
}