package main

import (
	"context"
	"fmt"
	"neo/internal/config"
	"neo/internal/core"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/types"
	"neo/internal/utils"
	"time"
)

func main() {
	fmt.Println("=== 测试核心服务 ===")
	
	// 创建注册中心
	reg := registry.NewServiceRegistry(registry.WithLogger(utils.DefaultLogger))
	
	// 手动注册一个Python服务
	instance := &registry.ServiceInstance{
		ID:       "test-python.math-127.0.0.1:32999",
		Name:     "python.math", 
		Address:  "127.0.0.1:32999",
		Metadata: map[string]string{"version": "1.0.0"},
	}
	
	ctx := context.Background()
	if err := reg.Register(ctx, instance); err != nil {
		fmt.Printf("❌ 服务注册失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 手动注册服务成功: %s\n", instance.Name)
	
	// 创建传输层
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
	
	// 创建核心服务（不使用AsyncIPC，模拟原始问题）
	serviceOpts := core.ServiceOptions{
		Name:      "test-service",
		Transport: transport,
		Registry:  reg,
		Timeout:   30 * time.Second,
		Logger:    utils.DefaultLogger,
		AsyncIPC:  nil, // 不设置AsyncIPC来测试服务发现
	}
	coreService := core.NewService(serviceOpts)
	
	// 创建测试请求
	req := types.Request{
		ID:       "test-123",
		Service:  "python.math",
		Method:   "add",
		Body:     []byte(`{"a": 5, "b": 3}`),
		Metadata: make(map[string]string),
	}
	
	fmt.Printf("🔍 发送测试请求到服务: %s\n", req.Service)
	
	// 调用核心服务
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	resp, err := coreService.HandleRequest(ctx, req)
	if err != nil {
		fmt.Printf("❌ 核心服务调用失败: %v\n", err)
		return
	}
	
	fmt.Printf("✅ 核心服务响应:\n")
	fmt.Printf("   ID: %s\n", resp.ID)
	fmt.Printf("   Status: %d\n", resp.Status)
	fmt.Printf("   Error: %s\n", resp.Error)
	fmt.Printf("   Body: %s\n", string(resp.Body))
}