package connection

import (
	"context"
	"errors"
	"neo/internal/config"
	"neo/internal/types"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// 连接池错误定义
var (
	ErrPoolClosed     = errors.New("connection pool is closed")
	ErrPoolExhausted  = errors.New("connection pool exhausted")
	ErrInvalidConfig  = errors.New("invalid pool configuration")
	ErrConnectionFail = errors.New("failed to create connection")
)

// NewTCPConnectionPool 创建新的TCP连接池
func NewTCPConnectionPool(cfg types.Config, factory func() (net.Conn, error), balancer types.Balancer) (*types.TCPConnectionPool, error) {
	// 验证配置
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	// 初始化连接池
	pool := &types.TCPConnectionPool{
		Config:            cfg,
		Factory:           factory,
		Balancer:          balancer,
		Connections:       make([]*types.Connection, 0, cfg.MaxSize),
		Mu:                &sync.RWMutex{},
		Done:              make(chan struct{}),
		WaitConn:          make(chan struct{}, cfg.MaxSize),
		MaxSize:           cfg.MaxSize,
		MinSize:           cfg.MinSize,
		InitialSize:       cfg.InitialSize,
		IdleTimeout:       cfg.IdleTimeout,
		KeepAliveInterval: cfg.KeepAliveInterval,
	}

	// 初始化指标收集器
	if cfg.MaxSize > 0 {
		registry := prometheus.NewRegistry()
		pool.Metrics = types.NewMetrics(registry)
	}

	// 预创建初始连接
	if err := createInitialConnections(pool); err != nil {
		ClosePool(pool)
		return nil, err
	}

	// 启动维护协程
	go maintain(pool)

	return pool, nil
}

// validateConfig 验证连接池配置
func validateConfig(cfg types.Config) error {
	if cfg.MaxSize <= 0 {
		return ErrInvalidConfig
	}
	if cfg.MinSize < 0 || cfg.MinSize > cfg.MaxSize {
		return ErrInvalidConfig
	}
	if cfg.InitialSize < 0 || cfg.InitialSize > cfg.MaxSize {
		return ErrInvalidConfig
	}
	if cfg.IdleTimeout < 0 {
		return ErrInvalidConfig
	}
	if cfg.KeepAliveInterval < 0 {
		return ErrInvalidConfig
	}
	return nil
}

// createInitialConnections 创建初始连接
func createInitialConnections(pool *types.TCPConnectionPool) error {
	initialSize := pool.InitialSize
	if initialSize == 0 {
		initialSize = pool.MinSize
	}

	var wg sync.WaitGroup
	errCh := make(chan error, initialSize)

	for i := 0; i < initialSize; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := createConnection(pool)
			if err != nil {
				errCh <- err
				return
			}

			pool.Mu.Lock()
			pool.Connections = append(pool.Connections, conn)
			pool.Mu.Unlock()
		}()
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

// createConnection 创建新连接
func createConnection(pool *types.TCPConnectionPool) (*types.Connection, error) {
	// 重试机制
	var conn net.Conn
	var err error
	maxRetry := pool.Config.MaxRetryCount
	retryInterval := pool.Config.RetryInterval

	for i := 0; i <= maxRetry; i++ {
		conn, err = pool.Factory()
		if err == nil {
			break
		}

		if i < maxRetry {
			time.Sleep(retryInterval)
		}
	}

	if err != nil {
		return nil, ErrConnectionFail
	}

	// 创建连接统计信息
	stats := types.NewConnectionStats()

	return &types.Connection{
		Conn:      conn,
		Pool:      pool,
		Stats:     stats,
		LastUsed:  time.Now(),
		LastCheck: time.Now(),
	}, nil
}

// getAvailableConnection 获取可用连接
func getAvailableConnection(pool *types.TCPConnectionPool) (*types.Connection, error) {
	pool.Mu.RLock()
	defer pool.Mu.RUnlock()

	// 使用负载均衡器选择连接
	if pool.Balancer != nil {
		return pool.Balancer.Select(pool.Connections)
	}

	// 如果没有负载均衡器，简单返回第一个可用连接
	for _, conn := range pool.Connections {
		if !conn.InUse && !conn.Closed && time.Since(conn.LastUsed) < pool.IdleTimeout {
			conn.InUse = true
			return conn, nil
		}
	}

	return nil, ErrPoolExhausted
}

// Acquire 获取连接
func Acquire(pool *types.TCPConnectionPool) (*types.Connection, error) {
	if pool.Closed {
		return nil, ErrPoolClosed
	}

	for {
		// 尝试获取可用连接
		conn, err := getAvailableConnection(pool)
		if err == nil {
			return conn, nil
		}

		// 连接池耗尽，检查是否可以创建新连接
		pool.Mu.RLock()
		currentSize := len(pool.Connections)
		pool.Mu.RUnlock()

		if currentSize < pool.MaxSize {
			// 创建新连接
			newConn, err := createConnection(pool)
			if err != nil {
				return nil, err
			}

			pool.Mu.Lock()
			pool.Connections = append(pool.Connections, newConn)
			pool.Mu.Unlock()

			return newConn, nil
		}

		// 等待可用连接
		select {
		case <-pool.Done:
			return nil, ErrPoolClosed
		case <-pool.WaitConn:
			// 重试获取连接
		}
	}
}

// Release 释放连接
func Release(pool *types.TCPConnectionPool, conn *types.Connection, err error) {
	if pool.Closed {
		conn.Conn.Close()
		return
	}

	// 更新连接状态
	conn.LastUsed = time.Now()
	conn.InUse = false

	// 处理错误
	if err != nil {
		conn.ErrorCount++

		// 如果错误次数超过阈值，关闭并替换连接
		if conn.ErrorCount >= pool.Config.MaxErrorCount {
			removeConnection(pool, conn)
			newConn, err := createConnection(pool)
			if err != nil {
				return
			}

			pool.Mu.Lock()
			pool.Connections = append(pool.Connections, newConn)
			pool.Mu.Unlock()
		}
	} else {
		conn.ErrorCount = 0
	}

	// 通知等待的协程
	select {
	case pool.WaitConn <- struct{}{}:
	default:
	}
}

// removeConnection 从连接池中移除连接
func removeConnection(pool *types.TCPConnectionPool, conn *types.Connection) {
	pool.Mu.Lock()
	defer pool.Mu.Unlock()

	for i, c := range pool.Connections {
		if c == conn {
			// 关闭连接
			c.Conn.Close()
			c.Closed = true
			// 从切片中移除
			pool.Connections = append(pool.Connections[:i], pool.Connections[i+1:]...)
			break
		}
	}
}

// ClosePool 关闭连接池
func ClosePool(pool *types.TCPConnectionPool) {
	pool.Mu.Lock()
	defer pool.Mu.Unlock()

	if pool.Closed {
		return
	}

	// 关闭通道
	close(pool.Done)
	pool.Closed = true

	// 关闭所有连接
	for _, conn := range pool.Connections {
		conn.Conn.Close()
		conn.Closed = true
	}
	pool.Connections = nil

	// 停止指标收集
	if pool.Metrics != nil {
		pool.Metrics.Close(context.Background()) // 添加context参数
	}
}

// maintain 维护连接池
func maintain(pool *types.TCPConnectionPool) {
	// 健康检查和空闲连接清理间隔
	checkInterval := pool.KeepAliveInterval
	if checkInterval <= 0 {
		checkInterval = 30 * time.Second
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pool.Done:
			return
		case <-ticker.C:
			// 清理空闲连接
			cleanIdleConnections(pool)

			// 健康检查
			if pool.Config.HealthCheck {
				performHealthCheck(pool)
			}

			// 自动扩缩容
			if pool.Config.AutoScaling {
				autoScale(pool)
			}
		}
	}
}

// cleanIdleConnections 清理空闲连接
func cleanIdleConnections(pool *types.TCPConnectionPool) {
	pool.Mu.Lock()
	defer pool.Mu.Unlock()

	var activeConnections []*types.Connection
	for _, conn := range pool.Connections {
		if conn.InUse || time.Since(conn.LastUsed) < pool.IdleTimeout || len(pool.Connections) <= pool.MinSize {
			activeConnections = append(activeConnections, conn)
		} else {
			conn.Conn.Close()
			conn.Closed = true
		}
	}

	pool.Connections = activeConnections
}

// performHealthCheck 执行健康检查
func performHealthCheck(pool *types.TCPConnectionPool) {
	// 实现健康检查逻辑
}

// autoScale 自动扩缩容
func autoScale(pool *types.TCPConnectionPool) {
	// 实现自动扩缩容逻辑
}

func InitPool() *types.TCPConnectionPool {
	cfg := config.GetGlobalConfig() // 使用正确的配置获取函数
	maxConns := cfg.IPC.MaxConnections
	addr := cfg.IPC.Host + ":" + strconv.Itoa(cfg.IPC.Port)
	return types.NewTCPConnectionPool(addr, maxConns)
}
