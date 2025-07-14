package protocol_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"neo/internal/protocol"
	"neo/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试协议工厂
func TestNewCodec(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		wantErr  bool
	}{
		{"HTTP协议", "http", false},
		{"IPC协议", "ipc", false},
		{"不支持的协议", "unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			codec, err := protocol.NewCodec(tt.protocol)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, codec)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, codec)
			}
		})
	}
}

// 测试HTTP编解码
func TestHTTPCodec(t *testing.T) {
	codec := protocol.NewHTTPCodec()

	t.Run("版本", func(t *testing.T) {
		assert.Equal(t, "1.0", codec.Version())
	})

	t.Run("编解码对称性", func(t *testing.T) {
		msg := types.Message{
			ID:      "test-123",
			Type:    types.REQUEST,
			Service: "test.service",
			Method:  "TestMethod",
			Metadata: map[string]string{
				"trace-id": "abc123",
				"user-id":  "user456",
			},
			Body:      []byte(`{"name":"test","value":123}`),
			Timestamp: time.Now(),
		}

		// 编码
		encoded, err := codec.Encode(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		// 验证是有效的JSON
		assert.True(t, json.Valid(encoded))

		// 解码
		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		// 验证字段
		assert.Equal(t, msg.ID, decoded.ID)
		assert.Equal(t, msg.Type, decoded.Type)
		assert.Equal(t, msg.Service, decoded.Service)
		assert.Equal(t, msg.Method, decoded.Method)
		assert.Equal(t, msg.Metadata, decoded.Metadata)
		assert.JSONEq(t, string(msg.Body), string(decoded.Body))
	})

	t.Run("空Body处理", func(t *testing.T) {
		msg := types.Message{
			ID:      "test-empty",
			Type:    types.RESPONSE,
			Service: "test.service",
			Method:  "EmptyMethod",
		}

		encoded, err := codec.Encode(msg)
		require.NoError(t, err)

		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		assert.Empty(t, decoded.Body)
	})

	t.Run("非JSON Body处理", func(t *testing.T) {
		msg := types.Message{
			ID:      "test-text",
			Type:    types.REQUEST,
			Service: "test.service",
			Method:  "TextMethod",
			Body:    []byte("plain text content"),
		}

		encoded, err := codec.Encode(msg)
		require.NoError(t, err)

		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		// 非JSON内容会被编码为JSON字符串
		var decodedText string
		err = json.Unmarshal(decoded.Body, &decodedText)
		require.NoError(t, err)
		assert.Equal(t, "plain text content", decodedText)
	})

	t.Run("消息类型转换", func(t *testing.T) {
		msgTypes := []types.MessageType{
			types.REQUEST,
			types.RESPONSE,
			types.REGISTER,
			types.HEARTBEAT,
		}

		for _, msgType := range msgTypes {
			msg := types.Message{
				ID:   "test-type",
				Type: msgType,
			}

			encoded, err := codec.Encode(msg)
			require.NoError(t, err)

			decoded, err := codec.Decode(encoded)
			require.NoError(t, err)

			assert.Equal(t, msgType, decoded.Type)
		}
	})

	t.Run("错误数据处理", func(t *testing.T) {
		// 无效JSON
		_, err := codec.Decode([]byte("invalid json"))
		assert.Error(t, err)

		// 空数据
		_, err = codec.Decode([]byte{})
		assert.Error(t, err)
	})
}

// 测试IPC编解码
func TestIPCCodec(t *testing.T) {
	codec := protocol.NewIPCCodec()

	t.Run("版本", func(t *testing.T) {
		assert.Equal(t, "1.0", codec.Version())
	})

	t.Run("编解码对称性", func(t *testing.T) {
		msg := types.Message{
			ID:      "test-ipc-123",
			Type:    types.REQUEST,
			Service: "test.ipc.service",
			Method:  "TestIPCMethod",
			Metadata: map[string]string{
				"session-id": "sess123",
				"version":    "1.0",
			},
			Body:      []byte("binary test data"),
			Timestamp: time.Now(),
		}

		// 编码
		encoded, err := codec.Encode(msg)
		require.NoError(t, err)
		assert.NotEmpty(t, encoded)

		// 验证长度字段
		assert.True(t, len(encoded) > 4)

		// 解码
		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		// 验证字段
		assert.Equal(t, msg.ID, decoded.ID)
		assert.Equal(t, msg.Type, decoded.Type)
		assert.Equal(t, msg.Service, decoded.Service)
		assert.Equal(t, msg.Method, decoded.Method)
		assert.Equal(t, msg.Metadata, decoded.Metadata)
		assert.Equal(t, msg.Body, decoded.Body)
	})

	t.Run("空字段处理", func(t *testing.T) {
		msg := types.Message{
			ID:   "",
			Type: types.HEARTBEAT,
		}

		encoded, err := codec.Encode(msg)
		require.NoError(t, err)

		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		assert.Equal(t, msg.ID, decoded.ID)
		assert.Equal(t, msg.Type, decoded.Type)
		assert.Empty(t, decoded.Service)
		assert.Empty(t, decoded.Method)
		assert.Empty(t, decoded.Metadata)
		assert.Empty(t, decoded.Body)
	})

	t.Run("大消息处理", func(t *testing.T) {
		// 创建1MB的数据
		largeData := make([]byte, 1024*1024)
		for i := range largeData {
			largeData[i] = byte(i % 256)
		}

		msg := types.Message{
			ID:      "large-msg",
			Type:    types.REQUEST,
			Service: "large.service",
			Method:  "LargeMethod",
			Body:    largeData,
		}

		encoded, err := codec.Encode(msg)
		require.NoError(t, err)

		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		assert.Equal(t, msg.Body, decoded.Body)
	})

	t.Run("错误数据处理", func(t *testing.T) {
		// 数据太短
		_, err := codec.Decode([]byte{1, 2, 3})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too short")

		// 长度字段错误
		data := make([]byte, 4)
		data[0] = 0xFF // 设置一个很大的长度
		data[1] = 0xFF
		data[2] = 0xFF
		data[3] = 0xFF
		_, err = codec.Decode(data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too large")

		// 不完整的消息
		incompleteData := make([]byte, 10)
		incompleteData[3] = 100 // 声称有100字节，但实际只有6字节
		_, err = codec.Decode(incompleteData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "incomplete")
	})

	t.Run("字符串长度限制", func(t *testing.T) {
		// 创建一个超长的ID（超过65535字节）
		longID := string(make([]byte, 65536))
		msg := types.Message{
			ID:   longID,
			Type: types.REQUEST,
		}

		_, err := codec.Encode(msg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "string too long")
	})

	t.Run("元数据处理", func(t *testing.T) {
		// 大量元数据
		metadata := make(map[string]string)
		for i := 0; i < 100; i++ {
			metadata[fmt.Sprintf("%c%d", 'a'+i%26, i)] = fmt.Sprintf("value%d", i)
		}

		msg := types.Message{
			ID:       "metadata-test",
			Type:     types.REQUEST,
			Metadata: metadata,
		}

		encoded, err := codec.Encode(msg)
		require.NoError(t, err)

		decoded, err := codec.Decode(encoded)
		require.NoError(t, err)

		assert.Equal(t, len(metadata), len(decoded.Metadata))
		for k, v := range metadata {
			assert.Equal(t, v, decoded.Metadata[k])
		}
	})
}

// 测试协议版本兼容性
func TestProtocolVersionCompatibility(t *testing.T) {
	httpCodec := protocol.NewHTTPCodec()
	ipcCodec := protocol.NewIPCCodec()

	t.Run("HTTP版本", func(t *testing.T) {
		assert.Equal(t, "1.0", httpCodec.Version())
	})

	t.Run("IPC版本", func(t *testing.T) {
		assert.Equal(t, "1.0", ipcCodec.Version())
	})
}

// 性能基准测试
func BenchmarkHTTPCodec(b *testing.B) {
	codec := protocol.NewHTTPCodec()
	msg := types.Message{
		ID:      "bench-123",
		Type:    types.REQUEST,
		Service: "bench.service",
		Method:  "BenchMethod",
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Body:      []byte(`{"test":"data","number":123}`),
		Timestamp: time.Now(),
	}

	b.Run("Encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = codec.Encode(msg)
		}
	})

	encoded, _ := codec.Encode(msg)
	b.Run("Decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = codec.Decode(encoded)
		}
	})

	b.Run("EncodeDecode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, _ := codec.Encode(msg)
			_, _ = codec.Decode(encoded)
		}
	})
}

func BenchmarkIPCCodec(b *testing.B) {
	codec := protocol.NewIPCCodec()
	msg := types.Message{
		ID:      "bench-ipc-123",
		Type:    types.REQUEST,
		Service: "bench.ipc.service",
		Method:  "BenchIPCMethod",
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Body:      []byte("binary benchmark data"),
		Timestamp: time.Now(),
	}

	b.Run("Encode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = codec.Encode(msg)
		}
	})

	encoded, _ := codec.Encode(msg)
	b.Run("Decode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = codec.Decode(encoded)
		}
	})

	b.Run("EncodeDecode", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encoded, _ := codec.Encode(msg)
			_, _ = codec.Decode(encoded)
		}
	})
}

// 测试并发安全性
func TestCodecConcurrency(t *testing.T) {
	codecs := []protocol.Codec{
		protocol.NewHTTPCodec(),
		protocol.NewIPCCodec(),
	}

	for _, codec := range codecs {
		t.Run(codec.Version(), func(t *testing.T) {
			msg := types.Message{
				ID:      "concurrent-test",
				Type:    types.REQUEST,
				Service: "concurrent.service",
				Method:  "ConcurrentMethod",
				Body:    []byte("test data"),
			}

			// 并发编解码
			done := make(chan bool, 100)
			for i := 0; i < 100; i++ {
				go func(id int) {
					defer func() { done <- true }()

					// 创建唯一消息
					testMsg := msg
					testMsg.ID = fmt.Sprintf("concurrent-%d", id)

					// 编码
					encoded, err := codec.Encode(testMsg)
					if err != nil {
						t.Errorf("encode error: %v", err)
						return
					}

					// 解码
					decoded, err := codec.Decode(encoded)
					if err != nil {
						t.Errorf("decode error: %v", err)
						return
					}

					// 验证
					if decoded.ID != testMsg.ID {
						t.Errorf("ID mismatch: expected %s, got %s", testMsg.ID, decoded.ID)
					}
				}(i)
			}

			// 等待所有goroutine完成
			for i := 0; i < 100; i++ {
				<-done
			}
		})
	}
}

// 辅助函数：比较两个Message是否相等（忽略Timestamp）
func messagesEqual(t *testing.T, expected, actual types.Message) {
	assert.Equal(t, expected.ID, actual.ID)
	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Service, actual.Service)
	assert.Equal(t, expected.Method, actual.Method)
	assert.Equal(t, expected.Metadata, actual.Metadata)
	assert.True(t, bytes.Equal(expected.Body, actual.Body))
}