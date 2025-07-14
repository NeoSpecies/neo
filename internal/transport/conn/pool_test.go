package conn_test

import (
	"context"
	"neo/internal/transport/conn"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockConnection 模拟连接
type mockConnection struct {
	id       string
	addr     string
	healthy  bool
	closed   bool
	sendErr  error
	recvErr  error
	recvData []byte
}

func newMockConnection(id, addr string) *mockConnection {
	return &mockConnection{
		id:      id,
		addr:    addr,
		healthy: true,
	}
}

func (m *mockConnection) Send(ctx context.Context, data []byte) error {
	if m.sendErr != nil {
		return m.sendErr
	}
	return nil
}

func (m *mockConnection) Receive(ctx context.Context) ([]byte, error) {
	if m.recvErr != nil {
		return nil, m.recvErr
	}
	return m.recvData, nil
}

func (m *mockConnection) Close() error {
	m.closed = true
	m.healthy = false
	return nil
}

func (m *mockConnection) IsHealthy() bool {
	return m.healthy && !m.closed
}

func (m *mockConnection) RemoteAddr() string {
	return m.addr
}

func (m *mockConnection) LocalAddr() string {
	return "127.0.0.1:0"
}

func (m *mockConnection) ID() string {
	return m.id
}

// mockDialFunc 模拟连接创建函数
func mockDialFunc(ctx context.Context, addr string) (conn.Connection, error) {
	return newMockConnection("mock-"+addr, addr), nil
}

func TestConnectionPool_Basic(t *testing.T) {
	config := &conn.PoolConfig{
		MaxSize:             5,
		MinSize:             2,
		MaxIdleTime:         1 * time.Minute,
		ConnectionTimeout:   5 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		MaxRetries:          3,
	}

	pool := conn.NewConnectionPool(config, mockDialFunc)
	defer pool.Close()

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	t.Run("获取和归还连接", func(t *testing.T) {
		// 获取连接
		connection, err := pool.Get(ctx, addr)
		require.NoError(t, err)
		assert.NotNil(t, connection)
		assert.True(t, connection.IsHealthy())

		// 归还连接
		err = pool.Put(connection)
		require.NoError(t, err)
	})

	t.Run("连接复用", func(t *testing.T) {
		// 获取连接
		conn1, err := pool.Get(ctx, addr)
		require.NoError(t, err)

		// 归还连接
		err = pool.Put(conn1)
		require.NoError(t, err)

		// 再次获取连接，应该复用之前的连接
		conn2, err := pool.Get(ctx, addr)
		require.NoError(t, err)

		// 由于是包装的连接，ID可能不同，但底层连接应该是复用的
		assert.True(t, conn2.IsHealthy())

		pool.Put(conn2)
	})

	t.Run("统计信息", func(t *testing.T) {
		stats := pool.Stats()
		assert.GreaterOrEqual(t, stats.TotalRequests, int64(2))
		assert.GreaterOrEqual(t, stats.TotalHits, int64(1))
	})
}

func TestConnectionPool_MaxSize(t *testing.T) {
	config := &conn.PoolConfig{
		MaxSize:             2,
		MinSize:             1,
		MaxIdleTime:         1 * time.Minute,
		ConnectionTimeout:   1 * time.Second,
		HealthCheckInterval: 100 * time.Millisecond,
		MaxRetries:          3,
	}

	pool := conn.NewConnectionPool(config, mockDialFunc)
	defer pool.Close()

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	// 获取最大数量的连接
	var connections []conn.Connection
	for i := 0; i < config.MaxSize; i++ {
		connection, err := pool.Get(ctx, addr)
		require.NoError(t, err)
		connections = append(connections, connection)
	}

	// 尝试获取超过最大数量的连接，应该等待
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := pool.Get(ctx, addr)
	assert.ErrorIs(t, err, context.DeadlineExceeded)

	// 归还一个连接
	err = pool.Put(connections[0])
	require.NoError(t, err)

	// 现在应该能获取到连接
	ctx = context.Background()
	conn, err := pool.Get(ctx, addr)
	require.NoError(t, err)
	pool.Put(conn)

	// 归还剩余连接
	for i := 1; i < len(connections); i++ {
		pool.Put(connections[i])
	}
}

func TestConnectionPool_HealthCheck(t *testing.T) {
	config := &conn.PoolConfig{
		MaxSize:             10,
		MinSize:             2,
		MaxIdleTime:         50 * time.Millisecond, // 很短的空闲时间
		ConnectionTimeout:   1 * time.Second,
		HealthCheckInterval: 50 * time.Millisecond, // 很短的健康检查间隔
		MaxRetries:          3,
	}

	pool := conn.NewConnectionPool(config, mockDialFunc)
	defer pool.Close()

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	// 获取连接
	connection, err := pool.Get(ctx, addr)
	require.NoError(t, err)

	// 归还连接
	err = pool.Put(connection)
	require.NoError(t, err)

	// 等待超过空闲时间，让健康检查清理连接
	time.Sleep(150 * time.Millisecond)

	// 验证统计信息
	stats := pool.Stats()
	t.Logf("Stats after health check: %+v", stats)
}

func TestConnectionPool_Concurrent(t *testing.T) {
	config := &conn.PoolConfig{
		MaxSize:             20,
		MinSize:             5,
		MaxIdleTime:         1 * time.Minute,
		ConnectionTimeout:   5 * time.Second,
		HealthCheckInterval: 1 * time.Second,
		MaxRetries:          3,
	}

	pool := conn.NewConnectionPool(config, mockDialFunc)
	defer pool.Close()

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	// 并发获取和归还连接
	var wg sync.WaitGroup
	numGoroutines := 50
	numOperations := 10

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < numOperations; j++ {
				// 获取连接
				connection, err := pool.Get(ctx, addr)
				if err != nil {
					t.Errorf("Failed to get connection: %v", err)
					continue
				}

				// 模拟使用连接
				time.Sleep(1 * time.Millisecond)

				// 归还连接
				if err := pool.Put(connection); err != nil {
					t.Errorf("Failed to put connection: %v", err)
				}
			}
		}()
	}

	wg.Wait()

	// 验证统计信息
	stats := pool.Stats()
	expectedRequests := int64(numGoroutines * numOperations)
	assert.Equal(t, expectedRequests, stats.TotalRequests)
	assert.Zero(t, stats.ActiveConnections) // 所有连接都应该归还了
}

func TestConnectionPool_ErrorHandling(t *testing.T) {
	config := conn.DefaultPoolConfig()

	// 创建会失败的dial函数
	failingDialFunc := func(ctx context.Context, addr string) (conn.Connection, error) {
		return nil, assert.AnError
	}

	pool := conn.NewConnectionPool(config, failingDialFunc)
	defer pool.Close()

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	// 尝试获取连接应该失败
	_, err := pool.Get(ctx, addr)
	assert.Error(t, err)

	// 验证错误统计
	stats := pool.Stats()
	assert.Greater(t, stats.TotalErrors, int64(0))
}

func TestConnectionPool_Close(t *testing.T) {
	config := conn.DefaultPoolConfig()
	pool := conn.NewConnectionPool(config, mockDialFunc)

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	// 获取一些连接
	var connections []conn.Connection
	for i := 0; i < 3; i++ {
		connection, err := pool.Get(ctx, addr)
		require.NoError(t, err)
		connections = append(connections, connection)
	}

	// 关闭连接池
	err := pool.Close()
	require.NoError(t, err)

	// 尝试获取连接应该失败
	_, err = pool.Get(ctx, addr)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection pool is closed")

	// 尝试归还连接应该失败
	err = pool.Put(connections[0])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection pool is closed")
}

func TestDefaultDialFunc(t *testing.T) {
	// 启动一个测试服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	// 在goroutine中接受连接
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	ctx := context.Background()

	// 测试默认dial函数
	connection, err := conn.DefaultDialFunc(ctx, addr)
	require.NoError(t, err)
	assert.NotNil(t, connection)
	assert.True(t, connection.IsHealthy())

	connection.Close()
}

func BenchmarkConnectionPool(b *testing.B) {
	config := &conn.PoolConfig{
		MaxSize:             100,
		MinSize:             10,
		MaxIdleTime:         5 * time.Minute,
		ConnectionTimeout:   5 * time.Second,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
	}

	pool := conn.NewConnectionPool(config, mockDialFunc)
	defer pool.Close()

	ctx := context.Background()
	addr := "127.0.0.1:8080"

	b.Run("GetPut", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				connection, err := pool.Get(ctx, addr)
				if err != nil {
					b.Fatal(err)
				}
				pool.Put(connection)
			}
		})
	})

	b.Run("Stats", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			pool.Stats()
		}
	})
}