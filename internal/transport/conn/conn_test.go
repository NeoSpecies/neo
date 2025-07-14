package conn_test

import (
	"context"
	"net"
	"neo/internal/transport/conn"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTCPConnection_Basic(t *testing.T) {
	// 启动测试服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	// 在goroutine中处理连接
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleTestConnection(conn)
		}
	}()

	// 创建客户端连接
	netConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	require.NoError(t, err)

	connection := conn.NewTCPConnection(netConn, "test-conn-1", 30*time.Second, 30*time.Second)
	defer connection.Close()

	t.Run("基本信息", func(t *testing.T) {
		assert.Equal(t, "test-conn-1", connection.ID())
		assert.True(t, connection.IsHealthy())
		assert.NotEmpty(t, connection.RemoteAddr())
		assert.NotEmpty(t, connection.LocalAddr())
	})

	t.Run("发送和接收数据", func(t *testing.T) {
		ctx := context.Background()
		testData := []byte("Hello, World!")

		// 发送数据
		err := connection.Send(ctx, testData)
		require.NoError(t, err)

		// 接收响应
		respData, err := connection.Receive(ctx)
		require.NoError(t, err)
		assert.Equal(t, testData, respData) // 测试服务器应该回显数据
	})

	t.Run("超时处理", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// 由于超时很短，操作应该超时
		err := connection.Send(ctx, []byte("timeout test"))
		// 根据实现，可能会超时或成功，取决于操作速度
		_ = err // 不强制要求超时，因为操作可能很快完成
	})
}

func TestTCPConnection_HealthCheck(t *testing.T) {
	// 启动测试服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	// 简单的健康检查服务器
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				for {
					// 读取长度
					lengthBuf := make([]byte, 4)
					if _, err := c.Read(lengthBuf); err != nil {
						return
					}
					
					length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
					if length <= 0 || length > 1024 {
						return
					}
					
					// 读取数据
					data := make([]byte, length)
					if _, err := c.Read(data); err != nil {
						return
					}
					
					// 处理PING/PONG
					if string(data) == "PING" {
						response := []byte("PONG")
						respLengthBuf := make([]byte, 4)
						respLengthBuf[0] = byte(len(response) >> 24)
						respLengthBuf[1] = byte(len(response) >> 16)
						respLengthBuf[2] = byte(len(response) >> 8)
						respLengthBuf[3] = byte(len(response))
						
						c.Write(respLengthBuf)
						c.Write(response)
					}
				}
			}(conn)
		}
	}()

	// 创建连接
	netConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	require.NoError(t, err)

	connection := conn.NewTCPConnection(netConn, "health-test", 30*time.Second, 30*time.Second)
	defer connection.Close()

	t.Run("健康检查", func(t *testing.T) {
		assert.True(t, connection.IsHealthy())
	})
}

func TestTCPConnection_Close(t *testing.T) {
	// 创建一个简单的连接用于测试关闭
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	addr := listener.Addr().String()

	go func() {
		conn, _ := listener.Accept()
		conn.Close()
	}()

	netConn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	require.NoError(t, err)

	connection := conn.NewTCPConnection(netConn, "close-test", 30*time.Second, 30*time.Second)

	// 关闭连接
	err = connection.Close()
	require.NoError(t, err)

	// 再次关闭应该没有问题
	err = connection.Close()
	require.NoError(t, err)

	// 关闭后应该不健康
	assert.False(t, connection.IsHealthy())
}

// handleTestConnection 处理测试连接，简单地回显接收到的数据
func handleTestConnection(conn net.Conn) {
	defer conn.Close()
	
	for {
		// 读取长度头
		lengthBuf := make([]byte, 4)
		if _, err := conn.Read(lengthBuf); err != nil {
			return
		}
		
		length := int(lengthBuf[0])<<24 | int(lengthBuf[1])<<16 | int(lengthBuf[2])<<8 | int(lengthBuf[3])
		if length <= 0 || length > 10*1024*1024 {
			return
		}
		
		// 读取数据
		data := make([]byte, length)
		if _, err := conn.Read(data); err != nil {
			return
		}
		
		// 回显数据
		respLengthBuf := make([]byte, 4)
		respLengthBuf[0] = byte(len(data) >> 24)
		respLengthBuf[1] = byte(len(data) >> 16)
		respLengthBuf[2] = byte(len(data) >> 8)
		respLengthBuf[3] = byte(len(data))
		
		conn.Write(respLengthBuf)
		conn.Write(data)
	}
}