package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// RequestHandler 请求处理器，管理异步请求-响应
type RequestHandler struct {
	pendingRequests sync.Map // requestID -> chan *IPCMessage
	mu              sync.RWMutex
}

// NewRequestHandler 创建新的请求处理器
func NewRequestHandler() *RequestHandler {
	return &RequestHandler{}
}

// SendRequestAsync 发送异步请求并等待响应
func (h *RequestHandler) SendRequestAsync(ctx context.Context, conn interface{}, msg *IPCMessage) (*IPCMessage, error) {
	// 创建响应通道
	respChan := make(chan *IPCMessage, 1)
	h.pendingRequests.Store(msg.ID, respChan)
	defer h.pendingRequests.Delete(msg.ID)

	// 发送请求
	if c, ok := conn.(MessageWriter); ok {
		if err := c.WriteMessage(msg); err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}
	} else {
		return nil, fmt.Errorf("invalid connection type")
	}

	// 等待响应
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-respChan:
		return resp, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// HandleResponse 处理收到的响应
func (h *RequestHandler) HandleResponse(msg *IPCMessage) {
	if ch, ok := h.pendingRequests.Load(msg.ID); ok {
		if respChan, ok := ch.(chan *IPCMessage); ok {
			select {
			case respChan <- msg:
			default:
				// 通道已满或已关闭
			}
		}
	}
}

// MessageWriter 消息写入接口
type MessageWriter interface {
	WriteMessage(msg *IPCMessage) error
}

// AsyncIPCServer 支持异步通信的IPC服务器
type AsyncIPCServer struct {
	*IPCServer
	requestHandler *RequestHandler
}

// NewAsyncIPCServer 创建支持异步的IPC服务器
func NewAsyncIPCServer(ipcServer *IPCServer) *AsyncIPCServer {
	asyncServer := &AsyncIPCServer{
		IPCServer:      ipcServer,
		requestHandler: NewRequestHandler(),
	}
	
	// 设置IPC服务器的异步处理器
	ipcServer.SetAsyncHandler(asyncServer.requestHandler)
	
	return asyncServer
}

// ForwardRequest 转发请求到目标服务并等待响应
func (s *AsyncIPCServer) ForwardRequest(ctx context.Context, serviceName string, method string, data []byte) ([]byte, error) {
	// 调试：打印查找的服务名
	fmt.Printf("ForwardRequest: Looking for service '%s', method '%s'\n", serviceName, method)
	
	// 调试：打印所有已注册的服务
	fmt.Printf("Available services in handlers:\n")
	count := 0
	s.handlers.Range(func(key, value interface{}) bool {
		fmt.Printf("  - %v (type: %T)\n", key, key)
		count++
		return true
	})
	fmt.Printf("Total services: %d\n", count)
	
	// 查找目标服务连接
	connInterface, ok := s.handlers.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	// 构建请求消息
	msg := &IPCMessage{
		Type:     TypeRequest,
		ID:       generateRequestID(),
		Service:  serviceName,
		Method:   method,
		Data:     data,
		Metadata: make(map[string]string),
	}

	// 发送请求并等待响应
	conn := &ConnWrapper{conn: connInterface, server: s}
	resp, err := s.requestHandler.SendRequestAsync(ctx, conn, msg)
	if err != nil {
		return nil, err
	}

	// 检查响应是否包含错误
	if resp.Metadata["error"] == "true" {
		var errorData struct {
			Error string `json:"error"`
		}
		if err := json.Unmarshal(resp.Data, &errorData); err == nil && errorData.Error != "" {
			return nil, fmt.Errorf("service error: %s", errorData.Error)
		}
	}

	return resp.Data, nil
}

// ConnWrapper 包装连接以实现MessageWriter接口
type ConnWrapper struct {
	conn   interface{}
	server *AsyncIPCServer
}

// WriteMessage 实现MessageWriter接口
func (c *ConnWrapper) WriteMessage(msg *IPCMessage) error {
	return c.server.IPCServer.writeMessage(c.conn, msg)
}