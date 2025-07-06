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
	"net"
	"os"
	"os/signal"
	"strconv"
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

	// 初始化服务器配置
	serverConfig := &types.TCPConfig{
		MaxConnections:    globalConfig.IPC.MaxConnections,
		MaxMsgSize:        globalConfig.Protocol.MaxMessageSize,
		ReadTimeout:       globalConfig.IPC.ReadTimeout,
		WriteTimeout:      globalConfig.IPC.WriteTimeout,
		WorkerCount:       globalConfig.IPC.WorkerCount,
		ConnectionTimeout: globalConfig.IPC.ConnectionTimeout,
		// 直接设置地址，避免在TCPConfig内部依赖config包
		Address: net.JoinHostPort(globalConfig.IPC.Host, strconv.Itoa(globalConfig.IPC.Port)),
	}

	messageHandler := func(data []byte) ([]byte, error) {
		log.Printf("收到请求数据: %s\n", string(data))
		// 使用 discovery.IPCRequest 而非 types.Request
		req := &discovery.IPCRequest{}

		if err := json.Unmarshal(data, req); err != nil {
			log.Printf("Failed to unmarshal request: %v, raw data: %s", err, string(data))
			return nil, fmt.Errorf("failed to unmarshal request: %v", err)
		}
		// 添加调试日志
		fmt.Printf("[DEBUG] 处理请求: %+v\n", req)

		// 处理注册请求
		if req.Type == "register" {
			service := &types.Service{}
			if err := mapstructure.Decode(req.Service, service); err != nil {
				// 返回包含result字段的错误响应
				return json.Marshal(map[string]interface{}{
					"result": nil,
					"error": map[string]interface{}{
						"code":    "INVALID_SERVICE_DATA",
						"message": fmt.Sprintf("invalid service data: %v", err),
					},
				})
			}

			// 调用发现服务的注册方法
			storage := discovery.NewInMemoryStorage()
			discoveryInstance := types.NewDiscovery(storage)
			discoveryService := &discovery.DiscoveryService{Discovery: discoveryInstance}

			if err := discoveryService.Register(context.Background(), service); err != nil {
				// 返回包含result字段的错误响应
				return json.Marshal(map[string]interface{}{
					"result": nil,
					"error": map[string]interface{}{
						"code":    "REGISTRATION_FAILED",
						"message": err.Error(),
					},
				})
			}

			// 返回包含result字段的成功响应
			return json.Marshal(map[string]interface{}{
				"result": service,
				"error":  nil,
			})
		}

		// 原有处理逻辑
		fmt.Printf("[DEBUG] 处理非注册请求: %s\n", string(data))
		respData, err := ipcprotocol.ProcessMessage(data, serviceRegistry, adaptedWorkerPool)
		if err != nil {
			log.Printf("[ERROR] 非注册请求处理失败: %v\n", err)
			return nil, err
		}

		return respData, nil
	}
	// 创建TCP服务器时注入tcp包的工厂函数
	server, err := tcp.NewServer(serverConfig, messageHandler)
	if err != nil {
		fmt.Printf("Failed to create TCP server: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Starting NEO IPC server on %s...\n", serverConfig.Address) // 修改
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
