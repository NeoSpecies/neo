package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io" // 新增 io 包（用于 ReadAll）
	"net"
	"strings" // 新增 strings 包（用于 SplitN）
	"sync"
)

// 协议魔数（设计文档定义）
const magicNumber = 0xAEBD

// 服务注册表（设计文档服务发现机制）
var serviceRegistry = make(map[string]func(map[string]interface{}) (interface{}, error))
var registryLock sync.RWMutex

// 注册服务（供 Go/Python 服务调用）
func RegisterService(name string, handler func(map[string]interface{}) (interface{}, error)) {
	registryLock.Lock()
	defer registryLock.Unlock()
	serviceRegistry[name] = handler
}

// 启动 TCP 服务（传输层实现）
func StartIpcServer(addr string) error {
	// 修改监听地址为 IPv4（原 :9090 改为 127.0.0.1:9090）
	listener, err := net.Listen("tcp", "127.0.0.1:9090") // 关键修改
	if err != nil {
		return err
	}
	fmt.Printf("IPC Server listening on %s\n", "127.0.0.1:9090")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn)
	}
}

// 处理连接（协议解析核心）
func handleConnection(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)

	// 跳过二进制协议解析（MVP 阶段简化）
	// 直接读取完整请求数据（格式：method|params_json）
	reqData, _ := io.ReadAll(reader)
	if len(reqData) == 0 {
		fmt.Println("接收到空请求数据") // 新增日志
		return
	}
	fmt.Printf("接收到请求数据：%s\n", string(reqData)) // 新增日志

	// 按 "|" 分割方法名和参数
	parts := strings.SplitN(string(reqData), "|", 2)
	if len(parts) != 2 {
		conn.Write([]byte(`{"error": "invalid request format"}`))
		return
	}
	method := parts[0]
	paramData := parts[1]

	// 反序列化参数（保持 JSON 格式）
	var params map[string]interface{}
	json.Unmarshal([]byte(paramData), &params)

	// 路由调用（设计文档路由分发）
	registryLock.RLock()
	handler, exists := serviceRegistry[method]
	registryLock.RUnlock()

	var result interface{}
	var err error
	if exists {
		result, err = handler(params)
	} else {
		err = fmt.Errorf("service %s not found", method)
	}

	// 返回响应（关键修改：添加日志和错误检查）
	response, errMarshal := json.Marshal(map[string]interface{}{"result": result, "error": err})
	if errMarshal != nil {
		fmt.Printf("响应序列化失败: %v\n", errMarshal)
		return
	}
	n, errWrite := conn.Write(response)
	if errWrite != nil {
		fmt.Printf("响应发送失败: %v\n", errWrite)
		return
	}
	fmt.Printf("成功发送响应（%d字节）: %s\n", n, string(response))
	conn.Close() // 显式关闭连接，通知客户端数据已发送完成
}

// 辅助函数（读取固定长度字节）
func readUint16(reader *bufio.Reader) uint16 {
	var val uint16
	binary.Read(reader, binary.BigEndian, &val)
	return val
}

func readUint32(reader *bufio.Reader) uint32 {
	var val uint32
	binary.Read(reader, binary.BigEndian, &val)
	return val
}

func readBytes(reader *bufio.Reader, length int) []byte {
	buf := make([]byte, length)
	reader.Read(buf)
	return buf
}
