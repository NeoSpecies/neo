package main

import (
	"context"
	"fmt"
	"neo/internal/registry"
	"neo/internal/utils"
)

func main() {
	// 创建一个测试注册中心
	reg := registry.NewServiceRegistry(registry.WithLogger(utils.DefaultLogger))
	
	// 创建测试服务实例
	instance := &registry.ServiceInstance{
		ID:       "test-python.math-127.0.0.1:54321",
		Name:     "python.math",
		Address:  "127.0.0.1:54321",
		Metadata: map[string]string{"version": "1.0.0"},
	}
	
	// 注册服务
	ctx := context.Background()
	if err := reg.Register(ctx, instance); err != nil {
		fmt.Printf("注册失败: %v\n", err)
		return
	}
	
	fmt.Printf("服务注册成功: %s\n", instance.Name)
	
	// 尝试发现服务
	instances, err := reg.Discover(ctx, "python.math")
	if err != nil {
		fmt.Printf("服务发现失败: %v\n", err)
		return
	}
	
	fmt.Printf("找到 %d 个服务实例:\n", len(instances))
	for i, inst := range instances {
		fmt.Printf("  %d. ID: %s, Address: %s, Status: %s\n", 
			i+1, inst.ID, inst.Address, inst.Status)
	}
	
	// 列出所有服务
	services, err := reg.ListServices(ctx)
	if err != nil {
		fmt.Printf("列出服务失败: %v\n", err)
		return
	}
	
	fmt.Printf("所有已注册的服务:\n")
	for _, service := range services {
		fmt.Printf("  - %s\n", service)
	}
}