package main

import (
	"context"
	"flag"
	"fmt"
	"neo/internal/config"
	"neo/internal/core"
	"neo/internal/gateway"
	"neo/internal/ipc"
	"neo/internal/registry"
	"neo/internal/transport"
	"neo/internal/utils"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Application Neo Framework应用程序
type Application struct {
	config         config.Config
	options        ApplicationOptions
	logger         utils.Logger
	registry       registry.ServiceRegistry
	transport      transport.Transport
	ipcServer      *ipc.IPCServer
	asyncIPC       *ipc.AsyncIPCServer
	coreService    core.Service
	httpGateway    *gateway.HTTPGateway
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

// ApplicationOptions 应用程序选项
type ApplicationOptions struct {
	ConfigPath string
	LogLevel   string
	HTTPPort   string
	IPCPort    string
}

func main() {
	fmt.Println("=== Neo Framework ===")
	fmt.Println("A high-performance microservice communication framework")
	fmt.Println()

	// 解析命令行参数
	opts := parseCommandLine()
	
	// 创建应用程序
	app, err := NewApplication(opts)
	if err != nil {
		fmt.Printf("Failed to create application: %v\n", err)
		os.Exit(1)
	}

	// 初始化应用程序
	if err := app.Initialize(); err != nil {
		fmt.Printf("Failed to initialize application: %v\n", err)
		os.Exit(1)
	}

	// 启动应用程序
	if err := app.Start(); err != nil {
		fmt.Printf("Failed to start application: %v\n", err)
		os.Exit(1)
	}

	// 等待关闭信号
	app.WaitForShutdown()
}

// parseCommandLine 解析命令行参数
func parseCommandLine() ApplicationOptions {
	opts := ApplicationOptions{}
	
	flag.StringVar(&opts.ConfigPath, "config", "configs/default.yml", "配置文件路径")
	flag.StringVar(&opts.LogLevel, "log", "info", "日志级别 (debug, info, warn, error)")
	flag.StringVar(&opts.HTTPPort, "http", ":28080", "HTTP网关端口")
	flag.StringVar(&opts.IPCPort, "ipc", ":29999", "IPC服务器端口")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Neo Framework - 高性能微服务通信框架\n\n")
		fmt.Fprintf(os.Stderr, "使用方法: %s [选项]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "选项:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n示例:\n")
		fmt.Fprintf(os.Stderr, "  %s                              # 使用默认配置启动\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -http :8080 -ipc :9999      # 指定端口启动\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -log debug                   # 启用调试日志\n", os.Args[0])
	}
	
	flag.Parse()
	return opts
}

// NewApplication 创建新的应用程序实例
func NewApplication(opts ApplicationOptions) (*Application, error) {
	app := &Application{
		options: opts,
	}
	
	// 创建日志器
	app.logger = utils.DefaultLogger
	app.logger.Info("Creating Neo Framework Application")
	
	// 创建关闭上下文
	app.shutdownCtx, app.shutdownCancel = context.WithCancel(context.Background())
	
	// 加载配置
	app.config = config.Config{
		Transport: config.TransportConfig{
			Timeout:         config.Duration(30 * time.Second),
			RetryCount:      3,
			MaxConnections:  100,
			MinConnections:  10,
			MaxIdleTime:     config.Duration(300 * time.Second),
		},
	}
	
	app.logger.Info("Configuration loaded successfully")
	return app, nil
}

// Initialize 初始化应用程序组件
func (app *Application) Initialize() error {
	app.logger.Info("Initializing Neo Framework components...")
	
	// 1. 创建服务注册中心
	app.registry = registry.NewServiceRegistry(registry.WithLogger(app.logger))
	app.logger.Info("Service registry initialized")
	
	// 2. 创建传输层
	app.transport = transport.NewTransport(app.config)
	app.logger.Info("Transport layer initialized")
	
	// 3. 创建IPC服务器
	app.ipcServer = ipc.NewIPCServer(app.options.IPCPort, app.registry)
	app.logger.Info("IPC server initialized")
	
	// 4. 创建异步IPC服务器
	app.asyncIPC = ipc.NewAsyncIPCServer(app.ipcServer)
	app.logger.Info("Async IPC server initialized")
	
	// 5. 创建核心服务
	serviceOpts := core.ServiceOptions{
		Name:      "neo-gateway",
		Transport: app.transport,
		Registry:  app.registry,
		Timeout:   30 * time.Second,
		Logger:    app.logger,
		AsyncIPC:  app.asyncIPC, // 添加AsyncIPC引用
	}
	app.coreService = core.NewService(serviceOpts)
	app.logger.Info("Core service initialized")
	
	// 6. 创建HTTP网关
	app.httpGateway = gateway.NewHTTPGateway(app.coreService, app.registry, app.options.HTTPPort)
	app.logger.Info("HTTP gateway initialized")
	
	app.logger.Info("All components initialized successfully")
	return nil
}

// Start 启动应用程序
func (app *Application) Start() error {
	app.logger.Info("Starting Neo Framework services...")
	
	// 1. 启动IPC服务器
	if err := app.ipcServer.Start(); err != nil {
		return fmt.Errorf("failed to start IPC server: %w", err)
	}
	app.logger.Info("IPC server started", utils.String("address", ":29999"))
	
	// 2. 启动传输层监听器
	go func() {
		if err := app.transport.StartListener(); err != nil {
			app.logger.Error("Transport listener failed", utils.String("error", err.Error()))
		}
	}()
	app.logger.Info("Transport listener started")
	
	// 3. 启动HTTP网关
	go func() {
		if err := app.httpGateway.Start(); err != nil {
			app.logger.Error("HTTP gateway failed", utils.String("error", err.Error()))
		}
	}()
	
	// 等待服务启动
	time.Sleep(200 * time.Millisecond)
	
	// 启动成功日志
	app.logger.Info("🚀 Neo Framework started successfully!")
	app.logger.Info("📡 HTTP Gateway: http://localhost" + app.options.HTTPPort)
	app.logger.Info("🔌 IPC Server: localhost" + app.options.IPCPort)
	app.logger.Info("💚 Health Check: http://localhost" + app.options.HTTPPort + "/health")
	app.logger.Info("📖 API Endpoint: http://localhost" + app.options.HTTPPort + "/api/{service}/{method}")
	
	fmt.Println("\n=== 服务启动成功 ===")
	fmt.Printf("HTTP网关: http://localhost%s\n", app.options.HTTPPort)
	fmt.Printf("IPC服务器: localhost%s\n", app.options.IPCPort)
	fmt.Printf("健康检查: http://localhost%s/health\n", app.options.HTTPPort)
	fmt.Printf("API接口: http://localhost%s/api/{service}/{method}\n", app.options.HTTPPort)
	fmt.Println("\n按 Ctrl+C 停止服务")
	
	return nil
}

// WaitForShutdown 等待关闭信号
func (app *Application) WaitForShutdown() {
	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	
	// 等待信号
	sig := <-sigCh
	app.logger.Info("Received shutdown signal", utils.String("signal", sig.String()))
	
	// 执行关闭
	app.Shutdown()
}

// Shutdown 优雅关闭应用程序
func (app *Application) Shutdown() {
	app.logger.Info("Starting graceful shutdown...")
	
	// 创建关闭超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// 1. 停止HTTP网关
	if app.httpGateway != nil {
		if err := app.httpGateway.Stop(ctx); err != nil {
			app.logger.Error("Failed to stop HTTP gateway", utils.String("error", err.Error()))
		} else {
			app.logger.Info("HTTP gateway stopped")
		}
	}
	
	// 2. 停止传输层
	if app.transport != nil {
		if err := app.transport.StopListener(); err != nil {
			app.logger.Error("Failed to stop transport", utils.String("error", err.Error()))
		} else {
			app.logger.Info("Transport stopped")
		}
	}
	
	// 3. 停止IPC服务器
	if app.ipcServer != nil {
		if err := app.ipcServer.Stop(); err != nil {
			app.logger.Error("Failed to stop IPC server", utils.String("error", err.Error()))
		} else {
			app.logger.Info("IPC server stopped")
		}
	}
	
	// 4. 关闭核心服务
	if app.coreService != nil {
		if err := app.coreService.Close(); err != nil {
			app.logger.Error("Failed to close core service", utils.String("error", err.Error()))
		} else {
			app.logger.Info("Core service closed")
		}
	}
	
	// 取消关闭上下文
	app.shutdownCancel()
	
	app.logger.Info("🏁 Neo Framework shutdown complete")
	fmt.Println("\n=== 服务已安全关闭 ===")
}