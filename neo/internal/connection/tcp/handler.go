package tcp

import (
	"fmt"
	"log"
	"net"
	"time"

	"neo/internal/connection"
	"neo/internal/ipcprotocol"
	"neo/internal/types"
)

// ConnectionHandler 处理TCP连接的生命周期和消息流转
type ConnectionHandler struct {
	config         *types.Config            // 修正为 types.Config
	connectionPool *types.TCPConnectionPool // 修正为 types.TCPConnectionPool
}

// NewConnectionHandler 创建新的连接处理器
func NewConnectionHandler(config *types.Config, pool *types.TCPConnectionPool) *ConnectionHandler { // 更新参数类型
	return &ConnectionHandler{
		config:         config,
		connectionPool: pool,
	}
}

// HandleConnection 处理新建立的TCP连接
func (h *ConnectionHandler) HandleConnection(conn net.Conn) error {
	// 将net.Conn转换为*net.TCPConn以访问TCP特定方法
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("无效的连接类型，期望*net.TCPConn")
	}

	// 配置TCP连接参数
	// 使用连接超时作为读写缓冲区大小的替代配置
	if h.config.ConnectTimeout > 0 {
		bufferSize := int(h.config.ConnectTimeout.Milliseconds())
		if bufferSize > 0 {
			if err := tcpConn.SetReadBuffer(bufferSize); err != nil {
				log.Printf("设置读取缓冲区大小失败: %v", err)
			}
			if err := tcpConn.SetWriteBuffer(bufferSize); err != nil {
				log.Printf("设置写入缓冲区大小失败: %v", err)
			}
		}
	}

	// 设置TCP保持连接
	if err := tcpConn.SetKeepAlive(true); err != nil {
		log.Printf("设置TCP保持连接失败: %v", err)
	} else {
		// 设置保持连接间隔
		if h.config.KeepAliveInterval > 0 {
			if err := tcpConn.SetKeepAlivePeriod(h.config.KeepAliveInterval); err != nil {
				log.Printf("设置TCP保持连接间隔失败: %v", err)
			}
		}
	}

	// 设置连接超时
	if h.config.IdleTimeout > 0 {
		if err := conn.SetReadDeadline(time.Now().Add(h.config.IdleTimeout)); err != nil {
			return fmt.Errorf("设置读超时失败: %v", err)
		}
	}

	if h.config.IdleTimeout > 0 {
		if err := conn.SetWriteDeadline(time.Now().Add(h.config.IdleTimeout)); err != nil {
			return fmt.Errorf("设置写超时失败: %v", err)
		}
	}

	// 创建带统计功能的连接 - 使用连接池的Connection类型
	// 从连接池获取一个连接包装器
	poolConn, err := connection.Acquire(h.connectionPool) // 修改为使用connection.Acquire
	if err != nil {
		return fmt.Errorf("创建连接池连接失败: %v", err)
	}
	// 将原始连接替换为配置好的TCP连接
	poolConn.Conn = tcpConn
	poolConn.Stats = types.NewConnectionStats() // 修改为使用types包的构造函数

	// 添加到连接池 - 使用正确的Release方法和参数
	connection.Release(h.connectionPool, poolConn, nil) // 添加第三个error参数
	defer func() {
		// 从连接池移除并关闭连接
		conn.Close()
	}()

	// 消息处理循环
	codec := ipcprotocol.NewCodec(conn, conn)
	for {
		// 读取消息
		msgFrame, err := codec.ReadFrame()
		if err != nil {
			// 检查是否为超时错误
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// 超时错误处理 - 继续等待新数据
				continue
			}
			return h.handleConnectionError(err)
		}

		// 成功读取后重置读超时
		if h.config.IdleTimeout > 0 {
			if readErr := conn.SetReadDeadline(time.Now().Add(h.config.IdleTimeout)); readErr != nil {
				return fmt.Errorf("重置读超时失败: %v", readErr)
			}
		}

		// 处理消息
		responseFrame, err := h.processMessageFrame(msgFrame)
		if err != nil {
			return h.handleConnectionError(err)
		}

		// 发送响应（如果有）
		if responseFrame != nil {
			if err := codec.WriteFrame(responseFrame); err != nil {
				return h.handleConnectionError(err)
			}
		}
	}
}

// processMessageFrame 处理接收到的消息帧
func (h *ConnectionHandler) processMessageFrame(frame *types.MessageFrame) (*types.MessageFrame, error) {
	// 实现消息处理逻辑
	// 此处为示例实现，实际应根据业务需求处理
	return &types.MessageFrame{
		Type:    ipcprotocol.MessageTypeResponse,
		Payload: []byte("已处理: " + string(frame.Payload)),
	}, nil
}

// handleConnectionError 处理连接错误
func (h *ConnectionHandler) handleConnectionError(err error) error {
	// 记录错误信息
	log.Printf("连接错误: %v", err)

	// 返回错误
	return fmt.Errorf("连接错误: %v", err)
}
