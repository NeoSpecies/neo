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
	"net/http"
	"os"
	"sync"
	"time"

	"go-ipc/discovery"

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
	// 新增：ServiceDiscovery 单例实例和 sync.Once 变量
	sdInstance *discovery.ServiceDiscovery
	sdOnce     sync.Once
	// 新增：异步任务存储（带锁）声明
	asyncTasks struct {
		sync.RWMutex
		tasks map[string]*AsyncTask
	}
)

// 获取 ServiceDiscovery 单例实例的函数
// 获取 ServiceDiscovery 单例实例的函数
func GetServiceDiscovery() *discovery.ServiceDiscovery {
	sdOnce.Do(func() {
		// 移除对 ETCDEndpoints 的引用，直接调用无参数的 GetInstance()
		sdInstance = discovery.GetInstance()
	})
	return sdInstance
}

// 初始化协程池和异步任务存储
func init() {
	workerPool = NewWorkerPool(100)
	// 移除连接池初始化逻辑
	// 初始化异步任务存储
	asyncTasks.tasks = make(map[string]*AsyncTask)
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

	// 获取 ServiceDiscovery 单例实例并启动服务发现

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
		var req struct{ Input string }
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
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
	sd := GetServiceDiscovery()
	// 获取所有服务
	services, err := sd.GetServices("python.service.demo")
	if err != nil {
		fmt.Printf("获取服务列表失败: %v\n", err)
	} else {
		fmt.Println("已注册的服务列表:")
		for _, service := range services {
			fmt.Printf("名称: %s, ID: %s, 地址: %s:%d\n", service.Name, service.ID, service.Address, service.Port)
		}
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("未找到可用的 Python 服务")
	}

	// 简单选择第一个服务
	service := services[0]
	serviceAddr := fmt.Sprintf("%s:%d", service.Address, service.Port)

	// 从连接池获取连接
	conn, err := connPool.Get(serviceAddr)
	if err != nil {
		return nil, fmt.Errorf("连接 Python IPC 服务失败: %v", err)
	}
	defer connPool.Put(serviceAddr, conn)

	// 序列化参数
	paramData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("参数序列化失败: %v", err)
	}

	// 检查是否需要压缩
	if ShouldCompress(paramData) {
		compressedData, compressErr := CompressData(paramData)
		if compressErr != nil {
			return nil, fmt.Errorf("压缩数据失败: %v", compressErr)
		}
		paramData = compressedData
	}

	msgID := []byte(uuid.New().String())
	request := new(bytes.Buffer)
	totalData := new(bytes.Buffer)

	// 写入协议头
	magic := uint16(0xAEBD)
	if err = binary.Write(request, binary.BigEndian, magic); err != nil {
		return nil, fmt.Errorf("写入魔数失败: %v", err)
	}
	binary.Write(totalData, binary.BigEndian, magic)

	version := byte(0x01)
	if err = request.WriteByte(version); err != nil {
		return nil, fmt.Errorf("写入版本失败: %v", err)
	}
	totalData.WriteByte(version)

	// 写入消息ID
	msgIDLen := uint16(len(msgID))
	if err = binary.Write(request, binary.BigEndian, msgIDLen); err != nil {
		return nil, fmt.Errorf("写入消息ID长度失败: %v", err)
	}
	if _, writeMsgIDErr := request.Write(msgID); writeMsgIDErr != nil {
		return nil, fmt.Errorf("写入消息ID失败: %v", writeMsgIDErr)
	}
	binary.Write(totalData, binary.BigEndian, msgIDLen)
	totalData.Write(msgID)

	// 写入方法名
	methodBytes := []byte(method)
	methodLen := uint16(len(methodBytes))
	if err = binary.Write(request, binary.BigEndian, methodLen); err != nil {
		return nil, fmt.Errorf("写入方法名长度失败: %v", err)
	}
	if _, writeMethodErr := request.Write(methodBytes); writeMethodErr != nil {
		return nil, fmt.Errorf("写入方法名失败: %v", writeMethodErr)
	}
	binary.Write(totalData, binary.BigEndian, methodLen)
	totalData.Write(methodBytes)

	// 写入参数内容
	paramLen := uint32(len(paramData))
	if err = binary.Write(request, binary.BigEndian, paramLen); err != nil {
		return nil, fmt.Errorf("写入参数长度失败: %v", err)
	}
	if _, writeParamErr := request.Write(paramData); writeParamErr != nil {
		return nil, fmt.Errorf("写入参数内容失败: %v", err)
	}
	binary.Write(totalData, binary.BigEndian, paramLen)
	totalData.Write(paramData)

	// 写入文件数量（0）
	fileCount := uint16(0)
	if err = binary.Write(request, binary.BigEndian, fileCount); err != nil {
		return nil, fmt.Errorf("写入文件数量失败: %v", err)
	}
	binary.Write(totalData, binary.BigEndian, fileCount)

	// 计算校验和
	checksum := crc32.ChecksumIEEE(totalData.Bytes())
	if err = binary.Write(request, binary.BigEndian, checksum); err != nil {
		return nil, fmt.Errorf("写入校验和失败: %v", err)
	}

	// 发送请求
	if _, sendErr := conn.Write(request.Bytes()); sendErr != nil {
		return nil, fmt.Errorf("发送请求失败: %v", sendErr)
	}

	reader := bufio.NewReader(conn)
	magic, err = readUint16(reader)
	if err != nil || magic != 0xAEBD {
		return nil, fmt.Errorf("响应魔数无效: %v", err)
	}

	version, err = reader.ReadByte()
	if err != nil || version > 1 {
		return nil, fmt.Errorf("不支持的响应版本: %v", version)
	}

	bodyLen, err := readUint32(reader)
	if err != nil {
		return nil, fmt.Errorf("读取响应体长度失败: %v", err)
	}

	bodyData, err := readBytes(reader, int(bodyLen))
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// 检查是否需要解压
	if ShouldCompress(bodyData) {
		decompressedData, err := DecompressData(bodyData)
		if err != nil {
			return nil, fmt.Errorf("解压数据失败: %v", err)
		}
		bodyData = decompressedData
	}

	var response map[string]interface{}
	if err := json.Unmarshal(bodyData, &response); err != nil {
		return nil, fmt.Errorf("响应反序列化失败: %v", err)
	}

	if response["error"] != nil {
		return nil, fmt.Errorf("Python 服务错误: %v", response["error"])
	}

	return response["result"], nil
}
