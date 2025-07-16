package transport

import (
	"context"
	"encoding/json"
	"fmt"
	"neo/internal/transport/conn"
	"neo/internal/types"
	"neo/internal/utils"
	"net"
	"sync"
	"time"
)

// Transport 传输层接口
type Transport interface {
	// Send 发送请求并等待响应
	Send(ctx context.Context, req types.Request) (types.Response, error)
	// SendAsync 异步发送请求
	SendAsync(ctx context.Context, req types.Request) (<-chan types.Response, error)
	// Subscribe 订阅消息
	Subscribe(pattern string, handler func(msg types.Message)) error
	// StartListener 启动监听器
	StartListener() error
	// StopListener 停止监听器
	StopListener() error
	// Close 关闭传输层
	Close() error
	// Stats 获取传输层统计信息
	Stats() TransportStats
}

// TransportStats 传输层统计信息
type TransportStats struct {
	RequestCount    int64
	ResponseCount   int64
	ErrorCount      int64
	AvgLatency      time.Duration
	ConnectionStats conn.PoolStats
}

// transportImpl 传输层实现
type transportImpl struct {
	config    Config  // 使用本地Config类型
	pool      conn.ConnectionPool
	listener  net.Listener
	logger    utils.Logger
	stats     TransportStats
	statsMu   sync.RWMutex
	handlers  map[string]func(msg types.Message)
	handlerMu sync.RWMutex
	closed    bool
	closeCh   chan struct{}
	wg        sync.WaitGroup
}

// Config 传输层配置
type Config struct {
	Timeout               time.Duration
	RetryCount            int
	MaxConnections        int
	MinConnections        int
	MaxIdleTime           time.Duration
	HealthCheckInterval   time.Duration
	ActivityCheckInterval time.Duration // 活动检查间隔
}

// NewTransport 创建传输层实例
func NewTransport(cfg Config) Transport {
	poolConfig := &conn.PoolConfig{
		MaxSize:             cfg.MaxConnections,
		MinSize:             cfg.MinConnections,
		MaxIdleTime:         cfg.MaxIdleTime,
		ConnectionTimeout:   cfg.Timeout,
		HealthCheckInterval: cfg.HealthCheckInterval,
		MaxRetries:          cfg.RetryCount,
	}

	pool := conn.NewConnectionPool(poolConfig, nil)

	return &transportImpl{
		config:   cfg, // 现在使用Transport.Config而不是config.TransportConfig
		pool:     pool,
		logger:   utils.DefaultLogger,
		handlers: make(map[string]func(msg types.Message)),
		closeCh:  make(chan struct{}),
	}
}

// Send 发送请求并等待响应
func (t *transportImpl) Send(ctx context.Context, req types.Request) (types.Response, error) {
	startTime := time.Now()
	
	// 更新统计
	t.updateStats(func(s *TransportStats) {
		s.RequestCount++
	})

	// 获取连接
	address := t.resolveAddress(req.Service)
	connection, err := t.pool.Get(ctx, address)
	if err != nil {
		t.updateStats(func(s *TransportStats) {
			s.ErrorCount++
		})
		return types.Response{}, fmt.Errorf("failed to get connection: %w", err)
	}
	defer t.pool.Put(connection)

	// 构建消息
	msg := types.Message{
		ID:        req.ID,
		Type:      types.REQUEST,
		Service:   req.Service,
		Method:    req.Method,
		Metadata:  req.Metadata,
		Body:      req.Body,
		Timestamp: time.Now(),
	}

	// 序列化消息
	data, err := json.Marshal(msg)
	if err != nil {
		t.updateStats(func(s *TransportStats) {
			s.ErrorCount++
		})
		return types.Response{}, fmt.Errorf("failed to marshal message: %w", err)
	}

	// 发送数据
	if err := connection.Send(ctx, data); err != nil {
		t.updateStats(func(s *TransportStats) {
			s.ErrorCount++
		})
		return types.Response{}, fmt.Errorf("failed to send data: %w", err)
	}

	// 接收响应
	respData, err := connection.Receive(ctx)
	if err != nil {
		t.updateStats(func(s *TransportStats) {
			s.ErrorCount++
		})
		return types.Response{}, fmt.Errorf("failed to receive response: %w", err)
	}

	// 反序列化响应
	var respMsg types.Message
	if err := json.Unmarshal(respData, &respMsg); err != nil {
		t.updateStats(func(s *TransportStats) {
			s.ErrorCount++
		})
		return types.Response{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 构建响应
	response := types.Response{
		ID:       respMsg.ID,
		Body:     respMsg.Body,
		Metadata: respMsg.Metadata,
	}

	// 更新统计
	latency := time.Since(startTime)
	t.updateStats(func(s *TransportStats) {
		s.ResponseCount++
		s.AvgLatency = (s.AvgLatency + latency) / 2
	})

	return response, nil
}

// SendAsync 异步发送请求
func (t *transportImpl) SendAsync(ctx context.Context, req types.Request) (<-chan types.Response, error) {
	respCh := make(chan types.Response, 1)
	
	go func() {
		defer close(respCh)
		
		resp, err := t.Send(ctx, req)
		if err != nil {
			t.logger.Error("async send failed", utils.String("error", err.Error()))
			return
		}
		
		select {
		case respCh <- resp:
		case <-ctx.Done():
		}
	}()
	
	return respCh, nil
}

// Subscribe 订阅消息
func (t *transportImpl) Subscribe(pattern string, handler func(msg types.Message)) error {
	t.handlerMu.Lock()
	defer t.handlerMu.Unlock()
	
	t.handlers[pattern] = handler
	return nil
}

// StartListener 启动监听器
func (t *transportImpl) StartListener() error {
	// 使用随机端口避免冲突
	address := "127.0.0.1:0"

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}

	t.listener = listener
	t.logger.Info("transport listener started", utils.String("address", address))

	// 启动接受连接的goroutine
	t.wg.Add(1)
	go t.acceptConnections()

	return nil
}

// StopListener 停止监听器
func (t *transportImpl) StopListener() error {
	if t.listener != nil {
		t.listener.Close()
		t.listener = nil
	}
	return nil
}

// Close 关闭传输层
func (t *transportImpl) Close() error {
	if t.closed {
		return nil
	}
	
	t.closed = true
	close(t.closeCh)
	
	// 停止监听器
	t.StopListener()
	
	// 关闭连接池
	if err := t.pool.Close(); err != nil {
		t.logger.Error("failed to close connection pool", utils.String("error", err.Error()))
	}
	
	// 等待后台任务完成
	t.wg.Wait()
	
	t.logger.Info("transport closed")
	return nil
}

// Stats 获取传输层统计信息
func (t *transportImpl) Stats() TransportStats {
	t.statsMu.RLock()
	stats := t.stats
	stats.ConnectionStats = t.pool.Stats()
	t.statsMu.RUnlock()
	return stats
}

// acceptConnections 接受客户端连接
func (t *transportImpl) acceptConnections() {
	defer t.wg.Done()
	
	for {
		select {
		case <-t.closeCh:
			return
		default:
		}
		
		if t.listener == nil {
			t.logger.Error("listener is nil")
			return
		}
		
		conn, err := t.listener.Accept()
		if err != nil {
			if t.closed {
				return
			}
			t.logger.Error("failed to accept connection", utils.String("error", err.Error()))
			continue
		}
		
		// 处理客户端连接
		t.wg.Add(1)
		go t.handleConnection(conn)
	}
}

// handleConnection 处理客户端连接
func (t *transportImpl) handleConnection(netConn net.Conn) {
	defer t.wg.Done()
	defer netConn.Close()
	
	// 包装为连接接口
	id := fmt.Sprintf("server-conn-%d", time.Now().UnixNano())
	tcpConn := conn.NewTCPConnection(netConn, id, t.config.Timeout, t.config.Timeout)
	// 如果有活动检查间隔配置，设置它
	if activityInterval := t.getActivityCheckInterval(); activityInterval > 0 {
		tcpConn.SetActivityCheckInterval(activityInterval)
	}
	connection := tcpConn
	
	t.logger.Info("client connected", 
		utils.String("remoteAddr", connection.RemoteAddr()),
		utils.String("connID", connection.ID()))
	
	for {
		select {
		case <-t.closeCh:
			return
		default:
		}
		
		// 接收消息
		ctx, cancel := context.WithTimeout(context.Background(), t.config.Timeout)
		data, err := connection.Receive(ctx)
		cancel()
		
		if err != nil {
			t.logger.Debug("connection receive error", utils.String("error", err.Error()))
			return
		}
		
		// 处理消息
		t.handleMessage(connection, data)
	}
}

// handleMessage 处理接收到的消息
func (t *transportImpl) handleMessage(connection conn.Connection, data []byte) {
	// 反序列化消息
	var msg types.Message
	if err := json.Unmarshal(data, &msg); err != nil {
		t.logger.Error("failed to unmarshal message", utils.String("error", err.Error()))
		return
	}
	
	t.logger.Debug("received message",
		utils.String("type", fmt.Sprintf("%d", msg.Type)),
		utils.String("service", msg.Service),
		utils.String("method", msg.Method))
	
	// 查找匹配的处理器
	t.handlerMu.RLock()
	handler, exists := t.handlers[msg.Service]
	t.handlerMu.RUnlock()
	
	if exists {
		// 异步处理消息
		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.logger.Error("message handler panic", 
						utils.String("service", msg.Service),
						utils.String("panic", fmt.Sprintf("%v", r)))
				}
			}()
			
			handler(msg)
		}()
	} else {
		t.logger.Warn("no handler found for service", utils.String("service", msg.Service))
	}
}

// resolveAddress 解析服务地址
func (t *transportImpl) resolveAddress(service string) string {
	// 这里应该通过服务注册中心解析地址
	// 暂时返回IPC服务器地址作为占位符
	return "127.0.0.1:30999"
}

// updateStats 更新统计信息
func (t *transportImpl) updateStats(fn func(*TransportStats)) {
	t.statsMu.Lock()
	defer t.statsMu.Unlock()
	fn(&t.stats)
}

// getActivityCheckInterval 获取活动检查间隔
// 如果未配置，返回0
func (t *transportImpl) getActivityCheckInterval() time.Duration {
	if t.config.ActivityCheckInterval > 0 {
		return t.config.ActivityCheckInterval
	}
	return 0
}
