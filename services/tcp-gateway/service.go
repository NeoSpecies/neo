package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// TCPMessage TCP消息格式
type TCPMessage struct {
	Service string                 `json:"service"`
	Method  string                 `json:"method"`
	Data    map[string]interface{} `json:"data"`
}

// TCPResponse TCP响应格式
type TCPResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

// TCPGatewayService TCP网关服务
type TCPGatewayService struct {
	// IPC客户端
	ipcClient   *IPCClient
	serviceName string

	// TCP服务器
	tcpListener net.Listener
	tcpAddr     string
	protocol    string // "json" or "binary"

	// 请求管理
	requestCounter uint64
	mu             sync.Mutex
	
	// 连接管理
	connections map[net.Conn]bool
	connMu      sync.RWMutex
}

// NewTCPGatewayService 创建TCP网关服务
func NewTCPGatewayService(tcpAddr string, protocol string) *TCPGatewayService {
	if protocol == "" {
		protocol = "json"
	}
	
	return &TCPGatewayService{
		serviceName:    "tcp-gateway",
		tcpAddr:        tcpAddr,
		protocol:       protocol,
		requestCounter: 0,
		connections:    make(map[net.Conn]bool),
	}
}

// ConnectToIPC 连接到IPC服务器
func (s *TCPGatewayService) ConnectToIPC(ipcAddr string) error {
	client, err := NewIPCClient(ipcAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to IPC: %w", err)
	}
	s.ipcClient = client

	// 注册TCP网关特有的处理器
	s.registerHandlers()

	// 注册服务
	metadata := map[string]string{
		"type":        "gateway",
		"protocol":    "tcp",
		"subprotocol": s.protocol,
		"version":     "1.0.0",
		"description": "TCP Gateway Service for Neo Framework",
	}
	
	if err := s.ipcClient.RegisterService(s.serviceName, metadata); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	// 启动心跳
	s.ipcClient.StartHeartbeat()

	log.Printf("TCP Gateway registered to IPC server at %s", ipcAddr)
	return nil
}

// registerHandlers 注册IPC处理器
func (s *TCPGatewayService) registerHandlers() {
	// 处理获取网关信息的请求
	s.ipcClient.AddHandler("getInfo", func(msg *Message) (*Message, error) {
		s.connMu.RLock()
		connCount := len(s.connections)
		s.connMu.RUnlock()
		
		info := map[string]interface{}{
			"service":     s.serviceName,
			"type":        "tcp-gateway",
			"tcpAddress":  s.tcpAddr,
			"protocol":    s.protocol,
			"connections": connCount,
			"status":      "running",
			"version":     "1.0.0",
		}
		
		data, _ := json.Marshal(info)
		return &Message{
			Metadata: map[string]string{},
			Data:     data,
		}, nil
	})

	// 处理获取连接统计的请求
	s.ipcClient.AddHandler("getStats", func(msg *Message) (*Message, error) {
		s.connMu.RLock()
		connCount := len(s.connections)
		s.connMu.RUnlock()
		
		stats := map[string]interface{}{
			"activeConnections": connCount,
			"totalRequests":     s.requestCounter,
			"protocol":          s.protocol,
		}
		
		data, _ := json.Marshal(stats)
		return &Message{
			Metadata: map[string]string{},
			Data:     data,
		}, nil
	})
}

// StartTCPServer 启动TCP服务器
func (s *TCPGatewayService) StartTCPServer() error {
	listener, err := net.Listen("tcp", s.tcpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.tcpAddr, err)
	}
	
	s.tcpListener = listener
	log.Printf("TCP Gateway listening on %s (protocol: %s)", s.tcpAddr, s.protocol)
	log.Printf("Accepting TCP connections...")
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}
		
		// 记录新连接
		s.connMu.Lock()
		s.connections[conn] = true
		s.connMu.Unlock()
		
		log.Printf("[TCP] New connection from %s", conn.RemoteAddr())
		go s.handleConnection(conn)
	}
}

// handleConnection 处理TCP连接
func (s *TCPGatewayService) handleConnection(conn net.Conn) {
	defer func() {
		// 清理连接
		s.connMu.Lock()
		delete(s.connections, conn)
		s.connMu.Unlock()
		
		conn.Close()
		log.Printf("[TCP] Connection closed: %s", conn.RemoteAddr())
	}()

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	
	for {
		// 根据协议读取消息
		var msg TCPMessage
		var err error
		
		if s.protocol == "json" {
			msg, err = s.readJSONMessage(conn)
		} else {
			msg, err = s.readBinaryMessage(conn)
		}
		
		if err != nil {
			if err != io.EOF {
				log.Printf("[TCP] Read error from %s: %v", conn.RemoteAddr(), err)
			}
			break
		}

		// 重置读取超时
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
		
		// 处理请求
		s.handleRequest(conn, msg)
	}
}

// handleRequest 处理TCP请求
func (s *TCPGatewayService) handleRequest(conn net.Conn, msg TCPMessage) {
	// 生成请求ID
	requestID := s.generateRequestID()
	
	log.Printf("[TCP] Request %s from %s: service='%s', method='%s'", 
		requestID, conn.RemoteAddr(), msg.Service, msg.Method)

	// 准备请求数据
	requestData, err := json.Marshal(msg.Data)
	if err != nil {
		s.sendErrorResponse(conn, "Invalid request data")
		return
	}

	// 通过IPC转发请求
	response, err := s.ipcClient.SendRequest(msg.Service, msg.Method, requestData, requestID)
	if err != nil {
		log.Printf("[TCP] IPC request failed: %v", err)
		s.sendErrorResponse(conn, fmt.Sprintf("Service call failed: %v", err))
		return
	}

	// 检查响应是否包含错误
	if response.Metadata != nil && response.Metadata["error"] == "true" {
		var errorData map[string]interface{}
		json.Unmarshal(response.Data, &errorData)
		errorMsg, _ := errorData["error"].(string)
		s.sendErrorResponse(conn, errorMsg)
		log.Printf("[TCP] Request %s returned error: %s", requestID, errorMsg)
		return
	}

	// 解析响应数据
	var responseData map[string]interface{}
	if err := json.Unmarshal(response.Data, &responseData); err != nil {
		// 如果无法解析为map，则将原始数据包装
		responseData = map[string]interface{}{
			"result": string(response.Data),
		}
	}

	// 发送成功响应
	s.sendSuccessResponse(conn, responseData)
	log.Printf("[TCP] Request %s completed successfully", requestID)
}

// readJSONMessage 读取JSON格式消息
func (s *TCPGatewayService) readJSONMessage(conn net.Conn) (TCPMessage, error) {
	var msg TCPMessage
	
	// 读取消息长度（4字节）
	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		return msg, err
	}
	
	// 限制消息大小（最大1MB）
	if msgLen > 1024*1024 {
		return msg, fmt.Errorf("message too large: %d bytes", msgLen)
	}
	
	// 读取消息内容
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return msg, err
	}
	
	// 解析JSON
	if err := json.Unmarshal(msgData, &msg); err != nil {
		return msg, fmt.Errorf("invalid JSON: %w", err)
	}
	
	return msg, nil
}

// readBinaryMessage 读取二进制格式消息（为将来扩展预留）
func (s *TCPGatewayService) readBinaryMessage(conn net.Conn) (TCPMessage, error) {
	// 目前使用与JSON相同的格式
	return s.readJSONMessage(conn)
}

// sendSuccessResponse 发送成功响应
func (s *TCPGatewayService) sendSuccessResponse(conn net.Conn, data map[string]interface{}) {
	resp := TCPResponse{
		Success: true,
		Data:    data,
	}
	s.sendResponse(conn, resp)
}

// sendErrorResponse 发送错误响应
func (s *TCPGatewayService) sendErrorResponse(conn net.Conn, errorMsg string) {
	resp := TCPResponse{
		Success: false,
		Error:   errorMsg,
	}
	s.sendResponse(conn, resp)
}

// sendResponse 发送响应
func (s *TCPGatewayService) sendResponse(conn net.Conn, resp TCPResponse) {
	if s.protocol == "json" {
		s.writeJSONMessage(conn, resp)
	} else {
		s.writeBinaryMessage(conn, resp)
	}
}

// writeJSONMessage 写入JSON消息
func (s *TCPGatewayService) writeJSONMessage(conn net.Conn, resp TCPResponse) error {
	// 序列化响应
	respData, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	
	// 写入消息长度
	if err := binary.Write(conn, binary.BigEndian, uint32(len(respData))); err != nil {
		return err
	}
	
	// 写入消息内容
	_, err = conn.Write(respData)
	return err
}

// writeBinaryMessage 写入二进制消息（为将来扩展预留）
func (s *TCPGatewayService) writeBinaryMessage(conn net.Conn, resp TCPResponse) error {
	// 目前使用与JSON相同的格式
	return s.writeJSONMessage(conn, resp)
}

// generateRequestID 生成请求ID
func (s *TCPGatewayService) generateRequestID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requestCounter++
	return fmt.Sprintf("tcp-%d-%d", time.Now().Unix(), s.requestCounter)
}

// Stop 停止服务
func (s *TCPGatewayService) Stop() error {
	// 关闭所有连接
	s.connMu.Lock()
	for conn := range s.connections {
		conn.Close()
	}
	s.connMu.Unlock()
	
	// 关闭监听器
	if s.tcpListener != nil {
		if err := s.tcpListener.Close(); err != nil {
			return err
		}
	}
	
	// 关闭IPC客户端
	if s.ipcClient != nil {
		return s.ipcClient.Close()
	}
	return nil
}