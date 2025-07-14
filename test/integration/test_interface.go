package main

import (
	"context"
	"fmt"
	"neo/internal/core"
	"neo/internal/ipc"
	"neo/internal/registry"
	"neo/internal/utils"
)

func main() {
	fmt.Println("=== 测试接口实现 ===")
	
	// 创建IPC服务器
	reg := registry.NewServiceRegistry(registry.WithLogger(utils.DefaultLogger))
	ipcServer := ipc.NewIPCServer(":32999", reg)
	asyncIPC := ipc.NewAsyncIPCServer(ipcServer)
	
	// 检查类型转换
	var client core.AsyncIPCClient = asyncIPC
	fmt.Printf("✅ AsyncIPCServer 实现了 AsyncIPCClient 接口\n")
	
	// 测试方法调用（虽然会失败，但能验证方法存在）
	ctx := context.Background()
	_, err := client.ForwardRequest(ctx, "test.service", "test.method", []byte("test"))
	fmt.Printf("ForwardRequest 调用结果: %v (预期会失败)\n", err)
	
	fmt.Println("✅ 接口实现正确")
}