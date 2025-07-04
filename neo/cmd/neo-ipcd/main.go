package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"neo/internal/config"
	"neo/internal/connection/tcp"
	"neo/internal/discovery"
	"neo/internal/ipcprotocol"
	"neo/internal/metrics"
	"neo/internal/transport"
	"neo/internal/types"
	"os"
	"os/signal"
	"syscall"

	"github.com/mitchellh/mapstructure"
)

func main() {
	// 获取全局配置
	globalConfig := config.GetGlobalConfig()

	// 启动指标服务器（修复循环依赖）
	if err := metrics.StartServer(&globalConfig.Metrics); err != nil {
		log.Fatalf("启动监控服务器失败: %v", err)
	}

	// 初始化服务注册和工作池
	serviceRegistry := types.NewServiceRegistry()
	workerPool := transport.NewWorkerPool(
		globalConfig.IPC.WorkerCount,
		globalConfig.IPC.WorkerCount*2, // 队列容量
	)
	// 使用types包中的WorkerPoolAdapter
	adaptedWorkerPool := &types.WorkerPoolAdapter{WorkerPool: workerPool}

	serverConfig := &types.TCPConfig{
		MaxConnections:    globalConfig.IPC.MaxConnections,
		MaxMsgSize:        globalConfig.Protocol.MaxMessageSize,
		ReadTimeout:       globalConfig.IPC.ReadTimeout,
		WriteTimeout:      globalConfig.IPC.WriteTimeout,
		WorkerCount:       globalConfig.IPC.WorkerCount,
		ConnectionTimeout: globalConfig.IPC.ConnectionTimeout,
	}

	messageHandler := func(data []byte) ([]byte, error) {
		// 使用 discovery.IPCRequest 而非 types.Request
		req := &discovery.IPCRequest{}
		if err := json.Unmarshal(data, req); err != nil {
			return nil, fmt.Errorf("failed to unmarshal request: %v", err)
		}

		// 添加调试日志
		fmt.Printf("[DEBUG] 处理请求: %+v\n", req)

		// 处理注册请求
		if req.Action == "register" {
			service := &types.Service{}
			if err := mapstructure.Decode(req.Service, service); err != nil {
				return json.Marshal(map[string]interface{}{
					"success": false,
					"error":   fmt.Sprintf("invalid service data: %v", err),
				})
			}

			// 调用发现服务的注册方法
			// 1. 确保已初始化Discovery实例（通常在main函数开头）
			// 修改服务发现初始化代码
			// 初始化内存存储（使用实际构造函数）
			storage := discovery.NewInMemoryStorage()
			// 创建Discovery实例 - 修改这一行
			// discoveryInstance := discovery.New(storage)
			discoveryInstance := types.NewDiscovery(storage)
			// 包装成DiscoveryService
			discoveryService := &discovery.DiscoveryService{Discovery: discoveryInstance}

			// 调用Register方法
			if err := discoveryService.Register(context.Background(), service); err != nil {
				return json.Marshal(map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				})
			}

			return json.Marshal(map[string]interface{}{
				"success": true,
				"result":  service,
			})
		}

		// 原有处理逻辑
		respData, err := ipcprotocol.ProcessMessage(data, serviceRegistry, adaptedWorkerPool)
		if err != nil {
			return nil, err
		}

		return respData, nil
	}

	server, err := tcp.NewServer(serverConfig, messageHandler)
	if err != nil {
		fmt.Printf("Failed to create TCP server: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting NEO IPC server...")
	if err := server.Start(); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
	defer server.Stop()

	fmt.Println("Server is running. Press Ctrl+C to stop.")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	fmt.Println("Received interrupt signal, shutting down...")
}
