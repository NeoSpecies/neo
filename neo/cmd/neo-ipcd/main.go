package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"neo/internal/config"
	neohttp "neo/internal/connection/http" // 仅需导入内部HTTP包
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
	"sync"
	"syscall"

	"github.com/mitchellh/mapstructure"
)

func main() {
	// 获取全局配置
	globalConfig := config.GetGlobalConfig()

	// 启动指标服务器
	if err := metrics.StartServer(&globalConfig.Metrics); err != nil {
		log.Fatalf("启动监控服务器失败: %v", err)
	}

	// 初始化服务注册和工作池
	serviceRegistry := types.NewServiceRegistry()
	workerPool := transport.NewWorkerPool(
		globalConfig.IPC.WorkerCount,
		globalConfig.IPC.WorkerCount*2, // 队列容量
	)
	adaptedWorkerPool := &types.WorkerPoolAdapter{WorkerPool: workerPool}

	// ======== TCP服务器启动 ========
	serverConfig := &types.TCPConfig{
		MaxConnections:    globalConfig.IPC.MaxConnections,
		MaxMsgSize:        globalConfig.Protocol.MaxMessageSize,
		ReadTimeout:       globalConfig.IPC.ReadTimeout,
		WriteTimeout:      globalConfig.IPC.WriteTimeout,
		WorkerCount:       globalConfig.IPC.WorkerCount,
		ConnectionTimeout: globalConfig.IPC.ConnectionTimeout,
		Address:           net.JoinHostPort(globalConfig.IPC.Host, strconv.Itoa(globalConfig.IPC.Port)),
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

		// 使用单例服务发现实例
		discoveryService := discovery.GetDiscoveryService()

		// 处理注册请求
		fmt.Printf("[DEBUG] 请求类型是否为注册: %v\n", req.Action == "register")
		// 测试不匹配时打印 req 的字段参数及类型
		if req.Action != "register" {
			fmt.Printf("[DEBUG] 请求类型不匹配，当前请求类型: %T, 值: %v\n", req.Action, req.Action)
			fmt.Printf("[DEBUG] 请求全部字段: %+v\n", req)
		}
		if req.Action == "register" {
			service := &types.Service{}
			if err := mapstructure.Decode(req.Service, service); err != nil {
				// 返回包含result字段的错误响应
				return json.Marshal(map[string]interface{}{
					"result":  "error",
					"message": fmt.Sprintf("failed to decode service: %v", err),
				})
			}

			// 新增：使用单例服务实例注册服务
			if err := discoveryService.Register(context.Background(), service); err != nil {
				return json.Marshal(map[string]interface{}{
					"result":  "error",
					"message": fmt.Sprintf("registration failed: %v", err),
				})
			}

			// 返回包含result字段的成功响应
			return json.Marshal(map[string]interface{}{
				"result": map[string]string{ // 修改为字典类型
					"id":     service.ID, // 包含服务ID
					"status": "registered",
				},
				"message": "service registered successfully",
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
	tcpServer, err := tcp.NewServer(serverConfig, messageHandler)
	if err != nil {
		fmt.Printf("Failed to create TCP server: %v\n", err)
		os.Exit(1)
	}

	// ======== HTTP服务器启动 ========
	httpServer := neohttp.NewServer(&globalConfig.HTTP)

	// 新增: 注册服务查询接口
	httpServer.RegisterHandler("/services", func(w neohttp.ResponseWriter, r *neohttp.Request) {
		// 使用完整包路径调用单例函数
		ds := discovery.GetDiscoveryService()
		// 修复：提供参数并接收两个返回值
		services, err := ds.GetServices("")
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(neohttp.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "error",
				"message": "failed to retrieve services",
				"error":   err.Error(),
			})
			return
		}

		resp := map[string]interface{}{
			"status":   "success",
			"count":    len(services),
			"services": services,
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(neohttp.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	// 使用WaitGroup等待两个服务器都启动
	var wg sync.WaitGroup

	// 启动TCP服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		fmt.Printf("Starting NEO IPC server on %s...\n", serverConfig.Address)
		// 使用自定义HTTP包的错误
		if err := tcpServer.Start(); err != nil && err != neohttp.ErrServerClosed {
			fmt.Printf("TCP server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	// 启动HTTP服务器
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 使用自定义HTTP包的错误
		if err := httpServer.Start(); err != nil && err != neohttp.ErrServerClosed {
			fmt.Printf("HTTP server failed: %v\n", err)
			os.Exit(1)
		}
	}()

	// 优雅关闭处理
	fmt.Println("Servers are running. Press Ctrl+C to stop.")
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Received interrupt signal, shutting down...")
	// 关闭HTTP服务器
	if err := httpServer.Close(); err != nil {
		fmt.Printf("HTTP server shutdown error: %v\n", err)
	}
	// 关闭TCP服务器
	if err := tcpServer.Stop(); err != nil {
		fmt.Printf("TCP server shutdown error: %v\n", err)
	}

	// 等待所有服务器关闭
	wg.Wait()
	fmt.Println("All servers stopped.")
}
