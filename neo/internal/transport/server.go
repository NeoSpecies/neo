package transport

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"neo/internal/config"
	"neo/internal/connection"
	"neo/internal/discovery"
	"neo/internal/ipcprotocol"
	"neo/internal/metrics"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 服务注册表（设计文档服务发现机制）
var serviceRegistry = make(map[string]func(map[string]interface{}) (interface{}, error))
var registryLock sync.RWMutex

// 全局协程池
var workerPool *WorkerPool

// 初始化协程池
func init() {
	workerPool = NewWorkerPool(10, 100)
}

// 注册服务（供 Go/Python 服务调用）
func RegisterService(name string, handler func(map[string]interface{}) (interface{}, error)) {
	registryLock.Lock()
	defer registryLock.Unlock()
	serviceRegistry[name] = handler
}

// 启动 TCP 服务（传输层实现）
func StartIpcServer() error {
	// 1. 获取配置并构建地址
	cfg := config.Get()
	addr := net.JoinHostPort(cfg.IPC.Host, strconv.Itoa(cfg.IPC.Port))

	// 2. 启动IPC服务监听
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("启动IPC服务失败: %v", err)
	}
	defer listener.Close()

	// 3. 定义连接工厂函数
	createConnection := func() (net.Conn, error) {
		return net.Dial("tcp", addr)
	}

	// 4. 初始化连接池（仅传入工厂函数）
	pool, err := connection.NewConnectionPool(createConnection)
	if err != nil {
		return fmt.Errorf("初始化连接池失败: %v", err)
	}
	defer pool.Close()

	// 启动监控服务器 - 移至循环前执行
	if err := metricsInstance.StartServer(); err != nil {
		log.Printf("监控服务器启动失败: %v", err)
		return err // 添加错误返回
	}

	log.Printf("IPC服务已启动，监听地址: %s", addr)

	// 5. 开始接受连接
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err)
			continue
		}
		go handleConnection(conn) // 仅传递conn参数
	}
}

// 处理连接（协议解析核心）
// 新增：服务发现实例（假设通过依赖注入获取）
var discoveryInstance *discovery.Discovery

// 处理连接（协议解析核心）
func handleConnection(conn net.Conn) {
	// 确保连接不会被意外关闭
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理连接时发生panic: %v", r)
		}
		conn.Close()
	}()

	log.Printf("新连接来自: %s", conn.RemoteAddr())
	defer conn.Close()

	// 添加指标收集 - 移至函数开头
	startTime := time.Now()
	var method []byte // 在defer前声明method变量
	defer func() {
		// 记录请求延迟 - 确保method已定义
		if method != nil {
			duration := time.Since(startTime)
			metrics.RecordLatency("ipc", string(method), duration)
			metrics.RecordRequest("ipc", string(method), "success")
		}
	}()

	reader := bufio.NewReader(conn)

	// 1. 读取固定头部（魔数+版本）
	fixedHeader := make([]byte, 3)
	if _, err := io.ReadFull(reader, fixedHeader); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4000, "error_msg": "read fixed header failed: %v"}`, err))
		return
	}
	magic := binary.BigEndian.Uint16(fixedHeader[:2])
	version := fixedHeader[2]
	log.Printf("Debug: 魔数=0x%X, 版本=%d", magic, version)

	// 2. 验证魔数和版本
	if magic != ipcprotocol.MAGIC_NUMBER {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4002, "error_msg": "invalid magic: expected 0x%X, got 0x%X"}`, ipcprotocol.MAGIC_NUMBER, magic))
		return
	}
	if version > ipcprotocol.ProtocolVersion1 {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4003, "error_msg": "unsupported version: %d"}`, version))
		return
	}

	// 3. 读取MsgID（TLV格式）
	var msgIDLen uint16
	if err := binary.Read(reader, binary.BigEndian, &msgIDLen); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4004, "error_msg": "read msgIDLen failed: %v"}`, err))
		return
	}
	log.Printf("Debug: MsgIDLen=%d", msgIDLen)

	msgID := make([]byte, msgIDLen)
	if _, err := io.ReadFull(reader, msgID); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4005, "error_msg": "read msgID failed: %v"}`, err))
		return
	}
	log.Printf("Debug: MsgID=%s", string(msgID))

	// 4. 读取Method（TLV格式）
	var methodLen uint16
	if err := binary.Read(reader, binary.BigEndian, &methodLen); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4006, "error_msg": "read methodLen failed: %v"}`, err))
		return
	}
	log.Printf("Debug: MethodLen=%d", methodLen)

	method = make([]byte, methodLen) // 这里仅赋值，不重新声明
	if _, err := io.ReadFull(reader, method); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4007, "error_msg": "read method failed: %v"}`, err))
		return
	}
	log.Printf("Debug: Method=%s", string(method))

	// 5. 读取Param（TLV格式）
	var paramLen uint32
	if err := binary.Read(reader, binary.BigEndian, &paramLen); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4008, "error_msg": "read paramLen failed: %v"}`, err))
		return
	}
	log.Printf("Debug: ParamLen=%d", paramLen)

	param := make([]byte, paramLen)
	if _, err := io.ReadFull(reader, param); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4009, "error_msg": "read param failed: %v"}`, err))
		return
	}
	log.Printf("Debug: Param=%s", string(param))

	// 6. 读取CRC32
	var crc32Value uint32
	if err := binary.Read(reader, binary.BigEndian, &crc32Value); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4010, "error_msg": "read crc32 failed: %v"}`, err))
		return
	}
	log.Printf("Debug: CRC32=0x%X", crc32Value)

	// 计算校验和（重新构建原始数据）
	rawData := make([]byte, 0)
	rawData = append(rawData, fixedHeader...)
	rawData = binary.BigEndian.AppendUint16(rawData, msgIDLen)
	rawData = append(rawData, msgID...)
	rawData = binary.BigEndian.AppendUint16(rawData, methodLen)
	rawData = append(rawData, method...)
	rawData = binary.BigEndian.AppendUint32(rawData, paramLen)
	rawData = append(rawData, param...)

	calculatedCRC32 := crc32.ChecksumIEEE(rawData)
	if calculatedCRC32 != crc32Value {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4011, "error_msg": "crc32 mismatch: expected 0x%X, got 0x%X"}`, calculatedCRC32, crc32Value))
		return
	}
	log.Printf("Debug: CRC32校验通过")

	// 解析参数JSON
	var params map[string]interface{}
	if err := json.Unmarshal(param, &params); err != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4012, "error_msg": "param json decode failed: %v"}`, err))
		return
	}
	log.Printf("Debug: 解析参数成功: %+v", params)

	// 调用服务注册逻辑
	registryLock.RLock()
	handler, exists := serviceRegistry[string(method)]
	registryLock.RUnlock()

	var result interface{}
	var err error
	if exists {
		result, err = handler(params)
	} else {
		err = fmt.Errorf("service %s not found", method)
	}
	log.Printf("Debug: 服务处理结果: %v, 错误: %v", result, err)

	// 构建响应数据
	responseData := map[string]interface{}{
		"msg_id": string(msgID),
		"result": result,
	}
	if err != nil {
		responseData["error"] = err.Error()
	} else {
		responseData["error"] = nil
	}

	// 序列化响应
	responseBody, errMarshal := json.Marshal(responseData)
	if errMarshal != nil {
		sendErrorResponse(conn, fmt.Sprintf(`{"error_code": 4013, "error_msg": "response marshal failed: %v"}`, errMarshal))
		return
	}

	// 创建响应消息
	responseBytes := ipcprotocol.NewResponse(responseBody)
	if _, writeErr := conn.Write(responseBytes); writeErr != nil {
		log.Printf("发送响应失败: %v", writeErr)
		return
	}
	log.Printf("Debug: 响应发送成功，长度: %d字节", len(responseBytes))
}

// Submit 提交任务到协程池
func (p *WorkerPool) Submit(job func()) {
	p.wg.Add(1)
	p.jobs <- job
}

// 发送错误响应（使用响应格式而非请求格式）
func sendErrorResponse(conn net.Conn, errorMsg string) {
	// 构建标准错误响应结构
	errorResponse := map[string]interface{}{
		"msg_id": uuid.New().String(), // 生成新UUID作为消息ID
		"error":  errorMsg,
		"result": nil,
	}

	// 序列化为JSON
	responseBody, err := json.Marshal(errorResponse)
	if err != nil {
		log.Printf("错误响应序列化失败: %v", err)
		// 发送最基础的错误响应
		minimalErr := []byte(`{"error":"序列化错误","result":null}`)
		conn.Write(ipcprotocol.NewResponse(minimalErr))
		return
	}

	// 使用响应格式发送错误
	responseBytes := ipcprotocol.NewResponse(responseBody)
	if _, writeErr := conn.Write(responseBytes); writeErr != nil {
		log.Printf("发送错误响应失败: %v", writeErr)
	}
}

// 初始化服务发现实例
func init() {
	workerPool = NewWorkerPool(10, 100)
	// 初始化服务发现（使用内存存储）
	storage := discovery.NewInMemoryStorage()
	discoveryInstance = discovery.New(storage)
	// 注册"register"服务处理函数
	RegisterService("register", registerServiceHandler)
}

// 服务注册处理函数
func registerServiceHandler(params map[string]interface{}) (interface{}, error) {
	if discoveryInstance == nil {
		return nil, fmt.Errorf("服务发现实例未初始化")
	}

	// 从嵌套的service对象中解析参数
	serviceData, ok := params["service"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid service data type")
	}

	serviceID, ok := serviceData["id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid service ID type")
	}

	serviceName, ok := serviceData["name"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid service name type")
	}

	address, ok := serviceData["address"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid address type")
	}

	portFloat, ok := serviceData["port"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid port type")
	}
	port := int(portFloat)

	// 转换metadata类型: map[string]interface{} -> map[string]string
	metadata := make(map[string]string)
	if meta, ok := params["metadata"].(map[string]interface{}); ok {
		for k, v := range meta {
			metadata[k] = fmt.Sprintf("%v", v)
		}
	}

	service := &discovery.Service{
		ID:        serviceID,
		Name:      serviceName,
		Address:   address,
		Port:      port,
		Status:    "healthy",
		Metadata:  metadata,
		UpdatedAt: time.Now(),
		ExpireAt:  time.Now().Add(30 * time.Second),
	}

	// 调用服务发现注册方法
	if err := discoveryInstance.Register(context.Background(), service); err != nil {
		return nil, fmt.Errorf("服务注册失败: %v", err)
	}

	// 返回包含服务ID的响应
	return map[string]interface{}{
		"status":  "success",
		"message": "服务注册成功",
		"id":      serviceID,
	}, nil
}

// 全局监控实例
var metricsInstance *metrics.Metrics

// 初始化监控
func init() {
	metricsInstance = metrics.NewMetrics()
}

// 新增TCPServer结构体封装所有依赖
type TCPServer struct {
	listener   net.Listener
	pool       *connection.ConnectionPool
	balancer   connection.Balancer
	workerPool *WorkerPool
	metrics    *metrics.Metrics
	config     *config.IPCConfig
	mu         sync.RWMutex
	isRunning  bool
}

// 构造函数实现依赖注入
func NewTCPServer(cfg *config.IPCConfig, balancer connection.Balancer) (*TCPServer, error) {
	// 初始化所有依赖组件
	workerCount := 100 // 使用默认值100，因为config.IPCConfig中没有WorkerCount字段
	workerPool := NewWorkerPool(10, workerCount)
	metrics := metrics.NewMetrics()

	return &TCPServer{
		config:     cfg,
		balancer:   balancer,
		workerPool: workerPool,
		metrics:    metrics,
		isRunning:  false,
	}, nil
}

// 拆分超大函数为独立方法
func (s *TCPServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return errors.New("server already running")
	}

	addr := net.JoinHostPort(s.config.Host, strconv.Itoa(s.config.Port))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen failed: %w", err)
	}
	s.listener = listener
	s.isRunning = true

	go s.acceptLoop()
	return nil
}

// 独立的连接接受循环
func (s *TCPServer) acceptLoop() {
	for s.isRunning {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Printf("接受连接失败: %v", err) // 替换metrics.RecordError
			continue
		}
		s.workerPool.Submit(func() {
			handleConnection(conn)
		})
	}
}
