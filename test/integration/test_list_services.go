package main

import (
	"context"
	"fmt"
	"neo/internal/config"
	"neo/internal/core"
	"neo/internal/ipc"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/utils"
	"time"
)

func main() {
	fmt.Println("=== 连接到运行中的Neo Framework并列出服务 ===")
	
	// 创建一个新的注册中心（注意：这不会访问运行中的框架的注册中心）
	// 我们需要通过其他方式来测试
	
	// 创建IPC客户端来测试连接
	fmt.Println("连接到IPC服务器...")
	
	// 首先测试连接
	cfg := config.Config{
		Transport: config.TransportConfig{
			Timeout: config.Duration(30 * time.Second),
		},
	}
	
	reg := registry.NewServiceRegistry(registry.WithLogger(utils.DefaultLogger))
	transport := transport.NewTransport(cfg)
	ipcServer := ipc.NewIPCServer(":33999", reg)
	asyncIPC := ipc.NewAsyncIPCServer(ipcServer)
	
	// 创建核心服务来测试
	serviceOpts := core.ServiceOptions{
		Name:      "test-client",
		Transport: transport,
		Registry:  reg,
		Timeout:   10 * time.Second,
		Logger:    utils.DefaultLogger,
		AsyncIPC:  asyncIPC,
	}
	_ = core.NewService(serviceOpts) // 仅用于测试
	
	// 手动注册一个测试服务来验证注册中心工作
	testInstance := &registry.ServiceInstance{
		ID:       "manual-test-service",
		Name:     "manual.test",
		Address:  "127.0.0.1:12345",
		Metadata: map[string]string{"manual": "true"},
	}
	
	ctx := context.Background()
	if err := reg.Register(ctx, testInstance); err != nil {
		fmt.Printf("❌ 手动注册失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 手动注册服务成功: %s\n", testInstance.Name)
	
	// 列出所有服务
	services, err := reg.ListServices(ctx)
	if err != nil {
		fmt.Printf("❌ 列出服务失败: %v\n", err)
		return
	}
	
	fmt.Printf("📋 当前注册的服务数量: %d\n", len(services))
	for i, service := range services {
		fmt.Printf("  %d. %s\n", i+1, service)
		
		// 获取服务实例详情
		instances, err := reg.Discover(ctx, service)
		if err != nil {
			fmt.Printf("     发现失败: %v\n", err)
			continue
		}
		
		fmt.Printf("     实例数量: %d\n", len(instances))
		for j, inst := range instances {
			fmt.Printf("       %d.%d ID: %s, Address: %s\n", i+1, j+1, inst.ID, inst.Address)
		}
	}
	
	fmt.Println("\n⚠️  注意：这个测试使用独立的注册中心，不能看到运行中的Neo Framework的服务")
	fmt.Println("如果要查看实际的注册状态，需要在运行的框架中添加调试输出")
}