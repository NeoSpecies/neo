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
	// 解析命令行参数
	configPath := "configs/default.yml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	// 加载配置
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		utils.DefaultLogger.Error("Failed to load config", utils.String("error", err.Error()))
		cfg = config.DefaultConfig()
	}

	// 设置日志
	logger := utils.DefaultLogger
	logger.Info("Starting Neo Framework Gateway")

	// 创建服务注册中心
	registryConfig := registry.RegistryConfig{
		CleanupInterval:     time.Duration(cfg.Registry.CleanupInterval),
		InstanceExpiry:      time.Duration(cfg.Registry.InstanceExpiry),
		HealthCheckInterval: time.Duration(cfg.Registry.HealthCheckInterval),
	}
	serviceRegistry := registry.NewServiceRegistry(
		registry.WithLogger(logger),
		registry.WithConfig(registryConfig),
	)

	// 创建并启动IPC服务器
	ipcConfig := ipc.IPCConfig{
		MaxMessageSize: cfg.IPC.MaxMessageSize,
		BufferSize:     cfg.IPC.BufferSize,
	}
	ipcServer := ipc.NewIPCServerWithConfig(cfg.IPC.Address, serviceRegistry, ipcConfig)
	if err := ipcServer.Start(); err != nil {
		logger.Error("Failed to start IPC server", utils.String("error", err.Error()))
		os.Exit(1)
	}
	logger.Info("IPC Server started", utils.String("address", cfg.IPC.Address))

	// 创建异步IPC服务器（暂时不使用，但保留以备将来扩展）
	_ = ipc.NewAsyncIPCServer(ipcServer)

	// 创建传输层
	transportConfig := transport.Config{
		Timeout:               time.Duration(cfg.Transport.Timeout),
		RetryCount:            cfg.Transport.RetryCount,
		MaxConnections:        cfg.Transport.MaxConnections,
		MinConnections:        cfg.Transport.MinConnections,
		MaxIdleTime:           time.Duration(cfg.Transport.MaxIdleTime),
		HealthCheckInterval:   time.Duration(cfg.Transport.HealthCheckInterval),
		ActivityCheckInterval: time.Duration(cfg.Transport.ActivityCheckInterval),
	}
	transportLayer := transport.NewTransport(transportConfig)

	// 创建核心服务选项
	serviceOpts := core.ServiceOptions{
		Name:      cfg.Server.Name,
		Transport: transportLayer,
		Registry:  serviceRegistry,
		Timeout:   time.Duration(cfg.Transport.Timeout),
		Logger:    logger,
	}

	// 创建核心服务
	coreService := core.NewService(serviceOpts)
	logger.Info("Core service created")

	// 创建并启动HTTP网关
	httpGateway := gateway.NewHTTPGateway(coreService, serviceRegistry, cfg.Gateway.Address)
	
	// 启动HTTP服务器（在goroutine中）
	go func() {
		logger.Info("Starting HTTP Gateway", utils.String("address", cfg.Gateway.Address))
		if err := httpGateway.Start(); err != nil {
			logger.Error("HTTP Gateway error", utils.String("error", err.Error()))
		}
	}()

	// 等待HTTP服务器启动
	time.Sleep(time.Duration(cfg.Server.StartupDelay))
	logger.Info("Neo Framework Gateway started successfully")
	logger.Info("HTTP Gateway: http://localhost" + cfg.Gateway.Address)
	logger.Info("IPC Server: localhost" + cfg.IPC.Address)
	logger.Info("Health Check: http://localhost" + cfg.Gateway.Address + "/health")

	// 等待中断信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Received shutdown signal, starting graceful shutdown...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.Server.ShutdownTimeout))
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