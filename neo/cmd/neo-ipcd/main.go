package main

import (
	"fmt"
	"log"
	"neo/internal/config"
	"neo/internal/config/loader"
	"neo/internal/discovery"
	"neo/internal/transport"
	"os"
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
	// 移除：无需重新获取配置，直接使用已加载的cfg变量
	err = transport.StartIpcServer(fmt.Sprintf("%s:%d", cfg.IPC.Host, cfg.IPC.Port))
	if err != nil {
		log.Fatalf("Failed to start IPC server: %v", err)
	}

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
	<-make(chan os.Signal, 1) // 阻塞等待信号
}
