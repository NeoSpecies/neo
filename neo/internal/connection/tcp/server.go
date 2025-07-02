package tcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	config      *ServerConfig // 修改为具体的ServerConfig结构体指针
	metrics     *types.Metrics
	connections *types.TCPConnectionPool
	callback    types.MessageCallback
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	taskChan    chan func()
	isShutdown  int32 // 原子操作标记服务器状态
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
		isShutdown:  0,
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
	s.cancel() // 先取消上下文
	close(s.taskChan)

	// 给goroutine时间响应
	time.Sleep(100 * time.Millisecond)

	// 再关闭监听器
	if err := s.listener.Close(); err != nil {
		fmt.Printf("[INFO] 监听器关闭完成: %v\n", err)
	}

	// 关闭所有连接
	s.connections.Mu.Lock()
	for _, conn := range s.connections.Connections {
		if !conn.Closed {
			// 设置超时确保关闭操作不会阻塞
			conn.Conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
			conn.Conn.Close()
			conn.Closed = true
		}
	}
	s.connections.Mu.Unlock()

	// 等待所有goroutine完成
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		log.Println("TCP服务器已优雅关闭")
	case <-time.After(2 * time.Second):
		log.Println("警告: 服务器关闭超时，可能存在资源泄漏")
	}

	return nil
}

// acceptLoop 持续接受新连接
func (s *TCPServer) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			log.Println("接受循环已终止")
			return
		default:
			s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(500 * time.Millisecond))

			// 接受连接并保存连接对象
			conn, err := s.listener.Accept()
			if err != nil {
				// 错误处理...
				continue
			}

			// 创建连接处理器并启动处理
			handler := NewTCPHandler(
				conn,
				s.callback,
				s.metrics,
				s.config.MaxMsgSize,
				s.config.ReadTimeout,
				s.config.WriteTimeout,
			)
			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				handler.Start()
			}()
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
	codec := NewCodec(h.Conn, h.Conn)

	for {
		// 设置读取超时
		h.Conn.SetReadDeadline(time.Now().Add(h.readTimeout))

		// 使用codec读取并解析IPC消息
		msg, err := codec.ReadIPCMessage()
		if err != nil {
			if connErr, ok := err.(*ConnectionError); ok {
				if connErr.Type == ErrorTypeConnectionClosed {
					fmt.Printf("[INFO] 客户端断开连接")
				} else {
					fmt.Printf("[ERROR] 消息处理失败: %v\n", err)
				}
			} else {
				fmt.Printf("[ERROR] 未知错误: %v\n", err)
			}
			return
		}

		// 将解析后的消息帧转换为JSON字节
		requestData, err := json.Marshal(msg)
		if err != nil {
			fmt.Printf("[ERROR] 消息序列化失败: %v\n", err)
			return
		}

		// 调用回调处理解析后的消息
		// 将err重命名为callbackErr以避免遮蔽上层作用域的err变量
		if _, callbackErr := h.callback(requestData); callbackErr != nil {
			fmt.Printf("[ERROR] 消息处理失败: %v\n", callbackErr)
			// 返回结构化错误响应
			errorResp := map[string]interface{}{
				"error": map[string]string{
					"code":    "PROCESSING_ERROR",
					"message": callbackErr.Error(),
				},
			}
			// 使用不同变量名避免遮蔽
			errorResponse, marshalErr := json.Marshal(errorResp)
			if marshalErr != nil {
				fmt.Printf("[ERROR] 错误响应序列化失败: %v\n", marshalErr)
				return
			}
			// 发送错误响应
			responseFrame := &types.MessageFrame{
				Type:    ipcprotocol.MessageTypeResponse,
				Payload: errorResponse,
			}
			if writeErr := codec.WriteIPCMessage(responseFrame); writeErr != nil {
				fmt.Printf("[ERROR] 发送错误响应失败: %v\n", writeErr)
				return
			}
			continue
		}

		// 设置写入超时
		h.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))

		// 解析请求消息Payload
		var payloadMap map[string]interface{}
		if unmarshalErr := json.Unmarshal(msg.Payload, &payloadMap); unmarshalErr != nil {
			fmt.Printf("[ERROR] 解析Payload失败: %v\n", unmarshalErr)
			return
		}

		// 提取服务信息并添加错误处理
		serviceID, ok := payloadMap["service_id"].(string)
		if !ok {
			fmt.Printf("[ERROR] 无效的service_id格式\n")
			return
		}

		name, ok := payloadMap["name"].(string)
		if !ok {
			fmt.Printf("[ERROR] 无效的name格式\n")
			return
		}

		address, ok := payloadMap["address"].(string)
		if !ok {
			fmt.Printf("[ERROR] 无效的address格式\n")
			return
		}

		portVal, ok := payloadMap["port"].(float64)
		if !ok {
			fmt.Printf("[ERROR] 无效的port格式\n")
			return
		}
		port := int(portVal)

		// 构建响应数据
		responseData := map[string]interface{}{
			"result": map[string]interface{}{
				"id":      serviceID,
				"name":    name,
				"address": address,
				"port":    port,
			},
		}

		// 序列化正常响应
		normalResponse, err := json.Marshal(responseData)
		if err != nil {
			fmt.Printf("[ERROR] 响应序列化失败: %v\n", err)
			return
		}

		// 构建并发送响应帧
		responseFrame := &types.MessageFrame{
			Type:    ipcprotocol.MessageTypeResponse,
			Payload: normalResponse,
		}
		if err := codec.WriteIPCMessage(responseFrame); err != nil {
			fmt.Printf("[ERROR] 发送响应失败: %v\n", err)
			return
		}
	}
}

// Response 定义响应结构
type Response struct {
	Type string `json:"type"`
}
