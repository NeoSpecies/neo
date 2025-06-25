package tcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"neo/internal/config"
	"neo/internal/ipcprotocol"
	"neo/internal/types"
)

// ServerConfig 定义TCP服务器配置
type ServerConfig struct {
	MaxConnections    int           `yaml:"max_connections"`
	MaxMsgSize        int           `yaml:"max_message_size"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	WorkerCount       int           `yaml:"worker_count"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
}

// 实现common.ServerConfig接口
func (c *ServerConfig) GetAddress() string {
	globalConfig := config.GetGlobalConfig()
	return net.JoinHostPort(globalConfig.IPC.Host, strconv.Itoa(globalConfig.IPC.Port))
}

// 实现common.ServerConfig接口
func (c *ServerConfig) GetMaxConnections() int {
	return c.MaxConnections
}

// 实现common.ServerConfig接口
func (c *ServerConfig) GetConnectionTimeout() time.Duration {
	return c.ConnectionTimeout
}

// 实现common.ServerConfig接口
func (c *ServerConfig) GetHandlerConfig() interface{} {
	return nil // 根据需要实现
}

// TCPServer 管理TCP连接和消息处理
type TCPServer struct {
	listener    net.Listener
	config      *ServerConfig
	metrics     *types.Metrics
	connections *types.TCPConnectionPool
	callback    types.MessageCallback
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	taskChan    chan func()
}

// NewServer 创建新的TCP服务器
func NewServer(
	config *types.TCPConfig,
	callback types.MessageCallback,
) (*TCPServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建任务通道
	taskChan := make(chan func(), config.WorkerCount)

	// 启动工作协程
	for i := 0; i < config.WorkerCount; i++ {
		go func() {
			for task := range taskChan {
				task()
			}
		}()
	}

	// 创建连接池配置
	poolConfig := types.Config{
		MaxSize:           int(config.MaxConnections),
		MinSize:           10,
		IdleTimeout:       30 * time.Second,
		KeepAliveInterval: 5 * time.Second,
	}

	// 创建连接池（使用正确的构造函数）
	connections := &types.TCPConnectionPool{
		Config:            poolConfig,
		MaxSize:           int(config.MaxConnections),
		MinSize:           10,
		IdleTimeout:       30 * time.Second,
		KeepAliveInterval: 5 * time.Second,
		Mu:                &sync.RWMutex{},
		Connections:       make([]*types.Connection, 0),
		Done:              make(chan struct{}),
		WaitConn:          make(chan struct{}, config.MaxConnections),
	}

	// 创建指标实例
	metrics := &types.Metrics{
		Registry: prometheus.NewRegistry(),
	}

	// 转换TCPConfig为ServerConfig
	serverConfig := &ServerConfig{
		MaxConnections:    config.MaxConnections,
		MaxMsgSize:        config.MaxMsgSize,
		ReadTimeout:       config.ReadTimeout,
		WriteTimeout:      config.WriteTimeout,
		WorkerCount:       config.WorkerCount,
		ConnectionTimeout: config.ConnectionTimeout,
	}

	return &TCPServer{
		config:      serverConfig,
		metrics:     metrics,
		connections: connections,
		callback:    callback,
		ctx:         ctx,
		cancel:      cancel,
		taskChan:    taskChan,
	}, nil
}

// Start 开始监听和接受连接
func (s *TCPServer) Start() error {
	address := s.config.GetAddress()

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %v", address, err)
	}

	s.listener = listener
	fmt.Printf("TCP server started on %s\n", address)

	s.wg.Add(1)
	go s.acceptLoop()
	return nil
}

// Stop 关闭服务器并清理资源
func (s *TCPServer) Stop() error {
	defer fmt.Println("TCP server stopped")
	defer s.cancel()

	// 关闭任务通道
	close(s.taskChan)

	// 关闭监听器
	if err := s.listener.Close(); err != nil {
		fmt.Printf("failed to close listener: %v\n", err)
	}

	// 关闭所有连接
	s.connections.Mu.Lock()
	for _, conn := range s.connections.Connections {
		if !conn.Closed {
			conn.Conn.Close()
			conn.Closed = true
		}
	}
	s.connections.Mu.Unlock()

	// 等待所有goroutine完成
	s.wg.Wait()

	return nil
}

// acceptLoop 持续接受新连接
func (s *TCPServer) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.ctx.Done():
					return
				default:
					fmt.Printf("[DEBUG] 接受连接失败: %v\n", err)
					time.Sleep(1 * time.Second)
				}
				continue
			}

			// 新增调试打印：连接信息
			remoteAddr := conn.RemoteAddr().String()
			fmt.Printf("[DEBUG] 新连接建立: %s\n", remoteAddr)

			// 检查连接限制
			s.connections.Mu.RLock()
			currentConnCount := len(s.connections.Connections)
			s.connections.Mu.RUnlock()

			if currentConnCount >= s.config.MaxConnections {
				fmt.Printf("[DEBUG] 连接数超限(%d/%d)，拒绝连接: %s\n", currentConnCount, s.config.MaxConnections, remoteAddr)
				conn.Close()
				fmt.Println("connection rejected - server is at capacity")
				continue
			}

			// 创建连接处理器
			handler := NewTCPHandler(
				conn,
				s.callback,
				s.metrics,
				s.config.MaxMsgSize,
				s.config.ReadTimeout,
				s.config.WriteTimeout,
			)

			// 添加到连接池
			s.connections.Mu.Lock()
			s.connections.Connections = append(s.connections.Connections, &types.Connection{
				Conn:     conn,
				Pool:     s.connections,
				Stats:    types.NewConnectionStats(),
				LastUsed: time.Now(),
				InUse:    true,
				Closed:   false,
			})
			s.connections.Mu.Unlock()

			// 在任务通道中处理连接
			select {
			case s.taskChan <- func() {
				defer func() {
					s.connections.Mu.Lock()
					// 从连接池移除连接
					for i, c := range s.connections.Connections {
						if c.Conn == conn {
							s.connections.Connections = append(s.connections.Connections[:i], s.connections.Connections[i+1:]...)
							break
						}
					}
					s.connections.Mu.Unlock()

					if r := recover(); r != nil {
						fmt.Printf("connection handler panicked: %v\n", r)
					}
				}()

				// 处理连接
				handler.Start()
			}:
			default:
				conn.Close()
				fmt.Println("task queue is full, connection rejected")
			}
		}
	}
}

// GetConnectionCount 返回当前活动连接数
func (s *TCPServer) GetConnectionCount() int {
	s.connections.Mu.RLock()
	defer s.connections.Mu.RUnlock()
	return len(s.connections.Connections)
}

// TCPHandler 处理单个TCP连接
type TCPHandler struct {
	Conn         net.Conn
	callback     types.MessageCallback
	metrics      *types.Metrics
	maxMsgSize   int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// NewTCPHandler 创建新的TCP连接处理器
func NewTCPHandler(
	conn net.Conn,
	callback types.MessageCallback,
	metrics *types.Metrics,
	maxMsgSize int,
	readTimeout, writeTimeout time.Duration,
) *TCPHandler {
	return &TCPHandler{
		Conn:         conn,
		callback:     callback,
		metrics:      metrics,
		maxMsgSize:   maxMsgSize,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// Start 开始处理连接
func (h *TCPHandler) Start() {
	// 创建编解码器实例
	codec := NewCodec(h.Conn, h.Conn)

	for {
		// 设置读取超时
		h.Conn.SetReadDeadline(time.Now().Add(h.readTimeout))

		// 使用codec读取并解析IPC消息
		messageFrame, err := codec.ReadIPCMessage()
		if err != nil {
			fmt.Printf("[ERROR] 消息解析失败: %v\n", err)
			return
		}

		// 将解析后的消息帧转换为JSON字节
		requestData, err := json.Marshal(messageFrame)
		if err != nil {
			fmt.Printf("[ERROR] 消息序列化失败: %v\n", err)
			return
		}

		// 调用回调处理解析后的消息
		response, err := h.callback(requestData)
		if err != nil {
			fmt.Printf("[ERROR] 消息处理失败: %v\n", err)
			// 返回结构化错误响应
			errorResp := map[string]interface{}{
				"error": map[string]string{
					"code":    "INVALID_CHECKSUM",
					"message": "校验和不匹配，请检查协议实现",
				},
			}
			response, _ = json.Marshal(errorResp)
		}

		// 设置写入超时
		h.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))

		// 发送响应
		responseFrame := &ipcprotocol.MessageFrame{
			Type:    ipcprotocol.MessageTypeResponse,
			Payload: response,
		}

		// 添加详细错误处理
		if err := codec.WriteIPCMessage(responseFrame); err != nil {
			fmt.Printf("[ERROR] 发送响应失败: %v\n", err)
			// 尝试直接写入原始响应（应急方案）
			if _, writeErr := h.Conn.Write(response); writeErr != nil {
				fmt.Printf("[ERROR] 直接写入响应失败: %v\n", writeErr)
			}
			return
		}
		fmt.Printf("[DEBUG] 响应已发送，长度: %d字节\n", len(response))
	}
}
