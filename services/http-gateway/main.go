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
		httpAddr = flag.String("http", ":8080", "HTTP listen address")
		help     = flag.Bool("help", false, "Show help")
	)
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "HTTP Gateway Service for Neo Framework\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -ipc localhost:9999 -http :8080\n", os.Args[0])
	}
	
	flag.Parse()
	
	if *help {
		flag.Usage()
		os.Exit(0)
	}

	fmt.Println("=== HTTP Gateway Service ===")
	fmt.Println("Neo Framework HTTP Gateway")
	fmt.Println()

	// 创建服务
	service := NewHTTPGatewayService(*httpAddr)

	// 连接到IPC服务器
	fmt.Printf("Connecting to IPC server at %s...\n", *ipcAddr)
	if err := service.ConnectToIPC(*ipcAddr); err != nil {
		log.Fatalf("Failed to connect to IPC server: %v", err)
	}
	fmt.Println("✓ Connected to IPC server")

	// 启动HTTP服务器
	fmt.Printf("Starting HTTP server on %s...\n", *httpAddr)
	
	// 在goroutine中启动HTTP服务器
	errChan := make(chan error, 1)
	go func() {
		if err := service.StartHTTPServer(); err != nil {
			errChan <- err
		}
	}()

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 等待信号或错误
	select {
	case err := <-errChan:
		log.Fatalf("HTTP server error: %v", err)
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %v, shutting down...\n", sig)
		if err := service.Stop(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		fmt.Println("HTTP Gateway Service stopped")
	}
}