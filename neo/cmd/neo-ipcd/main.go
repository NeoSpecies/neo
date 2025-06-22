package main

import (
	"context"
	"fmt"
	"log"
	"neo/internal/config"
	"neo/internal/config/loader"
	"neo/internal/discovery"
	"neo/internal/transport"
	"os"
	"time"
)

func main() {
	var cfg config.GlobalConfig
	err := loader.LoadFromFile("/www/neo/neo/configs/default.yml", &cfg)
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 新增：打印实际加载的配置值
	log.Printf("从文件加载的配置 - initial=%d, min=%d, max=%d",
		cfg.Pool.InitialSize, cfg.Pool.MinSize, cfg.Pool.MaxSize)

	// 新增：连接池配置验证
	if cfg.Pool.InitialSize < cfg.Pool.MinSize || cfg.Pool.InitialSize > cfg.Pool.MaxSize {
		log.Fatalf("连接池初始大小(%d)必须介于最小(%d)和最大(%d)连接数之间",
			cfg.Pool.InitialSize, cfg.Pool.MinSize, cfg.Pool.MaxSize)
	}

	// 新增：验证核心配置项
	if cfg.IPC.Port <= 0 || cfg.IPC.Port > 65535 {
		log.Fatalf("无效的IPC端口配置: %d", cfg.IPC.Port)
	}

	config.Update(cfg)
	// 新增：验证全局配置是否正确更新
	currentConfig := config.Get()
	log.Printf("全局配置已更新 - 连接池参数: initial=%d, min=%d, max=%d",
		currentConfig.Pool.InitialSize, currentConfig.Pool.MinSize, currentConfig.Pool.MaxSize)
	log.Printf("已加载配置: IPC=%s:%d, 连接池大小=%d-%d", cfg.IPC.Host, cfg.IPC.Port, cfg.Pool.MinSize, cfg.Pool.MaxSize)

	log.Println("Starting IPC server...")
	// 添加启动完成通知通道
	serverStarted := make(chan struct{})
	// 修改：将IPC服务器启动放入goroutine并添加启动回调
	go func() {
		err := transport.StartIpcServer()
		if err != nil {
			log.Fatalf("启动IPC服务失败: %v", err)
		}
	}()

	// 关键：等待IPC服务器启动完成
	<-serverStarted

	// 初始化服务发现存储
	log.Println("Initializing discovery storage...")
	discoveryStorage := discovery.NewInMemoryStorage()

	discoveryService := discovery.New(discoveryStorage)
	// 添加服务发现启动代码
	log.Println("Starting service discovery...")
	go func() {
		// 服务发现通过注册/监听自动运行，无需显式Start()
		// 可在此处添加服务注册逻辑
	}()
	defer discoveryService.Close()

	log.Println("服务已启动，按Ctrl+C退出...")
	// 在初始化discoveryService后添加以下代码
	// 注册服务发现的处理函数到IPC服务器
	transport.RegisterService("register", func(params map[string]interface{}) (interface{}, error) {
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

		// 新增：转换metadata类型
		// 修改注册处理函数中的参数检查
		metadataIface, ok := serviceData["metadata"].(map[string]interface{}) // 从serviceData获取metadata
		if !ok || metadataIface == nil {
			return nil, fmt.Errorf("metadata格式错误或为空")
		}
		metadata := make(map[string]string)
		for k, v := range metadataIface {
			metadata[k] = fmt.Sprintf("%v", v) // 将interface{}值转换为字符串
		}

		// 创建服务实例
		service := &discovery.Service{
			ID:        serviceID,
			Name:      serviceName,
			Address:   address,
			Port:      port,
			Metadata:  metadata, // 使用转换后的map[string]string
			Status:    "healthy",
			UpdatedAt: time.Now(),
			ExpireAt:  time.Now().Add(30 * time.Second),
		}

		// 调用服务发现模块注册服务
		if err := discoveryService.Register(context.Background(), service); err != nil {
			return nil, fmt.Errorf("注册失败: %v", err)
		}
		return map[string]string{"id": serviceID}, nil
	})
	<-make(chan os.Signal, 1) // 阻塞等待信号
}
