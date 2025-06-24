package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"neo/internal/config"
	"neo/internal/config/loader"
	"neo/internal/discovery"
	"neo/internal/ipcprotocol"
	"neo/internal/transport"
	"os"
	"os/signal"
	"syscall"
	"time"

	"neo/internal/common"
)

// 服务处理函数实现
func pingService(request *ipcprotocol.Request) (*ipcprotocol.Response, error) {
	// 正确使用ipcprotocol.Response结构体字段，并显式转换ErrorCode类型
	responseData := map[string]string{"message": "pong"}
	jsonData, err := json.Marshal(responseData)
	if err != nil {
		return nil, fmt.Errorf("序列化响应数据失败: %v", err)
	}

	return &ipcprotocol.Response{
		Code:    ipcprotocol.ErrorCodeSuccess,
		Message: "success",
		Data:    jsonData,
	}, nil
}

func configService(request *ipcprotocol.Request) (*ipcprotocol.Response, error) {
	currentConfig := config.Get()
	jsonData, err := json.Marshal(currentConfig)
	if err != nil {
		return nil, fmt.Errorf("序列化配置数据失败: %v", err)
	}

	return &ipcprotocol.Response{
		Code:    ipcprotocol.ErrorCodeSuccess,
		Message: "success",
		Data:    jsonData,
	}, nil
}

func statusService(request *ipcprotocol.Request) (*ipcprotocol.Response, error) {
	statusData := map[string]string{
		"status":  "running",
		"time":    time.Now().Format(time.RFC3339),
		"version": "1.0.0",
	}

	jsonData, err := json.Marshal(statusData)
	if err != nil {
		return nil, fmt.Errorf("序列化状态数据失败: %v", err)
	}

	return &ipcprotocol.Response{
		Code:    ipcprotocol.ErrorCodeSuccess,
		Message: "success",
		Data:    jsonData,
	}, nil
}

func main() {
	// 加载配置
	var cfg config.GlobalConfig
	configPath := "/www/neo/neo/configs/default.yml"
	if err := loader.LoadFromFile(configPath, &cfg); err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 验证配置
	log.Printf("从文件加载的配置 - initial=%d, min=%d, max=%d",
		cfg.Pool.InitialSize, cfg.Pool.MinSize, cfg.Pool.MaxSize)

	if cfg.Pool.InitialSize < cfg.Pool.MinSize || cfg.Pool.InitialSize > cfg.Pool.MaxSize {
		log.Fatalf("连接池初始大小(%d)必须介于最小(%d)和最大(%d)连接数之间",
			cfg.Pool.InitialSize, cfg.Pool.MinSize, cfg.Pool.MaxSize)
	}

	if cfg.IPC.Port <= 0 || cfg.IPC.Port > 65535 {
		log.Fatalf("无效的IPC端口配置: %d", cfg.IPC.Port)
	}

	// 更新全局配置
	config.Update(cfg)
	currentConfig := config.Get()
	log.Printf("全局配置已更新 - 连接池参数: initial=%d, min=%d, max=%d",
		currentConfig.Pool.InitialSize, currentConfig.Pool.MinSize, currentConfig.Pool.MaxSize)
	log.Printf("已加载配置: IPC=%s:%d, 连接池大小=%d-%d", cfg.IPC.Host, cfg.IPC.Port, cfg.Pool.MinSize, cfg.Pool.MaxSize)

	// 创建IPC服务器配置 - 修复指针类型不匹配问题
	ipcConfig := transport.NewIPCServerConfigFromGlobal(&currentConfig)

	// 创建IPC服务器
	server, err := transport.NewIPCServer(ipcConfig)
	if err != nil {
		log.Fatalf("创建IPC服务器失败: %v", err)
	}

	// 启动指标服务器
	if err := server.Metrics().StartServer(); err != nil {
		log.Printf("启动指标服务器失败: %v", err)
	}

	// 初始化服务发现存储
	log.Println("Initializing discovery storage...")
	discoveryStorage := discovery.NewInMemoryStorage()
	discoveryService := discovery.New(discoveryStorage)
	defer discoveryService.Close()

	// 注册服务发现的处理函数到IPC服务器
	// 使用common.ServiceHandlerFunc包装处理函数，实现ServiceHandler接口
	server.RegisterService("register", common.ServiceHandlerFunc(func(request *ipcprotocol.Request) (*ipcprotocol.Response, error) {
		// 解析请求参数 - 将Payload改为正确的Params字段
		var params map[string]interface{}
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return nil, fmt.Errorf("解析请求参数失败: %v", err)
		}

		// 直接从原始params中获取service字段
		serviceData, ok := params["service"].(map[string]interface{})
		if !ok || serviceData == nil {
			return nil, fmt.Errorf("service字段格式错误或为空")
		}

		// 从serviceData中提取字段
		serviceID, ok := serviceData["id"].(string)
		if !ok {
			return nil, fmt.Errorf("缺少服务ID或格式错误")
		}

		serviceName, ok := serviceData["name"].(string)
		if !ok {
			return nil, fmt.Errorf("缺少服务名称或格式错误")
		}

		address, ok := serviceData["address"].(string)
		if !ok {
			return nil, fmt.Errorf("缺少服务地址或格式错误")
		}

		portNum, ok := serviceData["port"].(float64)
		if !ok {
			return nil, fmt.Errorf("缺少服务端口或格式错误")
		}
		port := int(portNum)

		// 转换metadata类型
		metadataIface, ok := serviceData["metadata"].(map[string]interface{})
		if !ok || metadataIface == nil {
			return nil, fmt.Errorf("metadata格式错误或为空")
		}
		metadata := make(map[string]string)
		for k, v := range metadataIface {
			metadata[k] = fmt.Sprintf("%v", v)
		}

		// 创建服务实例
		service := &discovery.Service{
			ID:        serviceID,
			Name:      serviceName,
			Address:   address,
			Port:      port,
			Metadata:  metadata,
			Status:    "healthy",
			UpdatedAt: time.Now(),
			ExpireAt:  time.Now().Add(30 * time.Second),
		}

		// 调用服务发现模块注册服务
		if err := discoveryService.Register(context.Background(), service); err != nil {
			return nil, fmt.Errorf("注册失败: %v", err)
		}

		// 构建响应
		responseData := map[string]string{"id": serviceID}
		jsonData, err := json.Marshal(responseData)
		if err != nil {
			return nil, fmt.Errorf("序列化响应数据失败: %v", err)
		}

		return &ipcprotocol.Response{
			Code:    ipcprotocol.ErrorCodeSuccess,
			Message: "success",
			Data:    jsonData,
		}, nil
	}))

	// 注册内置服务 - 使用common.ServiceHandlerFunc包装处理函数
	server.RegisterService("ping", common.ServiceHandlerFunc(pingService))
	server.RegisterService("config", common.ServiceHandlerFunc(configService))
	server.RegisterService("status", common.ServiceHandlerFunc(statusService))

	// 启动服务器
	if err := server.Start(); err != nil {
		log.Fatalf("启动IPC服务器失败: %v", err)
	}
	defer server.Stop()

	log.Println("服务已启动，按Ctrl+C退出...")

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("开始关闭服务...")
}
