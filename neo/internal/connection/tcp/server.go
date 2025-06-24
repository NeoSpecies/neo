package tcp

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"neo/internal/common"
	"neo/internal/connection"
	"neo/internal/metrics"
)

// TCP服务器配置
type ServerConfig struct {
	Address           string
	MaxConnections    int
	ConnectionTimeout time.Duration
	HandlerConfig     *connection.Config
}

// 实现common.ServerConfig接口
func (c *ServerConfig) GetAddress() string {
	return c.Address
}

func (c *ServerConfig) GetMaxConnections() int {
	return c.MaxConnections
}

func (c *ServerConfig) GetConnectionTimeout() time.Duration {
	return c.ConnectionTimeout
}

func (c *ServerConfig) GetHandlerConfig() interface{} {
	return c.HandlerConfig
}

// TCP服务器
type Server struct {
	config          *ServerConfig
	listener        net.Listener
	handler         *ConnectionHandler
	serviceRegistry common.ServiceRegistry
	workerPool      common.WorkerPool
	metrics         *metrics.Metrics

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	mu                sync.Mutex
	activeConnections int
	started           bool
}

// NewServer 创建新的TCP服务器
func NewServer(
	config *ServerConfig,
	serviceRegistry common.ServiceRegistry,
	workerPool common.WorkerPool,
	metrics *metrics.Metrics,
) (*Server, error) {
	// 验证配置
	if config.Address == "" {
		return nil, errors.New("服务器地址不能为空")
	}
	if config.MaxConnections <= 0 {
		config.MaxConnections = 100 // 默认最大连接数
	}
	if config.HandlerConfig == nil {
		return nil, errors.New("处理器配置不能为空")
	}

	// 创建连接池
	// 修复：实现真正的TCP连接工厂函数
	poolFactory := func() (net.Conn, error) {
		// 从配置获取目标服务器地址（此处假设为远程服务器地址，需根据实际情况调整）
		// 注意：实际应用中应从配置或参数中获取目标地址
		targetAddr := "127.0.0.1:9090"
		// 修复字段名拼写错误：ConnectTimeout -> ConnectionTimeout
		conn, err := net.DialTimeout("tcp", targetAddr, config.ConnectionTimeout)
		if err != nil {
			return nil, fmt.Errorf("连接目标服务器失败: %w", err)
		}
		return conn, nil
	}
	connectionPool, err := connection.NewTCPConnectionPool(poolFactory)
	if err != nil {
		return nil, fmt.Errorf("创建连接池失败: %w", err)
	}

	// 创建连接处理器
	handler := NewConnectionHandler(config.HandlerConfig, connectionPool)

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		config:            config,
		handler:           handler,
		serviceRegistry:   serviceRegistry,
		workerPool:        workerPool,
		metrics:           metrics,
		activeConnections: 0,
		listener:          nil,
		started:           false,
		mu:                sync.Mutex{},
		ctx:               ctx,
		cancel:            cancel,
	}, nil
}

// 启动服务器
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return errors.New("服务器已启动")
	}

	// 创建监听器
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("创建TCP监听器失败: %w", err)
	}

	s.listener = listener
	s.started = true

	log.Printf("TCP服务器已启动，监听地址: %s", s.config.Address)

	// 启动接受连接的协程
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// 接受连接
func (s *Server) acceptConnections() {
	defer func() {
		s.wg.Done()
		log.Println("TCP连接接受循环已退出")
	}()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			// 接受新连接
			conn, err := s.listener.Accept()
			if err != nil {
				// 检查是否是关闭错误
				select {
				case <-s.ctx.Done():
					return
				default:
					log.Printf("接受TCP连接失败: %v", err)
					// 短暂延迟后重试
					time.Sleep(100 * time.Millisecond)
					continue
				}
			}

			// 设置连接超时
			if s.config.ConnectionTimeout > 0 {
				conn.SetDeadline(time.Now().Add(s.config.ConnectionTimeout))
			}

			// 检查连接数限制
			s.mu.Lock()
			if s.activeConnections >= s.config.MaxConnections {
				s.mu.Unlock()
				conn.Close()
				// 记录连接拒绝指标 - 修复metrics调用
				if s.metrics != nil {
					s.metrics.RecordError("tcp", "connection", "refused")
				}
				log.Printf("已达到最大连接数限制: %d", s.config.MaxConnections)
				continue
			}
			s.activeConnections++
			s.mu.Unlock()

			// 处理连接
			go func() {
				defer func() {
					s.mu.Lock()
					s.activeConnections--
					s.mu.Unlock()
				}()
				s.handler.HandleConnection(conn)
			}()
		}
	}
}

// 优雅关闭服务器
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return errors.New("服务器未启动")
	}

	log.Println("正在关闭TCP服务器...")

	// 取消上下文
	s.cancel()

	// 关闭监听器
	if s.listener != nil {
		s.listener.Close()
	}

	// 等待所有协程退出
	s.wg.Wait()

	s.started = false
	log.Println("TCP服务器已关闭")
	return nil
}

// 获取当前活动连接数
func (s *Server) ActiveConnections() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeConnections
}

// 获取服务器配置
func (s *Server) Config() *ServerConfig {
	return s.config
}

// 判断服务器是否已启动
func (s *Server) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}
