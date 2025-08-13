package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// 命令行参数
	var (
		ipcAddr  = flag.String("ipc", "localhost:9999", "IPC server address")
		tcpAddr  = flag.String("tcp", ":7777", "TCP listen address")
		protocol = flag.String("protocol", "json", "Protocol type (json or binary)")
		help     = flag.Bool("help", false, "Show help")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "TCP Gateway Service for Neo Framework\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nProtocol types:\n")
		fmt.Fprintf(os.Stderr, "  json   - JSON protocol with length prefix (default)\n")
		fmt.Fprintf(os.Stderr, "  binary - Binary protocol (future extension)\n")
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -ipc localhost:9999 -tcp :7777 -protocol json\n", os.Args[0])
	}
	
	flag.Parse()
	
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	// 验证协议类型
	if *protocol != "json" && *protocol != "binary" {
		fmt.Fprintf(os.Stderr, "Error: Invalid protocol '%s'. Must be 'json' or 'binary'\n", *protocol)
		os.Exit(1)
	}

	fmt.Println("=== TCP Gateway Service ===")
	fmt.Println("Neo Framework TCP Gateway")
	fmt.Printf("Protocol: %s\n", *protocol)
	fmt.Println()

	// 创建服务
	service := NewTCPGatewayService(*tcpAddr, *protocol)

	// 连接到IPC服务器
	fmt.Printf("Connecting to IPC server at %s...\n", *ipcAddr)
	if err := service.ConnectToIPC(*ipcAddr); err != nil {
		log.Fatalf("Failed to connect to IPC server: %v", err)
	}
	fmt.Println("✓ Connected to IPC server")

	// 启动TCP服务器
	fmt.Printf("Starting TCP server on %s...\n", *tcpAddr)
	
	// 在goroutine中启动TCP服务器
	errChan := make(chan error, 1)
	go func() {
		if err := service.StartTCPServer(); err != nil {
			errChan <- err
		}
	}()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号或错误
	select {
	case err := <-errChan:
		log.Fatalf("TCP server error: %v", err)
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
		if err := service.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		fmt.Println("TCP Gateway Service stopped")
	}
}