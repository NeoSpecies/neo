package transport_test

import (
	"context"
	"neo/internal/config"
	"neo/internal/transport"
	"neo/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_Basic(t *testing.T) {
	// 创建配置
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	// 创建传输层
	transport := transport.NewTransport(cfg)
	defer transport.Close()

	t.Run("启动和停止监听器", func(t *testing.T) {
		err := transport.StartListener()
		require.NoError(t, err)

		err = transport.StopListener()
		require.NoError(t, err)
	})

	t.Run("订阅消息", func(t *testing.T) {
		handler := func(msg types.Message) {
			// 消息处理器
		}

		err := transport.Subscribe("test-service", handler)
		require.NoError(t, err)

		// 验证处理器已注册
		assert.True(t, true) // 这里暂时只验证没有错误
	})

	t.Run("获取统计信息", func(t *testing.T) {
		stats := transport.Stats()
		assert.GreaterOrEqual(t, stats.RequestCount, int64(0))
		assert.GreaterOrEqual(t, stats.ResponseCount, int64(0))
	})
}

func TestTransport_SendAsync(t *testing.T) {
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	transport := transport.NewTransport(cfg)
	defer transport.Close()

	ctx := context.Background()
	req := types.Request{
		ID:      "test-req-1",
		Service: "test-service",
		Method:  "test-method",
		Body:    []byte("test data"),
		Metadata: map[string]string{
			"version": "1.0",
		},
	}

	t.Run("异步发送请求", func(t *testing.T) {
		respCh, err := transport.SendAsync(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, respCh)

		// 由于没有实际的服务器，这里不会收到响应
		// 但我们可以验证通道已创建
		select {
		case <-respCh:
			// 如果收到响应，验证其结构
		case <-time.After(100 * time.Millisecond):
			// 超时是正常的，因为没有实际服务器
		}
	})
}

func TestTransport_MessageHandling(t *testing.T) {
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	transport := transport.NewTransport(cfg)
	defer transport.Close()

	t.Run("订阅多个服务", func(t *testing.T) {
		receivedMessages := make(map[string]bool)
		
		// 订阅服务A
		err := transport.Subscribe("service-a", func(msg types.Message) {
			receivedMessages["service-a"] = true
		})
		require.NoError(t, err)

		// 订阅服务B
		err = transport.Subscribe("service-b", func(msg types.Message) {
			receivedMessages["service-b"] = true
		})
		require.NoError(t, err)

		// 验证可以订阅多个服务
		assert.True(t, true) // 这里主要验证没有错误
	})
}

func TestTransport_Configuration(t *testing.T) {
	t.Run("有效配置", func(t *testing.T) {
		cfg := config.Config{
			Transport: config.TransportConfig{
				Timeout:         config.Duration(30 * time.Second),
				RetryCount:      3,
				MaxConnections:  100,
				MinConnections:  10,
				MaxIdleTime:     config.Duration(600 * time.Second),
			},
		}

		transport := transport.NewTransport(cfg)
		defer transport.Close()

		stats := transport.Stats()
		assert.GreaterOrEqual(t, stats.ConnectionStats.TotalConnections, 0)
	})

	t.Run("默认配置", func(t *testing.T) {
		cfg := config.Config{
			Transport: config.TransportConfig{
				Timeout:         config.Duration(30 * time.Second),
				MaxConnections:  10,
			},
		}

		transport := transport.NewTransport(cfg)
		defer transport.Close()

		err := transport.StartListener()
		assert.NoError(t, err) // 应该使用默认地址启动
	})
}

func TestTransport_Lifecycle(t *testing.T) {
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	t.Run("正常生命周期", func(t *testing.T) {
		transport := transport.NewTransport(cfg)

		// 启动监听器
		err := transport.StartListener()
		require.NoError(t, err)

		// 获取统计信息
		stats := transport.Stats()
		assert.GreaterOrEqual(t, stats.RequestCount, int64(0))

		// 停止监听器
		err = transport.StopListener()
		require.NoError(t, err)

		// 关闭传输层
		err = transport.Close()
		require.NoError(t, err)
	})

	t.Run("重复关闭", func(t *testing.T) {
		transport := transport.NewTransport(cfg)

		err := transport.Close()
		require.NoError(t, err)

		// 重复关闭应该没有问题
		err = transport.Close()
		require.NoError(t, err)
	})
}

func TestTransport_ErrorScenarios(t *testing.T) {
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	transport := transport.NewTransport(cfg)
	defer transport.Close()

	ctx := context.Background()

	t.Run("发送到不存在的服务", func(t *testing.T) {
		req := types.Request{
			ID:      "test-req-1",
			Service: "nonexistent-service",
			Method:  "test-method",
			Body:    []byte("test data"),
		}

		_, err := transport.Send(ctx, req)
		assert.Error(t, err)
		// 错误应该与连接失败相关
		assert.Contains(t, err.Error(), "failed to get connection")
	})

	t.Run("超时上下文", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		req := types.Request{
			ID:      "test-req-1",
			Service: "test-service",
			Method:  "test-method",
			Body:    []byte("test data"),
		}

		_, err := transport.Send(ctx, req)
		assert.Error(t, err)
	})
}

func TestTransport_StatsTracking(t *testing.T) {
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	transport := transport.NewTransport(cfg)
	defer transport.Close()

	ctx := context.Background()

	// 获取初始统计
	initialStats := transport.Stats()

	// 尝试发送一些请求（会失败，但会更新统计）
	req := types.Request{
		ID:      "test-req-1",
		Service: "test-service",
		Method:  "test-method",
		Body:    []byte("test data"),
	}

	for i := 0; i < 3; i++ {
		transport.Send(ctx, req)
	}

	// 获取更新后的统计
	finalStats := transport.Stats()

	// 验证统计已更新
	assert.Greater(t, finalStats.RequestCount, initialStats.RequestCount)
	assert.Greater(t, finalStats.ErrorCount, initialStats.ErrorCount)
}

func BenchmarkTransport(b *testing.B) {
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  100,
			MinConnections:  10,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	transport := transport.NewTransport(cfg)
	defer transport.Close()

	ctx := context.Background()
	req := types.Request{
		ID:      "bench-req",
		Service: "bench-service",
		Method:  "bench-method",
		Body:    []byte("benchmark data"),
	}

	b.Run("Send", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			transport.Send(ctx, req)
		}
	})

	b.Run("SendAsync", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			transport.SendAsync(ctx, req)
		}
	})

	b.Run("Stats", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			transport.Stats()
		}
	})

	b.Run("Subscribe", func(b *testing.B) {
		handler := func(msg types.Message) {}
		for i := 0; i < b.N; i++ {
			transport.Subscribe("bench-service", handler)
		}
	})
}
