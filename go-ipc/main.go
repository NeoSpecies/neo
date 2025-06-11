package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"go-ipc/config"
	"go-ipc/config/loader"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
)

// 新增：异步任务状态枚举
type AsyncTaskStatus int

const (
	TaskPending AsyncTaskStatus = iota
	TaskSuccess
	TaskFailed
)

// 新增：异步任务结构体
type AsyncTask struct {
	TaskID    string
	Status    AsyncTaskStatus
	Result    interface{}
	Error     error
	Callback  func(interface{}, error) // 回调函数
	CreatedAt time.Time
}

var (
	connPool *ConnPool
	// 新增：异步任务存储（带锁）
	asyncTasks = struct {
		sync.RWMutex
		tasks map[string]*AsyncTask
	}{tasks: make(map[string]*AsyncTask)}
)

// 初始化协程池
func init() {
	workerPool = NewWorkerPool(100)
	// 移除连接池初始化逻辑
}

// 新增：异步任务提交方法（替代原有同步调用）
func CallAsync(method string, params map[string]interface{}, callback func(interface{}, error)) string {
	taskID := uuid.New().String()
	task := &AsyncTask{
		TaskID:    taskID,
		Status:    TaskPending,
		Callback:  callback,
		CreatedAt: time.Now(),
	}

	asyncTasks.Lock()
	asyncTasks.tasks[taskID] = task
	asyncTasks.Unlock()

	workerPool.Submit(func() {
		// 模拟耗时操作（实际应调用服务逻辑）
		time.Sleep(2 * time.Second)

		// 假设调用结果
		result, err := callPythonIpcService(method, params)

		// 更新任务状态
		asyncTasks.Lock()
		task.Status = TaskSuccess
		if err != nil {
			task.Status = TaskFailed
			task.Error = err
		}
		task.Result = result
		asyncTasks.Unlock()

		// 执行回调
		if task.Callback != nil {
			task.Callback(result, err)
		}
	})

	return taskID
}

// 新增：轮询任务结果接口
func handleAsyncResult(w http.ResponseWriter, r *http.Request) {
	var req struct{ TaskID string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "无效请求", 400)
		return
	}

	asyncTasks.RLock()
	task, exists := asyncTasks.tasks[req.TaskID]
	asyncTasks.RUnlock()

	if !exists {
		http.Error(w, "任务不存在", 404)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"task_id": req.TaskID,
		"status":  task.Status,
		"result":  task.Result,
		"error":   task.Error,
	})
}

func main() {
	// 加载配置文件（提升到main函数顶部，避免重复加载）
	var cfg config.GlobalConfig
	if err := loader.LoadFromFile("config/default.yml", &cfg); err != nil {
		fmt.Printf("加载配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 初始化全局配置（修正：完整赋值IPC和HTTP配置）
	var globalCfg config.GlobalConfig
	globalCfg.IPC = cfg.IPC   // 从config.Config获取IPC配置
	globalCfg.HTTP = cfg.HTTP // 从config.Config获取HTTP配置
	config.Update(globalCfg)

	// 在配置加载完成后初始化连接池
	connPool = NewConnPool()

	serverCfg := config.Get().HTTP // HTTP服务配置

	// 注册 Go 测试函数（供 Python 调用）
	RegisterService("go.service.test", func(params map[string]interface{}) (interface{}, error) {
		return fmt.Sprintf("Go 测试函数返回：%v", params["input"]), nil
	})

	// 启动 IPC 服务（修正：使用IPC配置参数）
	go func() {
		fmt.Println("正在启动 IPC 服务...")
		ipcCfg := config.Get().IPC                                // 获取IPC配置
		ipcAddr := fmt.Sprintf("%s:%d", ipcCfg.Host, ipcCfg.Port) // 使用IPC配置中的host和port
		if err := StartIpcServer(ipcAddr); err != nil {
			fmt.Printf("IPC 服务启动失败: %v\n", err)
			os.Exit(1)
		}
	}()

	// 等待 IPC 服务启动
	time.Sleep(time.Second)

	// 处理 HTTP 请求
	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		// fmt.Printf("接收到 HTTP 请求：%s %s\n", r.Method, r.URL.Path)

		var req struct{ Input string }
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			// fmt.Printf("解码请求体失败：%v\n", err)
			http.Error(w, "无效的请求体", 400)
			return
		}

		pythonResult, err := callPythonIpcService("python.service.demo", map[string]interface{}{"input": req.Input})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		if pythonResult == nil {
			http.Error(w, "Python 服务返回空结果", 500)
			return
		}

		// fmt.Printf("Python 返回结果类型：%T\n", pythonResult)

		result, ok := pythonResult.(map[string]interface{})
		if !ok {
			http.Error(w, fmt.Sprintf("Python 响应类型错误，预期 map[string]interface{}，实际类型：%T", pythonResult), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "HTTP 请求处理完成",
			"result":  result,
		})
	})

	// 移除文件上传路由
	// http.HandleFunc("/upload", handleFileUpload)

	fmt.Printf("HTTP 服务启动，监听地址 %s:%d\n", serverCfg.Host, serverCfg.Port)
	// 注册异步结果轮询路由
	http.HandleFunc("/async_result", handleAsyncResult)

	// 使用配置中的地址和端口
	http.ListenAndServe(fmt.Sprintf("%s:%d", serverCfg.Host, serverCfg.Port), nil)
}

// 修改IPC调用函数，使用连接池和压缩
func callPythonIpcService(method string, params map[string]interface{}) (interface{}, error) {
	// fmt.Printf("[DEBUG] 开始调用 Python 服务: method=%s, params=%+v\n", method, params)

	// 从连接池获取连接
	conn, err := connPool.Get("127.0.0.1:9091")
	if err != nil {
		// fmt.Printf("[ERROR] 连接 Python IPC 服务失败: %v\n", err)
		return nil, fmt.Errorf("连接 Python IPC 服务失败: %v", err)
	}
	defer connPool.Put("127.0.0.1:9091", conn)
	// fmt.Printf("[DEBUG] 成功获取连接池连接\n")

	// 序列化参数
	paramData, err := json.Marshal(params)
	if err != nil {
		// fmt.Printf("[ERROR] 参数序列化失败: %v\n", err)
		return nil, fmt.Errorf("参数序列化失败: %v", err)
	}
	// fmt.Printf("[DEBUG] 参数序列化成功，长度: %d bytes\n", len(paramData))

	// 检查是否需要压缩
	if ShouldCompress(paramData) {
		// fmt.Printf("[DEBUG] 数据需要压缩，原始大小: %d bytes\n", len(paramData))
		compressedData, compressErr := CompressData(paramData) // 重命名为 compressErr
		if compressErr != nil {
			// fmt.Printf("[ERROR] 压缩数据失败: %v\n", compressErr)
			return nil, fmt.Errorf("压缩数据失败: %v", compressErr)
		}
		paramData = compressedData
		// fmt.Printf("[DEBUG] 压缩后大小: %d bytes\n", len(paramData))
	}

	msgID := []byte(uuid.New().String())
	request := new(bytes.Buffer)
	totalData := new(bytes.Buffer)

	// 写入协议头
	magic := uint16(0xAEBD)
	if err = binary.Write(request, binary.BigEndian, magic); err != nil {
		// fmt.Printf("[ERROR] 写入魔数失败: %v\n", err)
		return nil, fmt.Errorf("写入魔数失败: %v", err)
	}
	binary.Write(totalData, binary.BigEndian, magic)
	// fmt.Printf("[DEBUG] 写入魔数: 0x%04X\n", magic)

	version := byte(0x01)
	if err = request.WriteByte(version); err != nil {
		// fmt.Printf("[ERROR] 写入版本失败: %v\n", err)
		return nil, fmt.Errorf("写入版本失败: %v", err)
	}
	totalData.WriteByte(version)
	// fmt.Printf("[DEBUG] 写入版本: %d\n", version)

	// 写入消息ID
	msgIDLen := uint16(len(msgID))
	if err = binary.Write(request, binary.BigEndian, msgIDLen); err != nil {
		// fmt.Printf("[ERROR] 写入消息ID长度失败: %v\n", err)
		return nil, fmt.Errorf("写入消息ID长度失败: %v", err)
	}
	if _, writeMsgIDErr := request.Write(msgID); writeMsgIDErr != nil { // 重命名为 writeMsgIDErr
		// fmt.Printf("[ERROR] 写入消息ID失败: %v\n", writeMsgIDErr)
		return nil, fmt.Errorf("写入消息ID失败: %v", writeMsgIDErr)
	}
	binary.Write(totalData, binary.BigEndian, msgIDLen)
	totalData.Write(msgID)
	// fmt.Printf("[DEBUG] 写入消息ID: %s\n", string(msgID))

	// 写入方法名
	methodBytes := []byte(method)
	methodLen := uint16(len(methodBytes))
	if err = binary.Write(request, binary.BigEndian, methodLen); err != nil {
		// fmt.Printf("[ERROR] 写入方法名长度失败: %v\n", err)
		return nil, fmt.Errorf("写入方法名长度失败: %v", err)
	}
	if _, writeMethodErr := request.Write(methodBytes); writeMethodErr != nil { // 重命名为 writeMethodErr
		// fmt.Printf("[ERROR] 写入方法名失败: %v\n", writeMethodErr)
		return nil, fmt.Errorf("写入方法名失败: %v", writeMethodErr)
	}
	binary.Write(totalData, binary.BigEndian, methodLen)
	totalData.Write(methodBytes)
	// fmt.Printf("[DEBUG] 写入方法名: %s\n", method)

	// 写入参数内容
	paramLen := uint32(len(paramData))
	if err = binary.Write(request, binary.BigEndian, paramLen); err != nil {
		// fmt.Printf("[ERROR] 写入参数长度失败: %v\n", err)
		return nil, fmt.Errorf("写入参数长度失败: %v", err)
	}
	if _, writeParamErr := request.Write(paramData); writeParamErr != nil { // 重命名为 writeParamErr
		// fmt.Printf("[ERROR] 写入参数内容失败: %v\n", err)
		return nil, fmt.Errorf("写入参数内容失败: %v", err)
	}
	binary.Write(totalData, binary.BigEndian, paramLen)
	totalData.Write(paramData)
	// fmt.Printf("[DEBUG] 写入参数，长度: %d bytes\n", paramLen)

	// 写入文件数量（0）
	fileCount := uint16(0)
	if err = binary.Write(request, binary.BigEndian, fileCount); err != nil {
		// fmt.Printf("[ERROR] 写入文件数量失败: %v\n", err)
		return nil, fmt.Errorf("写入文件数量失败: %v", err)
	}
	binary.Write(totalData, binary.BigEndian, fileCount)
	// fmt.Printf("[DEBUG] 写入文件数量: %d\n", fileCount)

	// 计算校验和
	checksum := crc32.ChecksumIEEE(totalData.Bytes())
	if err = binary.Write(request, binary.BigEndian, checksum); err != nil {
		// fmt.Printf("[ERROR] 写入校验和失败: %v\n", err)
		return nil, fmt.Errorf("写入校验和失败: %v", err)
	}
	// fmt.Printf("[DEBUG] 写入校验和: 0x%08X\n", checksum)

	// 发送请求
	if _, sendErr := conn.Write(request.Bytes()); sendErr != nil { // 重命名为 sendErr
		// fmt.Printf("[ERROR] 发送请求失败: %v\n", sendErr)
		return nil, fmt.Errorf("发送请求失败: %v", sendErr)
	}
	// fmt.Printf("[DEBUG] 成功发送请求，总长度: %d bytes\n", len(request.Bytes()))

	reader := bufio.NewReader(conn)
	magic, err = readUint16(reader)
	if err != nil || magic != 0xAEBD {
		// fmt.Printf("[ERROR] 响应魔数无效: 期望=0xAEBD, 实际=0x%04X, 错误=%v\n", magic, err)
		return nil, fmt.Errorf("响应魔数无效: %v", err)
	}
	// fmt.Printf("[DEBUG] 读取响应魔数: 0x%04X\n", magic)

	version, err = reader.ReadByte()
	if err != nil || version > 1 {
		// fmt.Printf("[ERROR] 不支持的响应版本: %d, 错误=%v\n", version, err)
		return nil, fmt.Errorf("不支持的响应版本: %v", version)
	}
	// fmt.Printf("[DEBUG] 读取响应版本: %d\n", version)

	bodyLen, err := readUint32(reader)
	if err != nil {
		// fmt.Printf("[ERROR] 读取响应体长度失败: %v\n", err)
		return nil, fmt.Errorf("读取响应体长度失败: %v", err)
	}
	// fmt.Printf("[DEBUG] 读取响应体长度: %d bytes\n", bodyLen)

	bodyData, err := readBytes(reader, int(bodyLen))
	if err != nil {
		// fmt.Printf("[ERROR] 读取响应体失败: %v\n", err)
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}
	// fmt.Printf("[DEBUG] 成功读取响应体，长度: %d bytes\n", len(bodyData))

	// 检查是否需要解压
	if ShouldCompress(bodyData) {
		fmt.Printf("[DEBUG] 响应数据需要解压，压缩大小: %d bytes\n", len(bodyData))
		decompressedData, err := DecompressData(bodyData)
		if err != nil {
			fmt.Printf("[ERROR] 解压数据失败: %v\n", err)
			return nil, fmt.Errorf("解压数据失败: %v", err)
		}
		bodyData = decompressedData
		// fmt.Printf("[DEBUG] 解压后大小: %d bytes\n", len(bodyData))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(bodyData, &response); err != nil {
		// fmt.Printf("[ERROR] 响应反序列化失败: %v\n", err)
		// fmt.Printf("[DEBUG] 响应体内容: %s\n", string(bodyData))
		return nil, fmt.Errorf("响应反序列化失败: %v", err)
	}
	// fmt.Printf("[DEBUG] 响应反序列化成功: %+v\n", response)

	if response["error"] != nil {
		fmt.Printf("[ERROR] Python 服务返回错误: %v\n", response["error"])
		return nil, fmt.Errorf("Python 服务错误: %v", response["error"])
	}

	// fmt.Printf("[DEBUG] 成功获取 Python 服务响应: %+v\n", response["result"])
	return response["result"], nil
}

// 辅助函数：按大端序读取 2 字节为 uint16
func readUint16(reader *bufio.Reader) (uint16, error) {
	b := make([]byte, 2)
	if _, err := io.ReadFull(reader, b); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint16(b), nil
}

// 辅助函数：按大端序读取 4 字节为 uint32
func readUint32(reader *bufio.Reader) (uint32, error) {
	b := make([]byte, 4)
	if _, err := io.ReadFull(reader, b); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b), nil
}

// 辅助函数：读取指定长度的字节
func readBytes(reader *bufio.Reader, length int) ([]byte, error) {
	b := make([]byte, length)
	if _, err := io.ReadFull(reader, b); err != nil {
		return nil, err
	}
	return b, nil
}
