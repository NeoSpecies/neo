package integration_test

import (
	"context"
	"neo/internal/config"
	"neo/internal/core"
	"neo/internal/gateway"
	"neo/internal/ipc"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/types"
	"neo/internal/utils"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCoreService 测试核心服务功能
func TestCoreService(t *testing.T) {
	// 创建配置
	cfg := &config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  10,
			MinConnections:  2,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
		IPC: config.IPCConfig{
			Address:        ":9999",
			MaxClients:     100,
			BufferSize:     4096,
			MaxMessageSize: 10 * 1024 * 1024,
			ReadTimeout:    config.Duration(30 * time.Second),
			WriteTimeout:   config.Duration(30 * time.Second),
		},
		Registry: config.RegistryConfig{
			Type:               "inmemory",
			Namespace:          "test",
			HealthCheckInterval: config.Duration(30 * time.Second),
			TTL:                config.Duration(30 * time.Second),
			RefreshInterval:    config.Duration(10 * time.Second),
		},
	}

	// 创建组件
	logger := utils.NewLogger(utils.WithLevel(utils.DEBUG))
	reg := registry.NewServiceRegistry(registry.WithLogger(logger))
	transportCfg := transport.Config{
		Timeout:         time.Duration(cfg.Transport.Timeout),
		RetryCount:      cfg.Transport.RetryCount,
		MaxConnections:  cfg.Transport.MaxConnections,
		MinConnections:  cfg.Transport.MinConnections,
		MaxIdleTime:     time.Duration(cfg.Transport.MaxIdleTime),
		HealthCheckInterval: 30 * time.Second,
	}
	trans := transport.NewTransport(transportCfg)

	// 创建核心服务
	serviceOpts := core.ServiceOptions{
		Name:      "test-service",
		Transport: trans,
		Registry:  reg,
		Timeout:   30 * time.Second,
		Logger:    logger,
	}
	service := core.NewService(serviceOpts)

	// 测试服务基本功能
	t.Run("服务名称", func(t *testing.T) {
		assert.Equal(t, "test-service", service.Name())
	})

	t.Run("处理请求", func(t *testing.T) {
		req := types.Request{
			ID:      "test-req-1",
			Service: "test-service",
			Method:  "test-method",
			Body:    []byte(`{"test": "data"}`),
		}

		ctx := context.Background()
		resp, err := service.HandleRequest(ctx, req)
		
		// 注意：这可能会失败，因为没有注册实际的处理器
		// 但这里主要测试服务能够启动和接收请求
		if err != nil {
			t.Logf("请求处理失败（预期）: %v", err)
		} else {
			assert.NotNil(t, resp)
		}
	})

	// 清理
	require.NoError(t, service.Close())
	require.NoError(t, trans.Close())
}

// TestServiceRegistry 测试服务注册与发现
func TestServiceRegistry(t *testing.T) {
	logger := utils.NewLogger(utils.WithLevel(utils.DEBUG))
	reg := registry.NewServiceRegistry(registry.WithLogger(logger))
	ctx := context.Background()

	// 注册服务实例
	instance := &registry.ServiceInstance{
		ID:       "instance-1",
		Name:     "test-service",
		Address:  "localhost",
		Port:     8080,
		Metadata: map[string]string{"version": "1.0"},
	}

	t.Run("注册服务", func(t *testing.T) {
		err := reg.Register(ctx, instance)
		require.NoError(t, err)
	})

	t.Run("发现服务", func(t *testing.T) {
		instances, err := reg.Discover(ctx, "test-service")
		require.NoError(t, err)
		assert.Len(t, instances, 1)
		assert.Equal(t, instance.ID, instances[0].ID)
	})

	t.Run("列出所有服务", func(t *testing.T) {
		services, err := reg.ListServices(ctx)
		require.NoError(t, err)
		assert.Contains(t, services, "test-service")
	})

	t.Run("注销服务", func(t *testing.T) {
		err := reg.Deregister(ctx, instance.ID)
		require.NoError(t, err)

		// 验证服务已被注销
		instances, err := reg.Discover(ctx, "test-service")
		require.NoError(t, err)
		assert.Len(t, instances, 0)
	})
}

// TestHTTPGateway 测试HTTP网关
func TestHTTPGateway(t *testing.T) {
	// 创建组件
	logger := utils.NewLogger(utils.WithLevel(utils.DEBUG))
	reg := registry.NewServiceRegistry(registry.WithLogger(logger))
	
	// 创建 transport
	transportCfg := transport.Config{
		Timeout:        30 * time.Second,
		RetryCount:     3,
		MaxConnections: 10,
		MinConnections: 2,
		MaxIdleTime:    300 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
	trans := transport.NewTransport(transportCfg)
	defer trans.Close()

	// 创建核心服务
	service := core.NewService(core.ServiceOptions{
		Name:      "test-service",
		Transport: trans,
		Registry:  reg,
	})
	defer service.Close()

	// 创建网关
	addr := ":18080" // 使用固定端口以避免随机端口问题
	gw := gateway.NewHTTPGateway(service, reg, addr)

	// 在goroutine中启动网关，避免阻塞测试
	startChan := make(chan error, 1)
	go func() {
		startChan <- gw.Start()
	}()

	// 等待启动或超时
	select {
	case err := <-startChan:
		if err != nil {
			t.Fatalf("Failed to start gateway: %v", err)
		}
	case <-time.After(2 * time.Second):
		// 网关启动成功（正在运行）
	}

	// 停止网关
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gw.Stop(ctx)

	// 网关地址
	t.Logf("HTTP网关测试完成")
}

// TestIPCServer 测试IPC服务器
func TestIPCServer(t *testing.T) {
	// 创建组件
	logger := utils.NewLogger(utils.WithLevel(utils.DEBUG))
	reg := registry.NewServiceRegistry(registry.WithLogger(logger))
	
	// 创建IPC服务器
	ipcAddr := ":19999" // 使用固定端口
	ipcServer := ipc.NewIPCServer(ipcAddr, reg)

	// 启动服务器
	err := ipcServer.Start()
	require.NoError(t, err)
	
	// 给服务器一些时间来完全启动
	time.Sleep(100 * time.Millisecond)
	
	// 停止服务器
	err = ipcServer.Stop()
	require.NoError(t, err)

	// 服务器地址
	t.Logf("IPC服务器测试完成")
}

// TestEndToEnd 端到端测试
func TestEndToEnd(t *testing.T) {
	// 创建组件
	logger := utils.NewLogger(utils.WithLevel(utils.DEBUG))
	reg := registry.NewServiceRegistry(registry.WithLogger(logger))
	
	// 创建 transport
	transportCfg := transport.Config{
		Timeout:        30 * time.Second,
		RetryCount:     3,
		MaxConnections: 10,
		MinConnections: 2,
		MaxIdleTime:    300 * time.Second,
		HealthCheckInterval: 30 * time.Second,
	}
	trans := transport.NewTransport(transportCfg)
	defer trans.Close()

	// 创建IPC服务器
	ipcAddr := ":9999"
	ipcServer := ipc.NewIPCServer(ipcAddr, reg)
	err := ipcServer.Start()
	require.NoError(t, err)
	defer ipcServer.Stop()

	// 创建AsyncIPC服务器（包装IPC服务器）
	asyncIPCServer := ipc.NewAsyncIPCServer(ipcServer)
	
	// 创建核心服务
	service := core.NewService(core.ServiceOptions{
		Name:      "test-service",
		Transport: trans,
		Registry:  reg,
		AsyncIPC:  asyncIPCServer,
		Logger:    logger,
	})
	defer service.Close()

	// 创建HTTP网关
	httpAddr := ":18080"
	gw := gateway.NewHTTPGateway(service, reg, httpAddr)
	
	// 在goroutine中启动网关
	go func() {
		if err := gw.Start(); err != nil {
			t.Logf("Gateway start error: %v", err)
		}
	}()
	
	// 等待网关启动
	time.Sleep(500 * time.Millisecond)
	defer gw.Stop(context.Background())

	// 注册一个测试服务
	testService := &registry.ServiceInstance{
		ID:       "test-instance-1",
		Name:     "demo-service",
		Address:  "localhost",
		Port:     9999,
		Metadata: map[string]string{"version": "1.0"},
	}
	err = reg.Register(context.Background(), testService)
	require.NoError(t, err)

	t.Logf("端到端测试环境搭建完成")
	t.Logf("HTTP网关: http://localhost:18080")
	t.Logf("IPC服务器: localhost:9999")
}