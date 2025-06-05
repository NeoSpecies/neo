package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32" // 替换为正确包名
	"net"
	"sync"
)

// 协议魔数（设计文档定义）
const magicNumber = 0xAEBD

// 服务注册表（设计文档服务发现机制）
var serviceRegistry = make(map[string]func(map[string]interface{}) (interface{}, error))
var registryLock sync.RWMutex

// 全局协程池
var workerPool *WorkerPool

// 初始化协程池
func init() {
	workerPool = NewWorkerPool(100) // 创建100个工作协程的协程池
}

// 注册服务（供 Go/Python 服务调用）
func RegisterService(name string, handler func(map[string]interface{}) (interface{}, error)) {
	registryLock.Lock()
	defer registryLock.Unlock()
	serviceRegistry[name] = handler
}

// 启动 TCP 服务（传输层实现）
func StartIpcServer(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("IPC Server listening on %s\n", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		// 使用协程池处理连接
		workerPool.Submit(func() {
			handleConnection(conn)
		})
	}
}

// 发送错误响应
func sendErrorResponse(conn net.Conn, errorMsg string) {
	// 构造响应头
	header := make([]byte, 0)
	header = binary.BigEndian.AppendUint16(header, magicNumber)
	header = append(header, 0x01) // 版本号

	// 构造响应体
	body := []byte(errorMsg)
	header = binary.BigEndian.AppendUint32(header, uint32(len(body)))

	// 发送完整响应
	conn.Write(append(header, body...))
}

// 处理连接（协议解析核心）
func handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// 将变量声明移动到函数体内
	var err error
	var (
		version   byte
		msgIDLen  uint16
		methodLen uint16
		paramLen  uint32
		checksum  uint32
		totalData []byte
	)

	// 1. 读取并验证魔数（此处开始处理协议头）
	magic, err := readUint16(reader)
	if err != nil || magic != magicNumber {
		// 修复：补充具体错误信息（包含实际魔数值）
		errorMsg := fmt.Sprintf(`{"error_code": 4001, "error_msg": "invalid magic number, expected %#x, got %v"}`, magicNumber, magic)
		sendErrorResponse(conn, errorMsg)
		return
	}
	// 将魔数添加到总数据中
	totalData = append(totalData, byte(magic>>8), byte(magic))

	// 2. 读取协议版本
	version, err = reader.ReadByte()
	if err != nil || version > 1 {
		// 修复：补充实际版本号
		errorMsg := fmt.Sprintf(`{"error_code": 4002, "error_msg": "unsupported protocol version, expected <=1, got %d"}`, version)
		sendErrorResponse(conn, errorMsg)
		return
	}
	totalData = append(totalData, version) // 记录版本到总数据

	// 3. 读取消息ID
	msgIDLen, err = readUint16(reader)
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4004, "error_msg": "read msgID length failed"}`)
		return
	}
	msgIDBytes, err := readBytes(reader, int(msgIDLen))
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4005, "error_msg": "read msgID failed"}`)
		return
	}
	msgID := string(msgIDBytes)
	// 拆分 msgIDLen 的字节和 msgIDBytes 为单个字节追加，解决 append 参数类型不匹配问题
	totalData = append(totalData, byte(msgIDLen>>8))
	totalData = append(totalData, byte(msgIDLen))
	totalData = append(totalData, msgIDBytes...)

	// 4. 读取方法名
	methodLen, err = readUint16(reader)
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4006, "error_msg": "read method length failed"}`)
		return
	}
	methodBytes, err := readBytes(reader, int(methodLen))
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4007, "error_msg": "read method failed"}`)
		return
	}
	method := string(methodBytes)
	// 拆分追加操作，解决 append 参数过多问题
	totalData = append(totalData, byte(methodLen>>8))
	totalData = append(totalData, byte(methodLen))
	totalData = append(totalData, methodBytes...)

	// 5. 读取参数内容
	paramLen, err = readUint32(reader)
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4008, "error_msg": "read param length failed"}`)
		return
	}
	paramData, err := readBytes(reader, int(paramLen))
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4009, "error_msg": "read param data failed"}`)
		return
	}
	var params map[string]interface{}
	if err = json.Unmarshal(paramData, &params); err != nil {
		sendErrorResponse(conn, `{"error_code": 4003, "error_msg": "invalid parameter format"}`)
		return
	}
	// 拆分追加操作，避免 append 参数过多问题
	totalData = append(totalData, byte(paramLen>>24))
	totalData = append(totalData, byte(paramLen>>16))
	totalData = append(totalData, byte(paramLen>>8))
	totalData = append(totalData, byte(paramLen))
	totalData = append(totalData, paramData...)

	// 8. 读取并验证校验和（新增）
	checksum, err = readUint32(reader)
	if err != nil {
		sendErrorResponse(conn, `{"error_code": 4016, "error_msg": "read checksum failed"}`)
		return
	}
	calculatedChecksum := crc32.ChecksumIEEE(totalData)
	if checksum != calculatedChecksum {
		errorMsg := fmt.Sprintf(`{"error_code": 4017, "error_msg": "checksum verification failed, expected %#x, got %#x"}`, calculatedChecksum, checksum)
		sendErrorResponse(conn, errorMsg)
		return
	}

	// 9. 路由调用（移除文件参数注入）
	// params["files"] = files // 已删除文件信息注入代码
	registryLock.RLock()
	handler, exists := serviceRegistry[method]
	registryLock.RUnlock()

	var result interface{}
	if exists {
		result, err = handler(params)
	} else {
		err = fmt.Errorf("service %s not found", method)
	}

	// 10. 返回响应（保持原有逻辑）
	response, errMarshal := json.Marshal(map[string]interface{}{
		"msg_id": msgID,
		"result": result,
		"error":  err,
	})
	if errMarshal != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4018, "error_msg": "response serialization failed: %v"}`, errMarshal))
		return
	}

	// 构造响应头
	header := make([]byte, 0)
	header = binary.BigEndian.AppendUint16(header, magicNumber)
	header = append(header, version)
	header = binary.BigEndian.AppendUint32(header, uint32(len(response)))
	conn.Write(append(header, response...))
}
