package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
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
	listener, err := net.Listen("tcp", "127.0.0.1:9090")
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
	var err error // 统一声明 err 变量（整个函数作用域有效）
	var (         // 新增变量声明块
		version   byte   // 协议版本（1字节）
		msgIDLen  uint16 // 消息ID长度（2字节）
		methodLen uint16 // 方法名长度（2字节）
		paramLen  uint32 // 参数内容长度（4字节）
	)

	// 1. 读取并验证魔数
	magic, err := readUint16(reader) // 首次使用 := 声明 magic 和 err
	if err != nil || magic != magicNumber {
		errorMsg := fmt.Sprintf(`{"error_code": 4001, "error_msg": "invalid magic number: %v"}`, err)
		conn.Write([]byte(errorMsg))
		return
	}

	// 2. 读取协议版本（使用已声明的 version 变量）
	version, err = reader.ReadByte() // 关键修改：version 已声明
	if err != nil || version > 1 {
		conn.Write([]byte(`{"error_code": 4002, "error_msg": "unsupported protocol version"}`))
		return
	}

	// 3. 读取消息ID（使用已声明的 msgIDLen 变量）
	msgIDLen, err = readUint16(reader) // 关键修改：msgIDLen 已声明
	if err != nil {
		conn.Write([]byte(`{"error_code": 4004, "error_msg": "read msgID length failed"}`))
		return
	}
	msgIDBytes, err := readBytes(reader, int(msgIDLen)) // 首次使用 := 声明 msgIDBytes 和 err（新作用域）
	if err != nil {
		conn.Write([]byte(`{"error_code": 4005, "error_msg": "read msgID failed"}`))
		return
	}
	msgID := string(msgIDBytes)

	// 4. 读取方法名（使用已声明的 methodLen 变量）
	methodLen, err = readUint16(reader) // 关键修改：methodLen 已声明
	if err != nil {
		conn.Write([]byte(`{"error_code": 4006, "error_msg": "read method length failed"}`))
		return
	}
	methodBytes, err := readBytes(reader, int(methodLen)) // 首次使用 := 声明 methodBytes 和 err（新作用域）
	if err != nil {
		conn.Write([]byte(`{"error_code": 4007, "error_msg": "read method failed"}`))
		return
	}
	method := string(methodBytes)

	// 5. 读取参数内容（使用已声明的 paramLen 变量）
	paramLen, err = readUint32(reader) // 关键修改：paramLen 已声明
	if err != nil {
		conn.Write([]byte(`{"error_code": 4008, "error_msg": "read param length failed"}`))
		return
	}
	paramData, err := readBytes(reader, int(paramLen)) // 首次使用 := 声明 paramData 和 err（新作用域）
	if err != nil {
		conn.Write([]byte(`{"error_code": 4009, "error_msg": "read param data failed"}`))
		return
	}

	// 6. 反序列化参数
	var params map[string]interface{}
	if err := json.Unmarshal(paramData, &params); err != nil {
		conn.Write([]byte(`{"error_code": 4003, "error_msg": "invalid parameter format"}`))
		return
	}

	// 路由调用（设计文档路由分发）
	registryLock.RLock()
	handler, exists := serviceRegistry[method]
	registryLock.RUnlock()

	var result interface{}
	// 关键修改：删除重复的 var err error 声明，直接使用已存在的 err 变量
	if exists {
		result, err = handler(params)
	} else {
		err = fmt.Errorf("service %s not found", method)
	}

	// 返回响应（关键修改：添加日志和错误检查）
	response, errMarshal := json.Marshal(map[string]interface{}{
		"msg_id": msgID,
		"result": result,
		"error":  err,
	})
	if errMarshal != nil {
		fmt.Printf("响应序列化失败: %v\n", errMarshal)
		return
	}

	// 按协议格式发送响应
	respHeader := make([]byte, 0)
	respHeader = binary.BigEndian.AppendUint16(respHeader, magicNumber)           // 魔数
	respHeader = append(respHeader, version)                                      // 版本
	respHeader = binary.BigEndian.AppendUint32(respHeader, uint32(len(response))) // 内容长度
	conn.Write(append(respHeader, response...))
}

// 辅助函数（读取固定长度字节，增加错误返回）
func readUint16(reader *bufio.Reader) (uint16, error) {
	var val uint16
	err := binary.Read(reader, binary.BigEndian, &val)
	return val, err
}

func readUint32(reader *bufio.Reader) (uint32, error) {
	var val uint32
	err := binary.Read(reader, binary.BigEndian, &val)
	return val, err
}

func readBytes(reader *bufio.Reader, length int) ([]byte, error) {
	buf := make([]byte, length)
	n, err := io.ReadFull(reader, buf)
	if err != nil {
		return nil, fmt.Errorf("readBytes failed: read %d/%d bytes, err=%v", n, length, err)
	}
	return buf, nil
}
