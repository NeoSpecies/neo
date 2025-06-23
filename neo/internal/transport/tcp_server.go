package transport

import (
	"errors"
	"fmt"
	"log"
	"neo/internal/config"
	"net"
	"strconv"
	"sync"
)

// TCPServer 专注于TCP层服务管理
type TCPServer struct {
	listener   net.Listener
	config     *config.IPCConfig
	handler    *ConnectionHandler
	workerPool *WorkerPool
	mu         sync.RWMutex
	isRunning  bool
}

// NewTCPServer 创建新的TCP服务器
func NewTCPServer(cfg *config.IPCConfig, handler *ConnectionHandler, workerPool *WorkerPool) *TCPServer {
	return &TCPServer{
		config:     cfg,
		handler:    handler,
		workerPool: workerPool,
		isRunning:  false,
	}
}

// Start 启动TCP服务器
func (s *TCPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return errors.New("server already running")
	}

	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	s.listener = listener
	s.isRunning = true

	go s.acceptLoop()
	return nil
}

// Stop 停止TCP服务器
func (s *TCPServer) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return errors.New("server not running")
	}

	s.isRunning = false
	if err := s.listener.Close(); err != nil {
		return fmt.Errorf("close listener failed: %w", err)
	}
	return nil
}

// acceptLoop 接受连接循环
func (s *TCPServer) acceptLoop() {
	for {
		s.mu.RLock()
		running := s.isRunning
		s.mu.RUnlock()

		if !running {
			break
		}

		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}

		s.workerPool.Submit(func() {
			s.handler.Handle(conn)
		})
	}
}
