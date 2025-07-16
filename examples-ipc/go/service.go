package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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

// Handler 处理函数类型
type Handler func(params map[string]interface{}) (interface{}, error)

// NeoIPCClient IPC客户端
type NeoIPCClient struct {
	conn        net.Conn
	serviceName string
	handlers    map[string]Handler
}

// NewNeoIPCClient 创建IPC客户端
func NewNeoIPCClient(addr string) (*NeoIPCClient, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}

	log.Printf("Connected to Neo IPC server at %s", addr)

	return &NeoIPCClient{
		conn:     conn,
		handlers: make(map[string]Handler),
	}, nil
}

// RegisterService 注册服务
func (c *NeoIPCClient) RegisterService(name string, metadata map[string]string) error {
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

// AddHandler 添加处理器
func (c *NeoIPCClient) AddHandler(method string, handler Handler) {
	c.handlers[method] = handler
	log.Printf("Handler registered for method: %s", method)
}

// sendMessage 发送消息
func (c *NeoIPCClient) sendMessage(msg *Message) error {
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
func (c *NeoIPCClient) readMessage() (*Message, error) {
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

// handleRequest 处理请求
func (c *NeoIPCClient) handleRequest(msg *Message) {
	handler, ok := c.handlers[msg.Method]
	if !ok {
		errorResp := &Message{
			Type:    RESPONSE,
			ID:      msg.ID,
			Service: msg.Service,
			Method:  msg.Method,
			Metadata: map[string]string{
				"error": "true",
			},
			Data: []byte(fmt.Sprintf(`{"error":"Method '%s' not found"}`, msg.Method)),
		}
		c.sendMessage(errorResp)
		return
	}

	// 解析请求参数
	var params map[string]interface{}
	if len(msg.Data) > 0 {
		json.Unmarshal(msg.Data, &params)
	}

	// 调用处理器
	result, err := handler(params)
	if err != nil {
		errorResp := &Message{
			Type:    RESPONSE,
			ID:      msg.ID,
			Service: msg.Service,
			Method:  msg.Method,
			Metadata: map[string]string{
				"error": "true",
			},
			Data: []byte(fmt.Sprintf(`{"error":"%s"}`, err.Error())),
		}
		c.sendMessage(errorResp)
		return
	}

	// 发送响应
	responseData, _ := json.Marshal(result)
	response := &Message{
		Type:     RESPONSE,
		ID:       msg.ID,
		Service:  msg.Service,
		Method:   msg.Method,
		Metadata: map[string]string{},
		Data:     responseData,
	}
	c.sendMessage(response)
}

// heartbeatLoop 心跳循环
func (c *NeoIPCClient) heartbeatLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
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
	}
}

// Run 运行服务
func (c *NeoIPCClient) Run() error {
	// 启动心跳
	go c.heartbeatLoop()

	// 消息处理循环
	for {
		msg, err := c.readMessage()
		if err != nil {
			return err
		}

		if msg.Type == REQUEST {
			go c.handleRequest(msg)
		}
	}
}

func main() {
	// 从环境变量读取配置
	host := os.Getenv("NEO_IPC_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("NEO_IPC_PORT")
	if port == "" {
		port = "29999"  // 使用正确的默认端口
	}

	// 创建客户端
	client, err := NewNeoIPCClient(fmt.Sprintf("%s:%s", host, port))
	if err != nil {
		log.Fatal(err)
	}

	// 注册处理器
	client.AddHandler("hello", func(params map[string]interface{}) (interface{}, error) {
		name, _ := params["name"].(string)
		if name == "" {
			name = "World"
		}
		return map[string]interface{}{
			"message":   fmt.Sprintf("Hello, %s!", name),
			"timestamp": time.Now().Format(time.RFC3339),
			"service":   "Go Demo Service",
		}, nil
	})

	client.AddHandler("calculate", func(params map[string]interface{}) (interface{}, error) {
		a, _ := params["a"].(float64)
		b, _ := params["b"].(float64)
		operation, _ := params["operation"].(string)

		var result interface{}
		switch operation {
		case "add":
			result = a + b
		case "subtract":
			result = a - b
		case "multiply":
			result = a * b
		case "divide":
			if b != 0 {
				result = a / b
			} else {
				result = "Cannot divide by zero"
			}
		default:
			result = "Unknown operation"
		}

		return map[string]interface{}{
			"result":    result,
			"operation": operation,
			"a":         a,
			"b":         b,
		}, nil
	})

	client.AddHandler("echo", func(params map[string]interface{}) (interface{}, error) {
		message, _ := params["message"].(string)
		return map[string]interface{}{
			"echo":     message,
			"length":   len(message),
			"reversed": reverseString(message),
		}, nil
	})

	client.AddHandler("getTime", func(params map[string]interface{}) (interface{}, error) {
		format, _ := params["format"].(string)
		now := time.Now()

		var timeStr string
		switch format {
		case "unix":
			timeStr = fmt.Sprintf("%d", now.Unix())
		case "readable":
			timeStr = now.Format("2006-01-02 15:04:05")
		default:
			timeStr = now.Format(time.RFC3339)
		}

		return map[string]interface{}{
			"time":     timeStr,
			"timezone": now.Location().String(),
			"format":   format,
		}, nil
	})

	client.AddHandler("getInfo", func(params map[string]interface{}) (interface{}, error) {
		handlers := make([]string, 0, len(client.handlers))
		for method := range client.handlers {
			handlers = append(handlers, method)
		}

		return map[string]interface{}{
			"service":  "demo-service",
			"language": "Go",
			"version":  "1.0.0",
			"handlers": handlers,
			"uptime":   "N/A",
			"system": map[string]interface{}{
				"platform":   "go",
				"go_version": "1.x",
			},
		}, nil
	})

	// 注册服务
	err = client.RegisterService("demo-service", map[string]string{
		"language":    "go",
		"version":     "1.0.0",
		"description": "Go demo service for Neo Framework",
	})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Go demo service is running...")
	log.Printf("Listening on %s:%s", host, port)
	log.Println("Available methods: hello, calculate, echo, getTime, getInfo")

	// 运行服务
	if err := client.Run(); err != nil {
		log.Fatal(err)
	}
}

// reverseString 反转字符串
func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}