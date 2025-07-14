package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// 简单的测试网关，包含IPC服务器功能
type TestGateway struct {
	httpServer *http.Server
	ipcListener net.Listener
	services   sync.Map // serviceName -> net.Conn
	requests   sync.Map // requestID -> chan response
}

type IPCMessage struct {
	Type     byte              `json:"type"`
	ID       string            `json:"id"`
	Service  string            `json:"service"`
	Method   string            `json:"method"`
	Data     json.RawMessage   `json:"data"`
	Metadata map[string]string `json:"metadata"`
}

func main() {
	gw := &TestGateway{}
	
	// 启动IPC服务器
	go gw.startIPCServer()
	
	// 启动HTTP服务器
	gw.startHTTPServer()
}

func (gw *TestGateway) startIPCServer() {
	listener, err := net.Listen("tcp", ":9999")
	if err != nil {
		fmt.Printf("IPC Server error: %v\n", err)
		return
	}
	defer listener.Close()
	
	gw.ipcListener = listener
	fmt.Println("IPC Server listening on :9999")
	
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Accept error: %v\n", err)
			continue
		}
		
		go gw.handleIPCClient(conn)
	}
}

func (gw *TestGateway) handleIPCClient(conn net.Conn) {
	defer conn.Close()
	
	fmt.Printf("New IPC client connected: %s\n", conn.RemoteAddr())
	
	for {
		// 读取消息长度
		var msgLen uint32
		if err := binary.Read(conn, binary.LittleEndian, &msgLen); err != nil {
			if err != io.EOF {
				fmt.Printf("Read length error: %v\n", err)
			}
			return
		}
		
		// 读取消息
		msgData := make([]byte, msgLen)
		if _, err := io.ReadFull(conn, msgData); err != nil {
			fmt.Printf("Read message error: %v\n", err)
			return
		}
		
		// 解析消息
		msg := gw.parseMessage(msgData)
		fmt.Printf("Received message: Type=%d, Service=%s, Method=%s\n", msg.Type, msg.Service, msg.Method)
		
		// 处理消息
		switch msg.Type {
		case 3: // REGISTER
			gw.handleRegister(conn, msg)
		case 2: // RESPONSE
			gw.handleResponse(msg)
		}
	}
}

func (gw *TestGateway) parseMessage(data []byte) *IPCMessage {
	msg := &IPCMessage{}
	offset := 0
	
	// Type
	msg.Type = data[offset]
	offset++
	
	// ID
	idLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	msg.ID = string(data[offset : offset+int(idLen)])
	offset += int(idLen)
	
	// Service
	serviceLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	msg.Service = string(data[offset : offset+int(serviceLen)])
	offset += int(serviceLen)
	
	// Method
	methodLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	msg.Method = string(data[offset : offset+int(methodLen)])
	offset += int(methodLen)
	
	// Metadata
	metadataLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	if metadataLen > 0 {
		metadataJSON := data[offset : offset+int(metadataLen)]
		json.Unmarshal(metadataJSON, &msg.Metadata)
		offset += int(metadataLen)
	}
	
	// Data
	dataLen := binary.LittleEndian.Uint32(data[offset:])
	offset += 4
	msg.Data = data[offset : offset+int(dataLen)]
	
	return msg
}

func (gw *TestGateway) handleRegister(conn net.Conn, msg *IPCMessage) {
	var regData struct {
		Name     string            `json:"name"`
		Metadata map[string]string `json:"metadata"`
	}
	
	json.Unmarshal(msg.Data, &regData)
	gw.services.Store(regData.Name, conn)
	
	fmt.Printf("Service '%s' registered\n", regData.Name)
}

func (gw *TestGateway) handleResponse(msg *IPCMessage) {
	if ch, ok := gw.requests.Load(msg.ID); ok {
		respChan := ch.(chan *IPCMessage)
		respChan <- msg
	}
}

func (gw *TestGateway) sendRequest(conn net.Conn, service, method string, data []byte) (*IPCMessage, error) {
	msg := &IPCMessage{
		Type:     1, // REQUEST
		ID:       fmt.Sprintf("req-%d", time.Now().UnixNano()),
		Service:  service,
		Method:   method,
		Data:     data,
		Metadata: make(map[string]string),
	}
	
	// 创建响应通道
	respChan := make(chan *IPCMessage, 1)
	gw.requests.Store(msg.ID, respChan)
	defer gw.requests.Delete(msg.ID)
	
	// 发送请求
	if err := gw.writeMessage(conn, msg); err != nil {
		return nil, err
	}
	
	// 等待响应
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

func (gw *TestGateway) writeMessage(conn net.Conn, msg *IPCMessage) error {
	// 序列化metadata
	metadataJSON, _ := json.Marshal(msg.Metadata)
	
	// 构建消息
	buf := &bytes.Buffer{}
	buf.WriteByte(msg.Type)
	
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
	binary.Write(buf, binary.LittleEndian, uint32(len(metadataJSON)))
	buf.Write(metadataJSON)
	
	// Data
	binary.Write(buf, binary.LittleEndian, uint32(len(msg.Data)))
	buf.Write(msg.Data)
	
	// 发送总长度和消息
	msgData := buf.Bytes()
	binary.Write(conn, binary.LittleEndian, uint32(len(msgData)))
	_, err := conn.Write(msgData)
	
	return err
}

func (gw *TestGateway) startHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", gw.handleAPI)
	mux.HandleFunc("/health", gw.handleHealth)
	
	gw.httpServer = &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	
	fmt.Println("HTTP Gateway listening on :8080")
	gw.httpServer.ListenAndServe()
}

func (gw *TestGateway) handleAPI(w http.ResponseWriter, r *http.Request) {
	// 解析路径
	path := r.URL.Path[5:] // 去掉 "/api/"
	parts := bytes.Split([]byte(path), []byte("/"))
	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	
	service := string(parts[0])
	method := string(parts[1])
	
	// 读取请求体
	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	
	fmt.Printf("\nHTTP Request: %s %s\n", r.Method, r.URL.Path)
	fmt.Printf("Service: %s, Method: %s\n", service, method)
	fmt.Printf("Body: %s\n", string(body))
	
	// 查找服务
	connInterface, ok := gw.services.Load(service)
	if !ok {
		fmt.Printf("Service not found: %s\n", service)
		http.Error(w, fmt.Sprintf("Service '%s' not found", service), http.StatusNotFound)
		return
	}
	
	conn := connInterface.(net.Conn)
	
	// 构建请求数据
	requestData := map[string]interface{}{
		"method": method,
		"params": json.RawMessage(body),
	}
	data, _ := json.Marshal(requestData)
	
	// 发送IPC请求
	resp, err := gw.sendRequest(conn, service, method, data)
	if err != nil {
		fmt.Printf("IPC request error: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	fmt.Printf("IPC Response: %s\n", string(resp.Data))
	
	// 返回响应
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp.Data)
}

func (gw *TestGateway) handleHealth(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

