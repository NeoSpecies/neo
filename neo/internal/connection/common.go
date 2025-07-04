package connection

import (
	"net"
	"time"

	"neo/internal/types"
)

// 保留类型别名，但移除方法定义
// Deprecated: 使用 types.StatsConnection 代替
type StatsConnection = types.StatsConnection

// Deprecated: 使用 types.TimeoutConnection 代替
type TimeoutConnection = types.TimeoutConnection

// Deprecated: 使用 types.ConnectionPool 代替
type ConnectionPool = types.ConnectionPool

// Deprecated: 使用 types.BasicConnectionPool 代替
type BasicConnectionPool = types.BasicConnectionPool

// 创建新的带超时控制的连接
func NewTimeoutConnection(conn net.Conn, readTimeout, writeTimeout time.Duration) *TimeoutConnection {
	return types.NewTimeoutConnection(conn, readTimeout, writeTimeout)
}

// 创建新的基础连接池
func NewBasicConnectionPool(size int, createFn func() (net.Conn, error)) (*BasicConnectionPool, error) {
	return types.NewBasicConnectionPool(size, createFn)
}
