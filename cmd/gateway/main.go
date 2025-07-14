package main

import (
	"context"
	"neo/internal/config"
	"neo/internal/core"
	"neo/internal/gateway"
	"neo/internal/ipc"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/utils"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 加载配置
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  100,
			MinConnections:  10,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}

	// 设置日志
	logger := utils.DefaultLogger
	logger.Info("Starting Neo Framework Gateway")

	// 创建服务注册中心
	serviceRegistry := registry.NewServiceRegistry(registry.WithLogger(logger))

	// 创建并启动IPC服务器
	ipcServer := ipc.NewIPCServer(":19999", serviceRegistry)
	if err := ipcServer.Start(); err != nil {
		logger.Error("Failed to start IPC server", utils.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("IPC Server started", utils.String("address", ":19999"))

	// 创建异步IPC服务器（暂时不使用，但保留以备将来扩展）
	_ = ipc.NewAsyncIPCServer(ipcServer)

	// 创建传输层
	transportLayer := transport.NewTransport(cfg)

	// 创建核心服务选项
	serviceOpts := core.ServiceOptions{
		Name:      "gateway-service",
		Transport: transportLayer,
		Registry:  serviceRegistry,
		Timeout:   30 * time.Second,
		Logger:    logger,
	}

	// 创建核心服务
	coreService := core.NewService(serviceOpts)
	logger.Info("Core service created")

	// 创建并启动HTTP网关
	httpGateway := gateway.NewHTTPGateway(coreService, serviceRegistry, ":18080")
	
	// 启动HTTP服务器（在goroutine中）
	go func() {
		logger.Info("Starting HTTP Gateway", utils.String("address", ":18080"))
		if err := httpGateway.Start(); err != nil {
			logger.Error("HTTP Gateway error", utils.String("error", err.Error()))
		}
	}()

	// 等待HTTP服务器启动
	time.Sleep(100 * time.Millisecond)
	logger.Info("Neo Framework Gateway started successfully")
	logger.Info("HTTP Gateway: http://localhost:18080")
	logger.Info("IPC Server: localhost:19999")
	logger.Info("Health Check: http://localhost:18080/health")

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Received shutdown signal, starting graceful shutdown...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 关闭HTTP网关
	if err := httpGateway.Stop(ctx); err != nil {
		logger.Error("HTTP Gateway shutdown error", utils.String("error", err.Error()))
	} else {
		logger.Info("HTTP Gateway stopped")
	}

	// 关闭IPC服务器
	if err := ipcServer.Stop(); err != nil {
		logger.Error("IPC Server shutdown error", utils.String("error", err.Error()))
	} else {
		logger.Info("IPC Server stopped")
	}

	// 关闭核心服务
	if err := coreService.Close(); err != nil {
		logger.Error("Core service shutdown error", utils.String("error", err.Error()))
	} else {
		logger.Info("Core service stopped")
	}

	logger.Info("Neo Framework Gateway shutdown complete")
}