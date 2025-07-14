package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"neo/internal/config"
	"neo/internal/core"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/utils"
)

// parseFlags 解析命令行参数
func parseFlags() string {
	configPath := flag.String("config", "configs/default.yml", "配置文件路径")
	flag.Parse()
	return *configPath
}

func main() {
	// 1. 解析命令行参数
	configPath := parseFlags()

	// 2. 创建日志器
	logger := utils.DefaultLogger
	logger.Info("Starting Neo Framework Service")

	// 3. 加载配置
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  100,
			MinConnections:  10,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}
	logger.Info("Configuration loaded", utils.String("configPath", configPath))

	// 4. 初始化服务注册中心
	registry := registry.NewServiceRegistry(registry.WithLogger(logger))

	// 5. 初始化传输层
	transport := transport.NewTransport(cfg)

	// 6. 初始化核心服务
	serviceOpts := core.ServiceOptions{
		Name:      "neo-service",
		Transport: transport,
		Registry:  registry,
		Timeout:   30 * time.Second,
		Logger:    logger,
	}
	coreSvc := core.NewService(serviceOpts)

	// 7. 启动监听器
	go func() {
		logger.Info("Starting transport listener")
		if err := transport.StartListener(); err != nil {
			logger.Error("Failed to start listener", utils.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	logger.Info("Neo Framework Service started successfully")

	// 8. 等待关闭信号
	waitForShutdown(coreSvc, transport, logger)
}

// shutdown 实现优雅关闭逻辑
func shutdown(coreSvc core.Service, transport transport.Transport, logger utils.Logger) {
	logger.Info("Starting graceful shutdown...")
	
	// 关闭监听器
	if err := transport.StopListener(); err != nil {
		logger.Error("Failed to stop listener", utils.String("error", err.Error()))
	} else {
		logger.Info("Transport listener stopped")
	}
	
	// 关闭核心服务
	if err := coreSvc.Close(); err != nil {
		logger.Error("Failed to close core service", utils.String("error", err.Error()))
	} else {
		logger.Info("Core service closed")
	}
	
	logger.Info("Neo Framework Service shutdown complete")
}

// waitForShutdown 监听系统信号触发关闭
func waitForShutdown(coreSvc core.Service, transport transport.Transport, logger utils.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	shutdown(coreSvc, transport, logger)
}
