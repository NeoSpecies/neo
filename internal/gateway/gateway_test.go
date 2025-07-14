package gateway_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"neo/internal/core"
	"neo/internal/gateway"
	"neo/internal/registry"
	"neo/internal/types"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockService 模拟服务
type mockService struct {
	handleFunc func(ctx context.Context, req types.Request) (types.Response, error)
}

func (m *mockService) Name() string {
	return "mock-service"
}

func (m *mockService) HandleRequest(ctx context.Context, req types.Request) (types.Response, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, req)
	}
	return types.Response{
		ID:     req.ID,
		Status: 200,
		Body:   []byte(`{"message": "success"}`),
	}, nil
}

func (m *mockService) Middleware() []core.Middleware {
	return nil
}

func (m *mockService) Close() error {
	return nil
}

// mockRegistry 模拟注册中心
type mockRegistry struct{}

func (m *mockRegistry) Register(ctx context.Context, instance *registry.ServiceInstance) error {
	return nil
}

func (m *mockRegistry) Deregister(ctx context.Context, instanceID string) error {
	return nil
}

func (m *mockRegistry) Discover(ctx context.Context, serviceName string) ([]*registry.ServiceInstance, error) {
	return []*registry.ServiceInstance{
		{
			ID:      "test-instance-1",
			Name:    serviceName,
			Address: "127.0.0.1",
			Port:    8080,
		},
	}, nil
}

func (m *mockRegistry) Watch(ctx context.Context, serviceName string) (<-chan registry.ServiceEvent, error) {
	ch := make(chan registry.ServiceEvent)
	close(ch)
	return ch, nil
}

func (m *mockRegistry) HealthCheck(ctx context.Context, instanceID string) error {
	return nil
}

func (m *mockRegistry) UpdateInstance(ctx context.Context, instance *registry.ServiceInstance) error {
	return nil
}

func (m *mockRegistry) GetInstance(ctx context.Context, instanceID string) (*registry.ServiceInstance, error) {
	return &registry.ServiceInstance{
		ID:      instanceID,
		Name:    "test-service",
		Address: "127.0.0.1",
		Port:    8080,
	}, nil
}

func (m *mockRegistry) ListServices(ctx context.Context) ([]string, error) {
	return []string{"test-service"}, nil
}

func TestHTTPGateway_Basic(t *testing.T) {
	service := &mockService{}
	registry := &mockRegistry{}
	
	t.Run("创建HTTP网关", func(t *testing.T) {
		gw := gateway.NewHTTPGateway(service, registry, ":0")
		require.NotNil(t, gw)
	})
}

func TestHTTPGateway_APIRequests(t *testing.T) {
	service := &mockService{}
	registry := &mockRegistry{}
	gw := gateway.NewHTTPGateway(service, registry, ":0")
	
	t.Run("成功的API请求", func(t *testing.T) {
		// 准备请求数据
		requestData := map[string]interface{}{
			"message": "test request",
		}
		jsonData, _ := json.Marshal(requestData)
		
		// 创建HTTP请求
		req := httptest.NewRequest("POST", "/api/test-service/test-method", bytes.NewReader(jsonData))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		
		// 创建响应记录器
		w := httptest.NewRecorder()
		
		// 执行请求
		gw.HandleAPIRequest(w, req)
		
		// 验证响应
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "success", response["message"])
	})
	
	t.Run("无效的API路径", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/invalid", nil)
		w := httptest.NewRecorder()
		
		gw.HandleAPIRequest(w, req)
		
		assert.Equal(t, 400, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid API path")
	})
	
	t.Run("服务调用失败", func(t *testing.T) {
		failingService := &mockService{
			handleFunc: func(ctx context.Context, req types.Request) (types.Response, error) {
				return types.Response{}, assert.AnError
			},
		}
		
		gw := gateway.NewHTTPGateway(failingService, registry, ":0")
		
		req := httptest.NewRequest("POST", "/api/test-service/test-method", bytes.NewReader([]byte("{}")))
		w := httptest.NewRecorder()
		
		gw.HandleAPIRequest(w, req)
		
		assert.Equal(t, 500, w.Code)
		assert.Contains(t, w.Body.String(), "Service call failed")
	})
	
	t.Run("请求体读取失败", func(t *testing.T) {
		// 创建一个会在读取时失败的请求体
		req := httptest.NewRequest("POST", "/api/test-service/test-method", nil)
		req.Body = &failingReader{}
		
		w := httptest.NewRecorder()
		
		gw.HandleAPIRequest(w, req)
		
		assert.Equal(t, 400, w.Code)
		assert.Contains(t, w.Body.String(), "Failed to read request body")
	})
}

func TestHTTPGateway_HealthCheck(t *testing.T) {
	service := &mockService{}
	registry := &mockRegistry{}
	gw := gateway.NewHTTPGateway(service, registry, ":0")
	
	t.Run("健康检查端点", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		
		gw.HandleHealth(w, req)
		
		assert.Equal(t, 200, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
		assert.NotEmpty(t, response["time"])
	})
}

func TestHTTPGateway_MetadataHandling(t *testing.T) {
	var capturedRequest types.Request
	service := &mockService{
		handleFunc: func(ctx context.Context, req types.Request) (types.Response, error) {
			capturedRequest = req
			return types.Response{
				ID:     req.ID,
				Status: 200,
				Body:   []byte(`{"success": true}`),
			}, nil
		},
	}
	
	registry := &mockRegistry{}
	gw := gateway.NewHTTPGateway(service, registry, ":0")
	
	t.Run("HTTP头转换为元数据", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/test-service/test-method", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "test-client")
		req.Header.Set("X-Request-ID", "test-request-123")
		
		w := httptest.NewRecorder()
		
		gw.HandleAPIRequest(w, req)
		
		assert.Equal(t, 200, w.Code)
		
		// 验证元数据
		assert.Equal(t, "test-service", capturedRequest.Service)
		assert.Equal(t, "test-method", capturedRequest.Method)
		assert.Equal(t, "application/json", capturedRequest.Metadata["Content-Type"])
		assert.Equal(t, "test-client", capturedRequest.Metadata["User-Agent"])
		assert.Equal(t, "test-request-123", capturedRequest.Metadata["X-Request-Id"])
		assert.Equal(t, "POST", capturedRequest.Metadata["http-method"])
	})
}

func TestHTTPGateway_ErrorStatusCodes(t *testing.T) {
	registry := &mockRegistry{}
	
	testCases := []struct {
		name           string
		responseStatus int
		expectedCode   int
	}{
		{"成功响应", 200, 200},
		{"客户端错误", 400, 400},
		{"未找到", 404, 404},
		{"服务器错误", 500, 500},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service := &mockService{
				handleFunc: func(ctx context.Context, req types.Request) (types.Response, error) {
					return types.Response{
						ID:     req.ID,
						Status: tc.responseStatus,
						Body:   []byte(`{"status": "test"}`),
					}, nil
				},
			}
			
			gw := gateway.NewHTTPGateway(service, registry, ":0")
			
			req := httptest.NewRequest("GET", "/api/test-service/test-method", nil)
			w := httptest.NewRecorder()
			
			gw.HandleAPIRequest(w, req)
			
			assert.Equal(t, tc.expectedCode, w.Code)
		})
	}
}

func TestHTTPGateway_Concurrency(t *testing.T) {
	service := &mockService{
		handleFunc: func(ctx context.Context, req types.Request) (types.Response, error) {
			// 模拟一些处理时间
			time.Sleep(10 * time.Millisecond)
			return types.Response{
				ID:     req.ID,
				Status: 200,
				Body:   []byte(`{"processed": true}`),
			}, nil
		},
	}
	
	registry := &mockRegistry{}
	gw := gateway.NewHTTPGateway(service, registry, ":0")
	
	t.Run("并发请求处理", func(t *testing.T) {
		const numRequests = 10
		results := make(chan int, numRequests)
		
		for i := 0; i < numRequests; i++ {
			go func(id int) {
				req := httptest.NewRequest("POST", "/api/test-service/test-method", 
					bytes.NewReader([]byte(`{"id": ` + string(rune(id+'0')) + `}`)))
				w := httptest.NewRecorder()
				
				gw.HandleAPIRequest(w, req)
				results <- w.Code
			}(i)
		}
		
		// 收集结果
		for i := 0; i < numRequests; i++ {
			select {
			case code := <-results:
				assert.Equal(t, 200, code)
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for response")
			}
		}
	})
}

func TestHTTPGateway_Lifecycle(t *testing.T) {
	service := &mockService{}
	registry := &mockRegistry{}
	
	t.Run("启动和停止", func(t *testing.T) {
		gw := gateway.NewHTTPGateway(service, registry, ":0")
		
		// 测试停止（即使没有启动也应该正常工作）
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		
		err := gw.Stop(ctx)
		assert.NoError(t, err)
	})
}

// failingReader 模拟读取失败的Reader
type failingReader struct{}

func (f *failingReader) Read(p []byte) (n int, err error) {
	return 0, assert.AnError
}

func (f *failingReader) Close() error {
	return nil
}

func BenchmarkHTTPGateway(b *testing.B) {
	service := &mockService{}
	registry := &mockRegistry{}
	gw := gateway.NewHTTPGateway(service, registry, ":0")
	
	requestData := []byte(`{"test": "data"}`)
	
	b.Run("API请求处理", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/api/test-service/test-method", bytes.NewReader(requestData))
			w := httptest.NewRecorder()
			
			gw.HandleAPIRequest(w, req)
		}
	})
	
	b.Run("健康检查", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			
			gw.HandleHealth(w, req)
		}
	})
}