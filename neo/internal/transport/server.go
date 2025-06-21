package transport

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"neo/internal/discovery"

	// "neo/internal/connection"
	// "neo/internal/discovery"
	// "neo/internal/config"
	"hash/crc32"
	"io"

	"log"
	"net"
	"sync"
	// "strconv"
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

// 删除以下连接池初始化代码块
/*
// 全局连接池
var connectionPool *connection.ConnectionPool

// 初始化连接池
func init() {
    // 创建连接池（实际实现需根据项目需求调整）
    factory := func() (net.Conn, error) {
        cfg := config.Get()
        return net.Dial("tcp", net.JoinHostPort(cfg.IPC.Host, strconv.Itoa(cfg.IPC.Port)))
    }
    var err error
    connectionPool, err = connection.NewConnectionPool(factory)
    if err != nil {
        log.Fatalf("Failed to create connection pool: %v", err)
    }
}
*/

// 注册服务（供 Go/Python 服务调用）
func RegisterService(name string, handler func(map[string]interface{}) (interface{}, error)) {
	registryLock.Lock()
	defer registryLock.Unlock()
	serviceRegistry[name] = handler
}

// 启动 TCP 服务（传输层实现）
// 修改StartIpcServer函数签名，添加启动完成回调
func StartIpcServer(addr string, onStarted func()) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	fmt.Printf("IPC Server listening on %s\n", addr)
	if onStarted != nil {
		onStarted() // 服务器启动成功后调用回调
	}

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
// 新增：服务发现实例（假设通过依赖注入获取）
var discoveryInstance *discovery.Discovery

// 修改handleConnection函数，处理服务注册请求
func handleConnection(conn net.Conn) {
    // 立即发送协议魔数 - 删除这行代码
    // if _, err := conn.Write([]byte{0xAE, 0xBD}); err != nil {
    //     log.Printf("发送魔数失败: %v", err)
    //     return
    // }

    // 确保连接不会被意外关闭
    defer func() {
        if r := recover(); r != nil {
            log.Printf("处理连接时发生panic: %v", r)
        }
        conn.Close()
    }()

	// 先读取并解析参数
	log.Printf("新连接来自: %s", conn.RemoteAddr())
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
	// 删除以下错误代码块
	// 兼容test.py的参数格式
	// if action, ok := params["action"]; ok && action == "register" {
	//     if serviceData, ok := params["service"].(map[string]interface{}); ok {
	//         params = serviceData
	//     }
	// }
	
	// 保留参数验证，但修改为检查原始params
	if _, ok := params["service"]; !ok {
	    sendErrorResponse(conn, `{"error_code": 4010, "error_msg": "missing required field: service"}`)
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

	// 10. 返回响应（修改为包含 msgID）
	responseData := map[string]interface{}{
		"msg_id": msgID,
		"result": result,
	}
	// 显式处理错误字段
	if err != nil {
		responseData["error"] = err.Error()
	} else {
		responseData["error"] = nil // 确保无错误时显式设置为null
	}

	response, errMarshal := json.Marshal(responseData)
	if errMarshal != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4018, "error_msg": "response serialization failed: %v"}`, errMarshal))
		return
	}

	// 构造响应头（发送响应头字节错误）
	// 修改响应头构造逻辑
	// 构造响应头（修复版本号字节错误）
	header := make([]byte, 7) // 2字节魔数 + 1字节版本 + 4字节长度 = 7字节
	// 魔数（2字节大端）
	binary.BigEndian.PutUint16(header[0:2], MAGIC_NUMBER)
	// 版本（1字节）
	header[2] = VERSION
	// 响应体长度（4字节大端）
	binary.BigEndian.PutUint32(header[3:7], uint32(len(response)))
	conn.Write(append(header, response...))
}

// 读取 uint16 类型数据
func readUint16(reader io.Reader) (uint16, error) {
	var num uint16
	err := binary.Read(reader, binary.BigEndian, &num)
	return num, err
}

// 读取指定长度的字节数据
func readBytes(reader io.Reader, length int) ([]byte, error) {
	data := make([]byte, length)
	_, err := io.ReadFull(reader, data)
	return data, err
}

// 读取 uint32 类型数据
func readUint32(reader io.Reader) (uint32, error) {
	var num uint32
	err := binary.Read(reader, binary.BigEndian, &num)
	return num, err
}

// 删除未使用的handleRequest函数
// func handleRequest(conn net.Conn) {
// 	// 解析客户端请求（假设请求格式包含msg_id）
// 	var req struct {
// 		MsgID string `json:"msg_id"`
// 	}
// 	if err := json.NewDecoder(conn).Decode(&req); err != nil {
// 		// 错误处理...
// 	}
// 	msgID := req.MsgID
// 	// 返回成功响应
// 	responseData, _ := json.Marshal(map[string]interface{}{
// 		"msg_id": msgID,
// 		"result": "服务注册成功",
// 	})
// 	// 发送响应到客户端
// 	_, err := conn.Write(responseData)
// 	if err != nil {
// 		log.Printf("发送响应失败: %v", err)
// 	}
// 	return
// }

// 新增WorkerPool实现
// WorkerPool 协程池实现
type WorkerPool struct {
	workerCount int
	jobs        chan func()
	wg          sync.WaitGroup
}

// NewWorkerPool 创建新的协程池
func NewWorkerPool(workerCount int) *WorkerPool {
	pool := &WorkerPool{
		workerCount: workerCount,
		jobs:        make(chan func(), 100), // 缓冲区大小可根据需求调整
	}

	// 启动工作协程
	for i := 0; i < workerCount; i++ {
		go func() {
			for job := range pool.jobs {
				job()
				pool.wg.Done()
			}
		}()
	}

	return pool
}

// Submit 提交任务到协程池
func (p *WorkerPool) Submit(job func()) {
	p.wg.Add(1)
	p.jobs <- job
}

// 协议常量（与test.py完全一致）
const (
    MAGIC_NUMBER = 0xAEBD  // 2字节大端魔数
    VERSION      = 0x01    // 1字节协议版本
)
