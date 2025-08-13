package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
)

// TCPMessage TCP消息格式
type TestTCPMessage struct {
	Service string                 `json:"service"`
	Method  string                 `json:"method"`
	Data    map[string]interface{} `json:"data"`
}

// TCPResponse TCP响应格式
type TestTCPResponse struct {
	Success bool                   `json:"success"`
	Data    map[string]interface{} `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
}

func sendMessage(conn net.Conn, msg TestTCPMessage) error {
	// 序列化消息
	msgData, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	
	// 发送长度
	if err := binary.Write(conn, binary.BigEndian, uint32(len(msgData))); err != nil {
		return err
	}
	
	// 发送数据
	_, err = conn.Write(msgData)
	return err
}

func readResponse(conn net.Conn) (TestTCPResponse, error) {
	var resp TestTCPResponse
	
	// 读取长度
	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		return resp, err
	}
	
	// 读取数据
	msgData := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, msgData); err != nil {
		return resp, err
	}
	
	// 解析响应
	err := json.Unmarshal(msgData, &resp)
	return resp, err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("TCP Gateway Test Client")
		fmt.Println("Usage: test_client <tcp_address>")
		fmt.Println("Example: test_client localhost:7777")
		os.Exit(1)
	}
	
	addr := os.Args[1]
	
	// 连接到TCP网关
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()
	
	fmt.Printf("Connected to TCP Gateway at %s\n", addr)
	
	// 测试1: 调用Python服务的calculate方法
	fmt.Println("\n--- Test 1: Call demo-service-python/calculate ---")
	msg1 := TestTCPMessage{
		Service: "demo-service-python",
		Method:  "calculate",
		Data: map[string]interface{}{
			"operation": "add",
			"a":         15.0,
			"b":         25.0,
		},
	}
	
	if err := sendMessage(conn, msg1); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}
	
	resp1, err := readResponse(conn)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}
	
	fmt.Printf("Response: %+v\n", resp1)
	
	// 测试2: 调用Python服务的echo方法
	fmt.Println("\n--- Test 2: Call demo-service-python/echo ---")
	msg2 := TestTCPMessage{
		Service: "demo-service-python",
		Method:  "echo",
		Data: map[string]interface{}{
			"message": "Hello from TCP client!",
		},
	}
	
	if err := sendMessage(conn, msg2); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}
	
	resp2, err := readResponse(conn)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}
	
	fmt.Printf("Response: %+v\n", resp2)
	
	// 测试3: 调用不存在的服务
	fmt.Println("\n--- Test 3: Call non-existent service ---")
	msg3 := TestTCPMessage{
		Service: "non-existent",
		Method:  "test",
		Data:    map[string]interface{}{},
	}
	
	if err := sendMessage(conn, msg3); err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}
	
	resp3, err := readResponse(conn)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}
	
	fmt.Printf("Response: %+v\n", resp3)
	
	fmt.Println("\nAll tests completed!")
}