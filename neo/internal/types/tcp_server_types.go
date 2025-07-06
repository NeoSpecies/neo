/*
 * 描述: 定义TCP服务器和IPC服务器的核心类型，包括服务器配置、连接管理和请求处理相关结构体与接口
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
)

// IPCServerConfig IPC服务器配置结构体
// 包含TCP配置和工作池参数
// +---------------------+-------------------+-----------------------------------+
// | 字段名               | 类型              | 描述                              |
// +---------------------+-------------------+-----------------------------------+
// | TCPConfig           | TCPConfig         | TCP连接基础配置                    |
// | WorkerPoolSize      | int               | 工作池大小                         |
// | WorkerQueueSize     | int               | 工作队列大小                       |
// +---------------------+-------------------+-----------------------------------+
type IPCServerConfig struct {
	TCPConfig       TCPConfig
	WorkerPoolSize  int
	WorkerQueueSize int
}

// Start 启动工作池
// 注意：接口中没有Start方法，需要移除或调整设计
// +----------------+-----------------------------------+
// | 返回值         | 描述                               |
// +----------------+-----------------------------------+
// | 无             | 启动工作池协程                      |
// +----------------+-----------------------------------+
func (a *WorkerPoolAdapter) Start() {
	// 接口中没有Start方法，需要移除或调整设计
}

// Stop 停止工作池
// 实现WorkerPool接口的Stop方法
// +----------------+-----------------------------------+
// | 返回值         | 描述                               |
// +----------------+-----------------------------------+
// | 无             | 停止工作池并释放资源                |
// +----------------+-----------------------------------+
func (a *WorkerPoolAdapter) Stop() {
	a.WorkerPool.Stop()
}

// SetWorkerCount 设置工作者数量
// 实现WorkerPool接口的SetWorkerCount方法
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | count          | 工作者数量                         |
// | 返回值         | 无                                 |
// +----------------+-----------------------------------+
func (a *WorkerPoolAdapter) SetWorkerCount(count int) {
	a.WorkerPool.SetWorkerCount(count)
}

// Shutdown 关闭工作池
// 实现WorkerPool接口的Shutdown方法
// +----------------+-----------------------------------+
// | 返回值         | 描述                               |
// +----------------+-----------------------------------+
// | 无             | 优雅关闭工作池                      |
// +----------------+-----------------------------------+
func (a *WorkerPoolAdapter) Shutdown() {
	a.WorkerPool.Shutdown()
}

// IPCServer IPC服务器结构体
// 用于处理进程间通信的服务器实现
// 注意：已删除所有未使用的字段

type IPCServer struct {
	// 删除所有未使用的字段
}

// NewIPCServerConfigFromGlobal 从全局配置创建IPC服务器配置
// 根据全局配置转换为IPC服务器专用配置
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | globalConfig   | 全局配置实例                      |
// | 返回值         | 转换后的IPCServerConfig实例       |
// +----------------+-----------------------------------+
func NewIPCServerConfigFromGlobal(globalConfig *GlobalConfig) IPCServerConfig {
	return IPCServerConfig{
		TCPConfig: TCPConfig{
			MaxConnections:    globalConfig.IPC.MaxConnections,
			MaxMsgSize:        globalConfig.Protocol.MaxMessageSize,
			ReadTimeout:       globalConfig.IPC.ReadTimeout,
			WriteTimeout:      globalConfig.IPC.WriteTimeout,
			WorkerCount:       globalConfig.IPC.WorkerCount,
			ConnectionTimeout: globalConfig.IPC.ConnectionTimeout,
		},
		WorkerPoolSize:  10,
		WorkerQueueSize: 100,
	}
}

// TCPServer TCP服务器结构体
// 管理TCP连接、请求处理和服务器生命周期
// +---------------------+-------------------+-----------------------------------+
// | 字段名              | 类型              | 描述                              |
// +---------------------+-------------------+-----------------------------------+
// | Listener            | net.Listener      | TCP监听器实例                     |
// | Config              | *TCPConfig        | TCP配置指针                       |
// | Metrics             | *Metrics          | 指标收集器实例                    |
// | Connections         | *TCPConnectionPool| 连接池实例                        |
// | Callback            | MessageCallback   | 消息回调函数                      |
// | wg                  | sync.WaitGroup    | 等待组，用于优雅关闭              |
// | ctx                 | context.Context   | 上下文，用于取消操作              |
// | cancel              | context.CancelFunc| 取消函数，用于终止上下文          |
// | taskChan            | chan func()       | 任务通道                          |
// | isShutdown          | int32             | 原子操作标记，指示服务器状态      |
// | handlerFactory      | TCPHandlerFactory | 处理器工厂，用于创建连接处理器    |
// +---------------------+-------------------+-----------------------------------+
type TCPServer struct {
	Listener       net.Listener       // 首字母大写导出
	Config         *TCPConfig         // 首字母大写导出
	Metrics        *Metrics           // 首字母大写导出
	Connections    *TCPConnectionPool // 首字母大写导出
	Callback       MessageCallback    // 首字母大写导出
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	taskChan       chan func()
	isShutdown     int32             // 原子操作标记服务器状态
	handlerFactory TCPHandlerFactory // 新增：处理器工厂
}

// Start 启动TCP服务器
// 实现Server接口，创建监听器并开始接受连接
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | error          | 启动过程中的错误，nil表示成功     |
// +----------------+-----------------------------------+
func (s *TCPServer) Start() error {
	// 创建TCP监听器
	listener, err := net.Listen("tcp", s.Config.Address)
	if err != nil {
		return fmt.Errorf("创建TCP监听器失败: %w", err)
	}
	s.Listener = listener

	// 启动接受连接循环
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.acceptLoop()
	}()

	return nil
}

// Stop 停止TCP服务器
// 关闭监听器并优雅终止所有协程
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | error          | 停止过程中的错误，nil表示成功     |
// +----------------+-----------------------------------+
func (s *TCPServer) Stop() error {
	if s.Listener != nil {
		s.Listener.Close()
	}
	s.cancel()
	s.wg.Wait()
	return nil
}

// acceptLoop 接受连接循环
// 持续接受新连接并使用处理器工厂创建处理器
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | 无             | 持续运行直到服务器停止            |
// +----------------+-----------------------------------+
func (s *TCPServer) acceptLoop() {
	for {
		conn, err := s.Listener.Accept() // 修正为大写Listener
		if err != nil {
			select {
			case <-s.ctx.Done():
				// 正常关闭，不记录错误
				return
			default:
				// 移除未定义的Metrics.RecordError调用
				log.Printf("[ERROR] 接受连接失败: %v", err)
				return
			}
		}
		// 使用工厂创建处理器
		handler := s.handlerFactory(conn)
		go handler.Start()
	}
}

// NewTCPServer 创建新的TCP服务器实例
// 初始化服务器配置和状态
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | config         | TCP配置指针                       |
// | callback       | 消息回调函数                      |
// | metrics        | 指标收集器实例                    |
// | connections    | 连接池实例                        |
// | ctx            | 上下文对象                        |
// | factory        | 处理器工厂函数                    |
// | 返回值         | 新创建的TCPServer实例             |
// +----------------+-----------------------------------+
func NewTCPServer(config *TCPConfig, callback MessageCallback, metrics *Metrics, connections *TCPConnectionPool, ctx context.Context, factory TCPHandlerFactory) *TCPServer {
	ctx, cancel := context.WithCancel(ctx)
	return &TCPServer{
		Config:         config,
		Callback:       callback,
		Metrics:        metrics,
		Connections:    connections,
		ctx:            ctx,
		cancel:         cancel,
		taskChan:       make(chan func(), 100),
		isShutdown:     0,
		handlerFactory: factory,
	}
}
