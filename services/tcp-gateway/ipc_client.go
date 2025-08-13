package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

// MessageType 消息类型
type MessageType byte

const (
	REQUEST   MessageType = 1
	RESPONSE  MessageType = 2
	REGISTER  MessageType = 3
	HEARTBEAT MessageType = 4
)

// Message IPC消息结构
type Message struct {
	Type     MessageType
	ID       string
	Service  string
	Method   string
	Metadata map[string]string
	Data     []byte
}

// RequestHandler 请求处理函数
type RequestHandler func(msg *Message) (*Message, error)

// IPCClient IPC客户端
type IPCClient struct {
	conn         net.Conn
	serviceName  string
	handlers     map[string]RequestHandler
	pendingReqs  map[string]chan *Message
	mu           sync.RWMutex
	stopChan     chan struct{}
	connected    bool
}

// NewIPCClient 创建IPC客户端
func NewIPCClient(addr string) (*IPCClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IPC server: %w", err)
	}

	log.Printf("Connected to Neo IPC server at %s", addr)

	client := &IPCClient{
		conn:        conn,
		handlers:    make(map[string]RequestHandler),
		pendingReqs: make(map[string]chan *Message),
		stopChan:    make(chan struct{}),
		connected:   true,
	}

	// 启动消息处理循环
	go client.messageLoop()

	return client, nil
}

// RegisterService 注册服务
func (c *IPCClient) RegisterService(name string, metadata map[string]string) error {
	c.serviceName = name

	registerData := map[string]interface{}{
		"name":     name,
		"metadata": metadata,
	}

	data, _ := json.Marshal(registerData)

	msg := &Message{
		Type:     REGISTER,
		ID:       "",
		Service:  name,
		Method:   "",
		Metadata: map[string]string{},
		Data:     data,
	}

	if err := c.sendMessage(msg); err != nil {
		return err
	}

	log.Printf("Service '%s' registered", name)
	return nil
}

// AddHandler 添加请求处理器
func (c *IPCClient) AddHandler(method string, handler RequestHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[method] = handler
	log.Printf("Handler registered for method: %s", method)
}

// SendRequest 发送请求并等待响应
func (c *IPCClient) SendRequest(targetService, method string, data []byte, requestID string) (*Message, error) {
	// 创建响应通道
	respChan := make(chan *Message, 1)
	
	c.mu.Lock()
	c.pendingReqs[requestID] = respChan
	c.mu.Unlock()

	// 构建请求消息
	msg := &Message{
		Type:     REQUEST,
		ID:       requestID,
		Service:  targetService,
		Method:   method,
		Metadata: map[string]string{},
		Data:     data,
	}

	// 发送消息
	if err := c.sendMessage(msg); err != nil {
		c.mu.Lock()
		delete(c.pendingReqs, requestID)
		c.mu.Unlock()
		return nil, err
	}

	// 等待响应（30秒超时）
	select {
	case resp := <-respChan:
		c.mu.Lock()
		delete(c.pendingReqs, requestID)
		c.mu.Unlock()
		return resp, nil
	case <-time.After(30 * time.Second):
		c.mu.Lock()
		delete(c.pendingReqs, requestID)
		c.mu.Unlock()
		return nil, fmt.Errorf("request timeout")
	}
}

// sendMessage 发送消息
func (c *IPCClient) sendMessage(msg *Message) error {
	if !c.connected {
		return fmt.Errorf("not connected to IPC server")
	}

	buf := &bytes.Buffer{}

	// 消息类型
	buf.WriteByte(byte(msg.Type))

	// ID
	binary.Write(buf, binary.LittleEndian, uint32(len(msg.ID)))
	buf.WriteString(msg.ID)

	// Service
	binary.Write(buf, binary.LittleEndian, uint32(len(msg.Service)))
	buf.WriteString(msg.Service)

	// Method
	binary.Write(buf, binary.LittleEndian, uint32(len(msg.Method)))
	buf.WriteString(msg.Method)

	// Metadata
	metadataJSON, _ := json.Marshal(msg.Metadata)
	binary.Write(buf, binary.LittleEndian, uint32(len(metadataJSON)))
	buf.Write(metadataJSON)

	// Data
	binary.Write(buf, binary.LittleEndian, uint32(len(msg.Data)))
	buf.Write(msg.Data)

	// 发送消息长度和内容
	content := buf.Bytes()
	binary.Write(c.conn, binary.LittleEndian, uint32(len(content)))
	_, err := c.conn.Write(content)

	return err
}

// readMessage 读取消息
func (c *IPCClient) readMessage() (*Message, error) {
	// 读取消息长度
	var msgLen uint32
	if err := binary.Read(c.conn, binary.LittleEndian, &msgLen); err != nil {
		return nil, err
	}

	// 读取消息内容
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(c.conn, msgData); err != nil {
		return nil, err
	}

	reader := bytes.NewReader(msgData)
	msg := &Message{}

	// 解析消息类型
	var msgType byte
	binary.Read(reader, binary.LittleEndian, &msgType)
	msg.Type = MessageType(msgType)

	// 解析ID
	var idLen uint32
	binary.Read(reader, binary.LittleEndian, &idLen)
	idBytes := make([]byte, idLen)
	io.ReadFull(reader, idBytes)
	msg.ID = string(idBytes)

	// 解析Service
	var serviceLen uint32
	binary.Read(reader, binary.LittleEndian, &serviceLen)
	serviceBytes := make([]byte, serviceLen)
	io.ReadFull(reader, serviceBytes)
	msg.Service = string(serviceBytes)

	// 解析Method
	var methodLen uint32
	binary.Read(reader, binary.LittleEndian, &methodLen)
	methodBytes := make([]byte, methodLen)
	io.ReadFull(reader, methodBytes)
	msg.Method = string(methodBytes)

	// 解析Metadata
	var metadataLen uint32
	binary.Read(reader, binary.LittleEndian, &metadataLen)
	metadataBytes := make([]byte, metadataLen)
	io.ReadFull(reader, metadataBytes)
	if len(metadataBytes) > 0 {
		json.Unmarshal(metadataBytes, &msg.Metadata)
	} else {
		msg.Metadata = make(map[string]string)
	}

	// 解析Data
	var dataLen uint32
	binary.Read(reader, binary.LittleEndian, &dataLen)
	msg.Data = make([]byte, dataLen)
	io.ReadFull(reader, msg.Data)

	return msg, nil
}

// messageLoop 消息处理循环
func (c *IPCClient) messageLoop() {
	for {
		select {
		case <-c.stopChan:
			return
		default:
			msg, err := c.readMessage()
			if err != nil {
				log.Printf("Failed to read message: %v", err)
				c.connected = false
				return
			}

			switch msg.Type {
			case REQUEST:
				// 处理接收到的请求
				go c.handleRequest(msg)
			case RESPONSE:
				// 处理响应
				c.handleResponse(msg)
			}
		}
	}
}

// handleRequest 处理请求
func (c *IPCClient) handleRequest(msg *Message) {
	c.mu.RLock()
	handler, ok := c.handlers[msg.Method]
	c.mu.RUnlock()

	var response *Message
	if !ok {
		// 方法未找到
		response = &Message{
			Type:    RESPONSE,
			ID:      msg.ID,
			Service: c.serviceName,
			Method:  msg.Method,
			Metadata: map[string]string{
				"error": "true",
			},
			Data: []byte(fmt.Sprintf(`{"error":"Method '%s' not found"}`, msg.Method)),
		}
	} else {
		// 调用处理器
		resp, err := handler(msg)
		if err != nil {
			response = &Message{
				Type:    RESPONSE,
				ID:      msg.ID,
				Service: c.serviceName,
				Method:  msg.Method,
				Metadata: map[string]string{
					"error": "true",
				},
				Data: []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())),
			}
		} else {
			response = resp
			response.Type = RESPONSE
			response.ID = msg.ID
			response.Service = c.serviceName
			response.Method = msg.Method
		}
	}

	// 发送响应
	if err := c.sendMessage(response); err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

// handleResponse 处理响应
func (c *IPCClient) handleResponse(msg *Message) {
	c.mu.RLock()
	respChan, ok := c.pendingReqs[msg.ID]
	c.mu.RUnlock()

	if ok {
		select {
		case respChan <- msg:
		default:
			log.Printf("Response channel blocked for request ID: %s", msg.ID)
		}
	}
}

// StartHeartbeat 启动心跳
func (c *IPCClient) StartHeartbeat() {
	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				msg := &Message{
					Type:     HEARTBEAT,
					ID:       "",
					Service:  c.serviceName,
					Method:   "",
					Metadata: map[string]string{},
					Data:     []byte{},
				}
				if err := c.sendMessage(msg); err != nil {
					log.Printf("Heartbeat error: %v", err)
					return
				}
				log.Println("Heartbeat sent")
			case <-c.stopChan:
				return
			}
		}
	}()
}

// Close 关闭客户端
func (c *IPCClient) Close() error {
	close(c.stopChan)
	c.connected = false
	return c.conn.Close()
}