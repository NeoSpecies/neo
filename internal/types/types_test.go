package types_test

import (
	"encoding/json"
	"neo/internal/types"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试 Message 结构体
func TestMessageStruct(t *testing.T) {
	t.Run("NewMessage", func(t *testing.T) {
		msg := types.NewMessage(types.REQUEST, "test.service", "testMethod")
		assert.NotEmpty(t, msg.ID)
		assert.Equal(t, types.REQUEST, msg.Type)
		assert.Equal(t, "test.service", msg.Service)
		assert.Equal(t, "testMethod", msg.Method)
		assert.NotNil(t, msg.Metadata)
		assert.NotZero(t, msg.Timestamp)
	})

	t.Run("Validate", func(t *testing.T) {
		// 有效消息
		msg := types.NewMessage(types.REQUEST, "test.service", "testMethod")
		assert.NoError(t, msg.Validate())

		// 无效ID
		msg.ID = ""
		assert.Equal(t, types.ErrInvalidMessageID, msg.Validate())

		// 无效类型
		msg.ID = "123"
		msg.Type = 0
		assert.Equal(t, types.ErrInvalidMessageType, msg.Validate())

		// 无效服务名
		msg.Type = types.REQUEST
		msg.Service = ""
		assert.Equal(t, types.ErrInvalidService, msg.Validate())
	})

	t.Run("JSON序列化", func(t *testing.T) {
		msg := types.NewMessage(types.REQUEST, "test.service", "testMethod")
		msg.Body = []byte("test body")
		msg.Metadata["key"] = "value"

		// 序列化
		data, err := json.Marshal(msg)
		require.NoError(t, err)

		// 反序列化
		var decoded types.Message
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, msg.ID, decoded.ID)
		assert.Equal(t, msg.Type, decoded.Type)
		assert.Equal(t, msg.Service, decoded.Service)
		assert.Equal(t, msg.Method, decoded.Method)
		assert.Equal(t, msg.Body, decoded.Body)
		assert.Equal(t, msg.Metadata["key"], decoded.Metadata["key"])
	})
}

// 测试 Request 结构体
func TestRequestStruct(t *testing.T) {
	t.Run("NewRequest", func(t *testing.T) {
		body := []byte("test body")
		req := types.NewRequest("test.service", "testMethod", body)
		assert.NotEmpty(t, req.ID)
		assert.Equal(t, "test.service", req.Service)
		assert.Equal(t, "testMethod", req.Method)
		assert.Equal(t, body, req.Body)
		assert.NotNil(t, req.Metadata)
	})

	t.Run("Timeout", func(t *testing.T) {
		req := types.NewRequest("test.service", "testMethod", nil)
		
		// 默认超时
		assert.Equal(t, 30*time.Second, req.GetTimeout())

		// 设置超时
		req.SetTimeout(5 * time.Second)
		assert.Equal(t, 5*time.Second, req.GetTimeout())
	})

	t.Run("Validate", func(t *testing.T) {
		req := types.NewRequest("test.service", "testMethod", nil)
		assert.NoError(t, req.Validate())

		// 无效ID
		req.ID = ""
		assert.Equal(t, types.ErrInvalidRequestID, req.Validate())

		// 无效服务名
		req.ID = "123"
		req.Service = ""
		assert.Equal(t, types.ErrInvalidService, req.Validate())

		// 无效方法名
		req.Service = "test.service"
		req.Method = ""
		assert.Equal(t, types.ErrInvalidMethod, req.Validate())
	})

	t.Run("ToMessage", func(t *testing.T) {
		req := types.NewRequest("test.service", "testMethod", []byte("body"))
		req.Metadata["key"] = "value"

		msg := req.ToMessage()
		assert.Equal(t, req.ID, msg.ID)
		assert.Equal(t, types.REQUEST, msg.Type)
		assert.Equal(t, req.Service, msg.Service)
		assert.Equal(t, req.Method, msg.Method)
		assert.Equal(t, req.Body, msg.Body)
		assert.Equal(t, req.Metadata, msg.Metadata)
	})

	t.Run("JSON序列化", func(t *testing.T) {
		req := types.NewRequest("test.service", "testMethod", []byte("body"))
		req.Metadata["key"] = "value"

		// 序列化
		data, err := json.Marshal(req)
		require.NoError(t, err)

		// 反序列化
		var decoded types.Request
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, req.ID, decoded.ID)
		assert.Equal(t, req.Service, decoded.Service)
		assert.Equal(t, req.Method, decoded.Method)
		assert.Equal(t, req.Body, decoded.Body)
		assert.Equal(t, req.Metadata["key"], decoded.Metadata["key"])
	})
}

// 测试 Response 结构体
func TestResponseStruct(t *testing.T) {
	t.Run("NewResponse", func(t *testing.T) {
		body := []byte("response body")
		resp := types.NewResponse("req123", 200, body)
		assert.Equal(t, "req123", resp.ID)
		assert.Equal(t, 200, resp.Status)
		assert.Equal(t, body, resp.Body)
		assert.Empty(t, resp.Error)
		assert.NotNil(t, resp.Metadata)
	})

	t.Run("NewErrorResponse", func(t *testing.T) {
		resp := types.NewErrorResponse("req123", 500, "internal server error")
		assert.Equal(t, "req123", resp.ID)
		assert.Equal(t, 500, resp.Status)
		assert.Equal(t, "internal server error", resp.Error)
		assert.Nil(t, resp.Body)
	})

	t.Run("IsSuccess", func(t *testing.T) {
		// 成功状态码
		assert.True(t, types.NewResponse("1", 200, nil).IsSuccess())
		assert.True(t, types.NewResponse("1", 201, nil).IsSuccess())
		assert.True(t, types.NewResponse("1", 299, nil).IsSuccess())

		// 失败状态码
		assert.False(t, types.NewResponse("1", 199, nil).IsSuccess())
		assert.False(t, types.NewResponse("1", 300, nil).IsSuccess())
		assert.False(t, types.NewResponse("1", 404, nil).IsSuccess())
		assert.False(t, types.NewResponse("1", 500, nil).IsSuccess())
	})

	t.Run("Validate", func(t *testing.T) {
		resp := types.NewResponse("req123", 200, nil)
		assert.NoError(t, resp.Validate())

		// 无效ID
		resp.ID = ""
		assert.Equal(t, types.ErrInvalidResponseID, resp.Validate())

		// 无效状态码
		resp.ID = "123"
		resp.Status = 0
		assert.Equal(t, types.ErrInvalidStatus, resp.Validate())
	})

	t.Run("ToMessage", func(t *testing.T) {
		resp := types.NewResponse("req123", 200, []byte("body"))
		resp.Metadata["key"] = "value"

		msg := resp.ToMessage()
		assert.Equal(t, resp.ID, msg.ID)
		assert.Equal(t, types.RESPONSE, msg.Type)
		assert.Equal(t, resp.Body, msg.Body)
		assert.Equal(t, resp.Metadata, msg.Metadata)
	})

	t.Run("JSON序列化", func(t *testing.T) {
		resp := types.NewResponse("req123", 200, []byte("body"))
		resp.Metadata["key"] = "value"

		// 序列化
		data, err := json.Marshal(resp)
		require.NoError(t, err)

		// 反序列化
		var decoded types.Response
		err = json.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, resp.ID, decoded.ID)
		assert.Equal(t, resp.Status, decoded.Status)
		assert.Equal(t, resp.Body, decoded.Body)
		assert.Equal(t, resp.Metadata["key"], decoded.Metadata["key"])
	})
}

// 测试 ID 生成
func TestGenerateID(t *testing.T) {
	t.Run("唯一性", func(t *testing.T) {
		ids := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			id := types.GenerateID()
			assert.NotEmpty(t, id)
			assert.False(t, ids[id], "ID重复: %s", id)
			ids[id] = true
		}
	})

	t.Run("并发生成", func(t *testing.T) {
		var wg sync.WaitGroup
		ids := sync.Map{}
		duplicates := 0

		// 启动100个goroutine，每个生成100个ID
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					id := types.GenerateID()
					if _, loaded := ids.LoadOrStore(id, true); loaded {
						duplicates++
					}
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, 0, duplicates, "发现重复ID")
	})
}

// 测试服务名验证
func TestValidateServiceName(t *testing.T) {
	tests := []struct {
		name    string
		service string
		wantErr bool
	}{
		{"有效服务名", "test.service", false},
		{"带下划线", "test_service", false},
		{"带短横线", "test-service", false},
		{"带数字", "service123", false},
		{"大写字母", "TestService", false},
		{"空服务名", "", true},
		{"特殊字符", "test@service", true},
		{"空格", "test service", true},
		{"中文", "测试服务", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateServiceName(tt.service)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// 测试方法名验证
func TestValidateMethodName(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{"有效方法名", "testMethod", false},
		{"带下划线", "test_method", false},
		{"带数字", "method123", false},
		{"大写字母", "TestMethod", false},
		{"空方法名", "", true},
		{"特殊字符", "test-method", true},
		{"带点号", "test.method", true},
		{"空格", "test method", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := types.ValidateMethodName(tt.method)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// 边界测试
func TestBoundaryConditions(t *testing.T) {
	t.Run("超大Body", func(t *testing.T) {
		// 创建1MB的数据
		largeBody := make([]byte, 1024*1024)
		for i := range largeBody {
			largeBody[i] = byte(i % 256)
		}

		req := types.NewRequest("test.service", "testMethod", largeBody)
		assert.Equal(t, len(largeBody), len(req.Body))

		// 测试序列化
		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded types.Request
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, len(largeBody), len(decoded.Body))
	})

	t.Run("大量Metadata", func(t *testing.T) {
		req := types.NewRequest("test.service", "testMethod", nil)
		
		// 添加1000个元数据项
		for i := 0; i < 1000; i++ {
			key := types.GenerateID()
			value := types.GenerateID()
			req.Metadata[key] = value
		}

		assert.Equal(t, 1000, len(req.Metadata))

		// 测试序列化
		data, err := json.Marshal(req)
		assert.NoError(t, err)

		var decoded types.Request
		err = json.Unmarshal(data, &decoded)
		assert.NoError(t, err)
		assert.Equal(t, 1000, len(decoded.Metadata))
	})

	t.Run("空值处理", func(t *testing.T) {
		// 空Body
		req := types.NewRequest("test.service", "testMethod", nil)
		assert.Nil(t, req.Body)

		// 空Metadata在序列化时应该被省略
		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotContains(t, string(data), `"metadata":{}`)
	})
}

// 性能基准测试
func BenchmarkGenerateID(b *testing.B) {
	for i := 0; i < b.N; i++ {
		types.GenerateID()
	}
}

func BenchmarkMessageSerialization(b *testing.B) {
	msg := types.NewMessage(types.REQUEST, "test.service", "testMethod")
	msg.Body = []byte("test body with some content")
	msg.Metadata["key1"] = "value1"
	msg.Metadata["key2"] = "value2"

	b.Run("Marshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			json.Marshal(msg)
		}
	})

	data, _ := json.Marshal(msg)
	b.Run("Unmarshal", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var decoded types.Message
			json.Unmarshal(data, &decoded)
		}
	})
}

func BenchmarkValidateServiceName(b *testing.B) {
	serviceName := "test.service.name"
	for i := 0; i < b.N; i++ {
		types.ValidateServiceName(serviceName)
	}
}