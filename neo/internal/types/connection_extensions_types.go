/*
 * 描述: 定义连接扩展类型，包括带统计功能的连接包装器、超时控制连接和基础连接池实现
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"errors"
	"net"
	"sync"
	"time"
)

// StatsConnection 带统计功能的连接包装器
// 包装底层网络连接，提供数据传输统计功能，包括读写字节数和最后活动时间跟踪
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Conn           | net.Conn         | 底层网络连接                      |
// | stats          | *ConnectionStats | 连接统计信息                      |
// | mu             | sync.Mutex       | 保证统计操作并发安全的互斥锁      |
// +----------------+------------------+-----------------------------------+
type StatsConnection struct {
	net.Conn
	stats *ConnectionStats
	mu    sync.Mutex
}

// NewStatsConnection 创建新的带统计功能的连接
// 初始化并返回一个包装了指定net.Conn的StatsConnection实例
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | conn           | net.Conn         | 要包装的底层网络连接              |
// +----------------+------------------+-----------------------------------+
// | 返回值         | *StatsConnection | 初始化后的带统计功能的连接实例    |
// +----------------+------------------+-----------------------------------+
func NewStatsConnection(conn net.Conn) *StatsConnection {
	return &StatsConnection{
		Conn:  conn,
		stats: NewConnectionStats(),
	}
}

// Read 读取数据并更新统计信息
// 重写底层Conn的Read方法，读取数据后更新读取字节数和最后活动时间
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | b              | []byte           | 数据接收缓冲区                    |
// +----------------+------------------+-----------------------------------+
// | 返回值         | int              | 读取的字节数                      |
// | 返回值         | error            | 读取过程中发生的错误              |
// +----------------+------------------+-----------------------------------+
func (c *StatsConnection) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if n > 0 {
		c.mu.Lock()
		c.stats.BytesRead += uint64(n) // 已与stats.go中的uint64类型匹配
		c.stats.LastActive = time.Now()
		c.mu.Unlock()
		// metrics.RecordMessageSize("connection", "read", "bytes", int64(n)) 需根据实际metrics包位置调整
	}
	return n, err
}

// Write 写入数据并更新统计信息
// 重写底层Conn的Write方法，写入数据后更新写入字节数和最后活动时间
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | b              | []byte           | 要写入的数据                      |
// +----------------+------------------+-----------------------------------+
// | 返回值         | int              | 写入的字节数                      |
// | 返回值         | error            | 写入过程中发生的错误              |
// +----------------+------------------+-----------------------------------+
func (c *StatsConnection) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if n > 0 {
		c.mu.Lock()
		c.stats.BytesWritten += uint64(n) // 已与stats.go中的uint64类型匹配
		c.stats.LastActive = time.Now()
		c.mu.Unlock()
		// metrics.RecordMessageSize("connection", "write", "bytes", int64(n)) 需根据实际metrics包位置调整
	}
	return n, err
}

// TimeoutConnection 带超时控制的连接
// 包装底层网络连接，提供读写超时控制功能
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Conn           | net.Conn         | 底层网络连接                      |
// | readTimeout    | time.Duration    | 读取超时时间                      |
// | writeTimeout   | time.Duration    | 写入超时时间                      |
// +----------------+------------------+-----------------------------------+
type TimeoutConnection struct {
	net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// NewTimeoutConnection 创建新的带超时控制的连接
// 初始化并返回一个包装了指定net.Conn的TimeoutConnection实例
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | conn           | net.Conn         | 要包装的底层网络连接              |
// | readTimeout    | time.Duration    | 读取超时时间                      |
// | writeTimeout   | time.Duration    | 写入超时时间                      |
// +----------------+------------------+-----------------------------------+
// | 返回值         | *TimeoutConnection | 初始化后的带超时控制的连接实例   |
// +----------------+------------------+-----------------------------------+
func NewTimeoutConnection(conn net.Conn, readTimeout, writeTimeout time.Duration) *TimeoutConnection {
	return &TimeoutConnection{
		Conn:         conn,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// Read 带超时控制的读取操作
// 设置读取超时后执行底层连接的Read操作
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | b              | []byte           | 数据接收缓冲区                    |
// +----------------+------------------+-----------------------------------+
// | 返回值         | int              | 读取的字节数                      |
// | 返回值         | error            | 读取过程中发生的错误              |
// +----------------+------------------+-----------------------------------+
func (c *TimeoutConnection) Read(b []byte) (int, error) {
	if c.readTimeout > 0 {
		if err := c.Conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Read(b)
}

// Write 带超时控制的写入操作
// 设置写入超时后执行底层连接的Write操作
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | b              | []byte           | 要写入的数据                      |
// +----------------+------------------+-----------------------------------+
// | 返回值         | int              | 写入的字节数                      |
// | 返回值         | error            | 写入过程中发生的错误              |
// +----------------+------------------+-----------------------------------+
func (c *TimeoutConnection) Write(b []byte) (int, error) {
	if c.writeTimeout > 0 {
		if err := c.Conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
			return 0, err
		}
	}
	return c.Conn.Write(b)
}

// SetReadTimeout 设置读取超时时间
// 更新连接的读取超时时间配置
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | timeout        | time.Duration    | 新的读取超时时间                  |
// +----------------+------------------+-----------------------------------+
func (c *TimeoutConnection) SetReadTimeout(timeout time.Duration) {
	c.readTimeout = timeout
}

// SetWriteTimeout 设置写入超时时间
// 更新连接的写入超时时间配置
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | timeout        | time.Duration    | 新的写入超时时间                  |
// +----------------+------------------+-----------------------------------+
func (c *TimeoutConnection) SetWriteTimeout(timeout time.Duration) {
	c.writeTimeout = timeout
}

// ConnectionPool 连接池接口
// 定义连接池的基本操作规范
// +----------------+------------------+-----------------------------------+
// | 方法名         | 参数             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Get            | 无               | 获取一个连接                      |
// | Put            | conn net.Conn    | 归还一个连接                      |
// | Close          | 无               | 关闭连接池并释放资源              |
// +----------------+------------------+-----------------------------------+
type ConnectionPool interface {
	Get() (net.Conn, error)
	Put(conn net.Conn)
	Close()
}

// BasicConnectionPool 基础连接池实现
// 基于通道实现的简单连接池，提供连接的获取、归还和关闭功能
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | pool           | chan net.Conn    | 存储连接的通道                    |
// | mu             | sync.Mutex       | 保证并发安全的互斥锁              |
// | closed         | bool             | 连接池是否已关闭的标志            |
// | createFn       | func() (net.Conn, error) | 创建新连接的工厂函数        |
// +----------------+------------------+-----------------------------------+
type BasicConnectionPool struct {
	pool     chan net.Conn
	mu       sync.Mutex
	closed   bool
	createFn func() (net.Conn, error)
}

// NewBasicConnectionPool 创建新的基础连接池
// 初始化并返回一个指定大小的BasicConnectionPool实例
// +----------------+------------------+-----------------------------------+
// | 参数           | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | size           | int              | 连接池容量                        |
// | createFn       | func() (net.Conn, error) | 创建新连接的工厂函数        |
// +----------------+------------------+-----------------------------------+
// | 返回值         | *BasicConnectionPool | 初始化后的连接池实例          |
// | 返回值         | error            | 初始化过程中发生的错误            |
// +----------------+------------------+-----------------------------------+
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
