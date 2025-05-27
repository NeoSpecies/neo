package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net"
	"net/http"

	"github.com/google/uuid"
)

// 注册 Go 测试函数（供 Python 调用）
func init() {
	RegisterService("go.service.test", func(params map[string]interface{}) (interface{}, error) {
		fmt.Printf("Go 服务接收到参数：%+v\n", params)
		return fmt.Sprintf("Go 测试函数返回：%v", params["input"]), nil
	})
}

func main() {
	// 启动 IPC 服务
	go func() {
		if err := StartIpcServer("127.0.0.1:9090"); err != nil {
			fmt.Printf("IPC 服务启动失败: %v\n", err)
		}
	}()

	// 处理 HTTP 请求（合并原第一个 main 函数的逻辑）
	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("接收到 HTTP 请求：%s %s\n", r.Method, r.URL.Path)

		var req struct{ Input string }
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			fmt.Printf("解码请求体失败：%v\n", err)
			http.Error(w, "无效的请求体", 400)
			return
		}

		pythonResult, err := callPythonIpcService("python.service.demo", map[string]interface{}{"input": req.Input})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// 新增：检查 Python 结果是否为 nil
		if pythonResult == nil {
			http.Error(w, "Python 服务返回空结果", 500)
			return
		}

		// 新增：打印 Python 返回结果的实际类型（调试用）
		fmt.Printf("Python 返回结果类型：%T\n", pythonResult)

		// 防御性检查：结果是否为 map 类型
		result, ok := pythonResult.(map[string]interface{})
		if !ok {
			// 错误信息中包含实际类型，便于定位
			http.Error(w, fmt.Sprintf("Python 响应类型错误，预期 map[string]interface{}，实际类型：%T", pythonResult), 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "HTTP 请求处理完成",
			"result":  result,
		})
	})

	// 处理文件上传
	http.HandleFunc("/upload", handleFileUpload)

	// 启动 HTTP 服务（合并端口监听）
	fmt.Println("HTTP 服务启动，监听端口 8080")
	http.ListenAndServe("127.0.0.1:8080", nil)
}

// 新增：HTTP 处理文件上传（原逻辑保留）
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	// 1. 解析 Apipost 上传的文件
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "文件解析失败: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "读取文件失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. 构造 IPC 请求参数（传递给 Python）
	ipcParams := map[string]interface{}{
		"files": []map[string]interface{}{
			{
				"meta": map[string]interface{}{
					"original_name": header.Filename,
					"mimetype":      header.Header.Get("Content-Type"),
				},
				"content": content,
			},
		},
	}

	// 4. 通过 IPC 调用 Python 服务处理文件
	pythonResult, err := callPythonIpcService("python.service.fileProcess", ipcParams)
	if err != nil {
		http.Error(w, "IPC 调用失败: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 防御性检查：结果是否为 nil
	if pythonResult == nil {
		http.Error(w, "Python 服务返回空结果", http.StatusInternalServerError)
		return
	}

	// 安全类型断言
	resultMap, ok := pythonResult.(map[string]interface{})
	if !ok {
		http.Error(w, "Python 响应类型错误，非预期的 map 类型", http.StatusInternalServerError)
		return
	}

	processedFileVal := resultMap["processed_file"]
	if processedFileVal == nil {
		http.Error(w, "Python 响应中未包含 processed_file", http.StatusInternalServerError)
		return
	}

	var processedFile map[string]interface{}                      // Declare variable
	processedFile, ok = processedFileVal.(map[string]interface{}) // Use = instead of :=
	if !ok {
		http.Error(w, "Python 响应中 processed_file 非预期的 map 类型", http.StatusInternalServerError)
		return
	}

	// 提取字段时增加类型检查（示例：content 应为 Base64 字符串，需解码为字节数组）
	contentStr, ok := processedFile["content"].(string)
	if !ok {
		http.Error(w, "Python 响应中 content 非预期的字符串类型", http.StatusInternalServerError)
		return
	}

	// 解码 Base64 字符串为字节数组

	fileContent, err := base64.StdEncoding.DecodeString(contentStr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Base64 解码失败: %v", err), http.StatusInternalServerError)
		return
	}

	mimetype, ok := processedFile["mimetype"].(string)
	if !ok {
		http.Error(w, "Python 响应中 mimetype 非预期的字符串", http.StatusInternalServerError)
		return
	}
	newName, ok := processedFile["new_name"].(string)
	if !ok {
		http.Error(w, "Python 响应中 new_name 非预期的字符串", http.StatusInternalServerError)
		return
	}

	// 最终返回文件
	w.Header().Set("Content-Type", mimetype)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", newName))
	w.Write(fileContent) // Use renamed variable
}

// 新增：IPC 调用 Python 服务的函数（简化版，需与协议匹配）
// 新增：计算CRC32校验和的辅助函数
func calculateChecksum(data []byte) uint32 {
	// 导入 crc32 包以解决 undefined 错误

	return crc32.ChecksumIEEE(data)
}

// 新增：累积请求数据的辅助变量（在构造请求时使用）
func callPythonIpcService(method string, params map[string]interface{}) (interface{}, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:9091")
	if err != nil {
		return nil, fmt.Errorf("连接 Python IPC 服务失败: %v", err)
	}
	defer conn.Close()

	// 1. 打包请求（与 Python 端 call_go_service 协议一致）
	paramData, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("参数序列化失败: %v", err)
	}
	msgID := []byte(uuid.New().String())

	request := new(bytes.Buffer)
	var totalData []byte // 用于计算校验和的完整数据

	// 魔数（2字节，大端）
	magic := uint16(0xAEBD)
	if err := binary.Write(request, binary.BigEndian, magic); err != nil {
		return nil, fmt.Errorf("写入魔数失败: %v", err)
	}
	totalData = append(totalData, byte(magic>>8), byte(magic)) // 记录魔数到总数据

	// 版本（1字节）
	version := byte(0x01)
	if err := request.WriteByte(version); err != nil {
		return nil, fmt.Errorf("写入版本失败: %v", err)
	}
	totalData = append(totalData, version) // 记录版本到总数据

	// 消息 ID 长度（2字节）+ 消息 ID
	msgIDLen := uint16(len(msgID))
	if err := binary.Write(request, binary.BigEndian, msgIDLen); err != nil {
		return nil, fmt.Errorf("写入消息ID长度失败: %v", err)
	}
	if _, err := request.Write(msgID); err != nil {
		return nil, fmt.Errorf("写入消息ID失败: %v", err)
	}
	totalData = append(totalData, byte(msgIDLen>>8), byte(msgIDLen)) // 记录消息ID长度
	totalData = append(totalData, msgID...)                          // 记录消息ID内容

	// 方法名长度（2字节）+ 方法名
	methodBytes := []byte(method)
	methodLen := uint16(len(methodBytes))
	if err := binary.Write(request, binary.BigEndian, methodLen); err != nil {
		return nil, fmt.Errorf("写入方法名长度失败: %v", err)
	}
	if _, err := request.Write(methodBytes); err != nil {
		return nil, fmt.Errorf("写入方法名失败: %v", err)
	}
	totalData = append(totalData, byte(methodLen>>8), byte(methodLen)) // 记录方法名长度
	totalData = append(totalData, methodBytes...)                      // 记录方法名内容

	// 参数长度（4字节）+ 参数内容
	paramLen := uint32(len(paramData))
	if err := binary.Write(request, binary.BigEndian, paramLen); err != nil {
		return nil, fmt.Errorf("写入参数长度失败: %v", err)
	}
	if _, err := request.Write(paramData); err != nil {
		return nil, fmt.Errorf("写入参数内容失败: %v", err)
	}
	totalData = append(totalData, byte(paramLen>>24), byte(paramLen>>16), byte(paramLen>>8), byte(paramLen)) // 记录参数长度
	totalData = append(totalData, paramData...)                                                              // 记录参数内容

	// 计算并写入校验和（4字节，大端）
	checksum := calculateChecksum(totalData)
	if err := binary.Write(request, binary.BigEndian, checksum); err != nil {
		return nil, fmt.Errorf("写入校验和失败: %v", err)
	}

	// 发送请求
	if _, err := conn.Write(request.Bytes()); err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}

	// 2. 解析响应（与 server.go 协议一致）
	reader := bufio.NewReader(conn) // Now uses correct bufio import
	// 读取魔数（2字节）
	magic, err = readUint16(reader)
	if err != nil || magic != 0xAEBD {
		return nil, fmt.Errorf("响应魔数无效: %v", err)
	}
	// 读取版本（1字节）
	version, err = reader.ReadByte()
	if err != nil || version > 1 {
		return nil, fmt.Errorf("不支持的响应版本: %v", version)
	}
	// 读取响应体长度（4字节）
	bodyLen, err := readUint32(reader)
	if err != nil {
		return nil, fmt.Errorf("读取响应体长度失败: %v", err)
	}
	// 读取响应体
	bodyData, err := readBytes(reader, int(bodyLen))
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %v", err)
	}

	// 反序列化响应（假设 Python 服务返回 JSON）
	var response map[string]interface{}
	if err := json.Unmarshal(bodyData, &response); err != nil {
		return nil, fmt.Errorf("响应反序列化失败: %v", err)
	}
	// 新增：打印 Go 接收到的 Python 响应（关键日志）

	if response["error"] != nil {
		return nil, fmt.Errorf("Python 服务错误: %v", response["error"])
	}

	return response["result"], nil
}

// 辅助函数：按大端序读取 2 字节为 uint16
func readUint16(reader *bufio.Reader) (uint16, error) {
	b, err := reader.Peek(2)
	if err != nil {
		return 0, err
	}
	val := binary.BigEndian.Uint16(b)
	// 消费已读取的字节
	reader.Discard(2)
	return val, nil
}

// 辅助函数：按大端序读取 4 字节为 uint32
func readUint32(reader *bufio.Reader) (uint32, error) {
	b, err := reader.Peek(4)
	if err != nil {
		return 0, err
	}
	val := binary.BigEndian.Uint32(b)
	reader.Discard(4)
	return val, nil
}

// 辅助函数：读取指定长度的字节
func readBytes(reader *bufio.Reader, length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := io.ReadFull(reader, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
