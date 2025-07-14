package ipc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"neo/internal/registry"
	"net"
	"sync"
	"time"
)

// MessageType IPC消息类型
type MessageType byte

const (
	TypeRequest   MessageType = 1
	TypeResponse  MessageType = 2
	TypeRegister  MessageType = 3
	TypeHeartbeat MessageType = 4
)

// IPCMessage IPC消息结构
type IPCMessage struct {
	Type     MessageType
	ID       string
	Service  string
	Method   string
	Data     []byte
	Metadata map[string]string
}

// IPCServer IPC服务器，处理进程间通信
type IPCServer struct {
	addr         string
	listener     net.Listener
	registry     registry.ServiceRegistry
	clients      sync.Map // clientID -> *IPCClient
	handlers     sync.Map // service -> net.Conn
	mu           sync.RWMutex
	asyncHandler ResponseHandler // 异步响应处理器
}

// ResponseHandler 响应处理接口
type ResponseHandler interface {
	HandleResponse(msg *IPCMessage)
}

// IPCClient 代表一个IPC客户端连接
type IPCClient struct {
	conn        net.Conn
	serviceName string
	registered  bool
}

// NewIPCServer 创建新的IPC服务器
func NewIPCServer(addr string, registry registry.ServiceRegistry) *IPCServer {
	return &IPCServer{
		addr:     addr,
		registry: registry,
		// handlers 和 clients 会在首次使用时自动初始化（sync.Map 的零值是可用的）
	}
}

// SetAsyncHandler 设置异步响应处理器
func (s *IPCServer) SetAsyncHandler(handler ResponseHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.asyncHandler = handler
	fmt.Printf("AsyncHandler set for IPC server\n")
}

// Start 启动IPC服务器
func (s *IPCServer) Start() error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	fmt.Printf("IPC Server listening on %s\n", s.addr)

	go s.acceptConnections()
	return nil
}

// Stop 停止IPC服务器
func (s *IPCServer) Stop() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// acceptConnections 接受客户端连接
func (s *IPCServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			return
		}

		client := &IPCClient{
			conn:       conn,
			registered: false,
		}

		clientID := conn.RemoteAddr().String()
		s.clients.Store(clientID, client)

		go s.handleClient(client)
	}
}

// handleClient 处理客户端连接
func (s *IPCServer) handleClient(client *IPCClient) {
	fmt.Printf("=== New IPC client connected: %s ===\n", client.conn.RemoteAddr())
	
	defer func() {
		fmt.Printf("=== IPC client disconnected: %s ===\n", client.conn.RemoteAddr())
		client.conn.Close()
		if client.serviceName != "" {
			s.handlers.Delete(client.serviceName)
			fmt.Printf("Service '%s' unregistered on disconnect\n", client.serviceName)
		}
	}()

	fmt.Printf("Starting message loop for client: %s\n", client.conn.RemoteAddr())
	for {
		msg, err := s.readMessage(client.conn)
		if err != nil {
			if err != io.EOF {
				fmt.Printf("Read error: %v\n", err)
			}
			return
		}

		fmt.Printf("Received message type %d from %s\n", msg.Type, client.conn.RemoteAddr())
		
		switch msg.Type {
		case TypeRegister:
			s.handleRegister(client, msg)
		case TypeResponse:
			s.handleResponse(client, msg)
		case TypeHeartbeat:
			// Handle heartbeat
		default:
			fmt.Printf("Unknown message type: %v\n", msg.Type)
		}
	}
}

// handleRegister 处理服务注册
func (s *IPCServer) handleRegister(client *IPCClient, msg *IPCMessage) {
	fmt.Printf("handleRegister: Received registration message\n")
	fmt.Printf("  Message ID: %s\n", msg.ID)
	fmt.Printf("  Message Service: %s\n", msg.Service)
	fmt.Printf("  Message Method: %s\n", msg.Method)
	fmt.Printf("  Data length: %d bytes\n", len(msg.Data))
	fmt.Printf("  Data content: %s\n", string(msg.Data))
	
	var regData struct {
		Name     string            `json:"name"`
		Metadata map[string]string `json:"metadata"`
	}

	if err := json.Unmarshal(msg.Data, &regData); err != nil {
		fmt.Printf("Register unmarshal error: %v\n", err)
		fmt.Printf("Raw data: %q\n", string(msg.Data))
		return
	}
	
	fmt.Printf("handleRegister: Parsed registration data:\n")
	fmt.Printf("  Name: %s\n", regData.Name)
	fmt.Printf("  Metadata: %v\n", regData.Metadata)

	// 注册到服务注册中心
	instanceID := fmt.Sprintf("%s-%s", regData.Name, client.conn.RemoteAddr().String())
	instance := registry.ServiceInstance{
		ID:       instanceID,
		Name:     regData.Name,
		Address:  client.conn.RemoteAddr().String(),
		Metadata: regData.Metadata,
	}

	ctx := context.Background()
	if err := s.registry.Register(ctx, &instance); err != nil {
		fmt.Printf("Service registration failed: %v\n", err)
		return
	}

	client.serviceName = regData.Name
	client.registered = true
	s.handlers.Store(regData.Name, client.conn)

	fmt.Printf("Service '%s' registered from %s\n", regData.Name, client.conn.RemoteAddr())
	
	// 调试：打印所有已注册的服务
	fmt.Printf("Currently registered services:\n")
	s.handlers.Range(func(key, value interface{}) bool {
		fmt.Printf("  - %v\n", key)
		return true
	})
}

// handleResponse 处理响应消息
func (s *IPCServer) handleResponse(client *IPCClient, msg *IPCMessage) {
	fmt.Printf("handleResponse: Received response for request %s\n", msg.ID)
	fmt.Printf("handleResponse: Response data length: %d bytes\n", len(msg.Data))
	
	// 如果有AsyncIPCServer实例，转发响应到请求处理器
	if s.asyncHandler != nil {
		fmt.Printf("handleResponse: Forwarding response to AsyncHandler\n")
		s.asyncHandler.HandleResponse(msg)
	} else {
		fmt.Printf("handleResponse: No AsyncHandler configured\n")
	}
}

// SendRequest 向指定服务发送请求
func (s *IPCServer) SendRequest(serviceName string, method string, data []byte) (*IPCMessage, error) {
	connInterface, ok := s.handlers.Load(serviceName)
	if !ok {
		return nil, fmt.Errorf("service %s not found", serviceName)
	}

	conn := connInterface.(net.Conn)
	
	msg := &IPCMessage{
		Type:     TypeRequest,
		ID:       generateRequestID(),
		Service:  serviceName,
		Method:   method,
		Data:     data,
		Metadata: make(map[string]string),
	}

	if err := s.writeMessage(conn, msg); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// 在实际实现中，这里应该等待响应
	// 目前简化处理，直接返回
	return msg, nil
}

// readMessage 从连接读取消息
func (s *IPCServer) readMessage(conn net.Conn) (*IPCMessage, error) {
	fmt.Printf("readMessage: Reading from %s\n", conn.RemoteAddr())
	
	// 读取消息长度
	var msgLen uint32
	if err := binary.Read(conn, binary.LittleEndian, &msgLen); err != nil {
		fmt.Printf("readMessage: Failed to read length: %v\n", err)
		return nil, err
	}
	
	fmt.Printf("readMessage: Message length: %d bytes\n", msgLen)
	
	// 验证消息长度合理性
	if msgLen > 1024*1024 { // 1MB limit
		fmt.Printf("readMessage: Message too large: %d bytes\n", msgLen)
		return nil, fmt.Errorf("message too large: %d bytes", msgLen)
	}

	// 读取消息内容
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		fmt.Printf("readMessage: Failed to read message data: %v\n", err)
		return nil, err
	}
	
	fmt.Printf("readMessage: Read %d bytes of message data\n", len(msgData))
	previewLen := 32
	if len(msgData) < previewLen {
		previewLen = len(msgData)
	}
	fmt.Printf("readMessage: First %d bytes: %x\n", previewLen, msgData[:previewLen])

	// 解析消息
	fmt.Printf("readMessage: Parsing message...\n")
	msg := &IPCMessage{}
	offset := 0

	// Type
	if offset >= len(msgData) {
		return nil, fmt.Errorf("insufficient data for message type")
	}
	msg.Type = MessageType(msgData[offset])
	fmt.Printf("readMessage: Message type: %d\n", msg.Type)
	offset++

	// ID
	if offset+4 > len(msgData) {
		return nil, fmt.Errorf("insufficient data for ID length")
	}
	idLen := binary.LittleEndian.Uint32(msgData[offset:])
	fmt.Printf("readMessage: ID length: %d\n", idLen)
	offset += 4
	if offset+int(idLen) > len(msgData) {
		return nil, fmt.Errorf("insufficient data for ID")
	}
	msg.ID = string(msgData[offset : offset+int(idLen)])
	fmt.Printf("readMessage: ID: '%s'\n", msg.ID)
	offset += int(idLen)

	// Service
	if offset+4 > len(msgData) {
		return nil, fmt.Errorf("insufficient data for service length")
	}
	serviceLen := binary.LittleEndian.Uint32(msgData[offset:])
	fmt.Printf("readMessage: Service length: %d\n", serviceLen)
	offset += 4
	if offset+int(serviceLen) > len(msgData) {
		return nil, fmt.Errorf("insufficient data for service")
	}
	msg.Service = string(msgData[offset : offset+int(serviceLen)])
	fmt.Printf("readMessage: Service: '%s'\n", msg.Service)
	offset += int(serviceLen)

	// Method
	methodLen := binary.LittleEndian.Uint32(msgData[offset:])
	offset += 4
	msg.Method = string(msgData[offset : offset+int(methodLen)])
	offset += int(methodLen)

	// Metadata
	metadataLen := binary.LittleEndian.Uint32(msgData[offset:])
	offset += 4
	if metadataLen > 0 {
		metadataJSON := msgData[offset : offset+int(metadataLen)]
		json.Unmarshal(metadataJSON, &msg.Metadata)
		offset += int(metadataLen)
	}

	// Data
	dataLen := binary.LittleEndian.Uint32(msgData[offset:])
	offset += 4
	msg.Data = msgData[offset : offset+int(dataLen)]

	return msg, nil
}

// writeMessage 向连接写入消息
func (s *IPCServer) writeMessage(conn interface{}, msg *IPCMessage) error {
	// 类型断言
	netConn, ok := conn.(net.Conn)
	if !ok {
		return fmt.Errorf("invalid connection type")
	}
	// 序列化metadata
	metadataJSON, _ := json.Marshal(msg.Metadata)

	// 计算总长度
	totalLen := 1 + // Type
		4 + len(msg.ID) + // ID
		4 + len(msg.Service) + // Service
		4 + len(msg.Method) + // Method
		4 + len(metadataJSON) + // Metadata
		4 + len(msg.Data) // Data

	// 写入总长度
	if err := binary.Write(netConn, binary.LittleEndian, uint32(totalLen)); err != nil {
		return err
	}

	// 写入Type
	if _, err := netConn.Write([]byte{byte(msg.Type)}); err != nil {
		return err
	}

	// 写入ID
	if err := binary.Write(netConn, binary.LittleEndian, uint32(len(msg.ID))); err != nil {
		return err
	}
	if _, err := netConn.Write([]byte(msg.ID)); err != nil {
		return err
	}

	// 写入Service
	if err := binary.Write(netConn, binary.LittleEndian, uint32(len(msg.Service))); err != nil {
		return err
	}
	if _, err := netConn.Write([]byte(msg.Service)); err != nil {
		return err
	}

	// 写入Method
	if err := binary.Write(netConn, binary.LittleEndian, uint32(len(msg.Method))); err != nil {
		return err
	}
	if _, err := netConn.Write([]byte(msg.Method)); err != nil {
		return err
	}

	// 写入Metadata
	if err := binary.Write(netConn, binary.LittleEndian, uint32(len(metadataJSON))); err != nil {
		return err
	}
	if _, err := netConn.Write(metadataJSON); err != nil {
		return err
	}

	// 写入Data
	if err := binary.Write(netConn, binary.LittleEndian, uint32(len(msg.Data))); err != nil {
		return err
	}
	if _, err := netConn.Write(msg.Data); err != nil {
		return err
	}

	return nil
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	return fmt.Sprintf("ipc-%d", time.Now().UnixNano())
}