package main

import (
	"bufio"
	"bytes"

	// "encoding/base64" 移除未使用的base64导入
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"os"
	"time"

	_ "net/http/pprof" // 匿名导入pprof，它会自动注册handler到默认的http.ServeMux

	"github.com/google/uuid"
)

// 全局连接池
var connPool *ConnPool

// 初始化连接池
func init() {
	connPool = NewConnPool(100, 30*time.Second) // 最大100个连接，30秒超时
}

// 注册 Go 测试函数（供 Python 调用）
func init() {
	RegisterService("go.service.test", func(params map[string]interface{}) (interface{}, error) {
		// fmt.Printf("Go 服务接收到参数：%+v\n", params)
		return fmt.Sprintf("Go 测试函数返回：%v", params["input"]), nil
	})
}

func main() {
	// 启动 IPC 服务
	go func() {
		fmt.Println("正在启动 IPC 服务...")
		if err := StartIpcServer("127.0.0.1:9090"); err != nil {
			fmt.Printf("IPC 服务启动失败: %v\n", err)
			os.Exit(1) // 如果 IPC 服务启动失败，整个程序退出
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

	fmt.Println("HTTP 服务启动，监听端口 8000")
	http.ListenAndServe("0.0.0.0:80", nil)
}

// 移除 handleFileUpload 函数完整定义
// func handleFileUpload(w http.ResponseWriter, r *http.Request) {
// 	// 原第96行：首次声明 err
// 	file, header, err := r.FormFile("file")
// 	if err != nil {
// 		http.Error(w, "文件解析失败: "+err.Error(), http.StatusBadRequest)
// 		return
// 	}
// 	defer file.Close()
//
// 	// 创建临时文件
// 	tempFile, err := os.CreateTemp("", "upload-*") // 复用外部 err（非重复声明）
// 	if err != nil {
// 		http.Error(w, "创建临时文件失败: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	defer tempFile.Close()
// 	defer os.Remove(tempFile.Name())
//
// 	// 原第114行：改为赋值操作（使用 = 而非 :=）
// 	_, err = io.Copy(tempFile, file) // 复用外部 err，避免影子声明
// 	if err != nil {
// 		http.Error(w, "保存文件失败: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
//
// 	// 重置文件指针到开始位置
// 	_, err = tempFile.Seek(0, 0) // 复用外部 err
// 	if err != nil {
// 		http.Error(w, "重置文件指针失败: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
//
// 	// 读取临时文件内容
// 	fileBytes, err := io.ReadAll(tempFile) // 复用外部 err（首次声明 fileBytes，允许使用 :=）
// 	if err != nil {
// 		http.Error(w, "读取临时文件失败: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
//
// 	// 将文件内容进行 Base64 编码
// 	encodedContent := base64.StdEncoding.EncodeToString(fileBytes)
//
// 	// 构造 IPC 请求参数
// 	ipcParams := map[string]interface{}{
// 		"files": []map[string]interface{}{
// 			{
// 				"meta": map[string]interface{}{
// 					"original_name": header.Filename,
// 					"mimetype":      header.Header.Get("Content-Type"),
// 				},
// 				"content": encodedContent, // 传递 Base64 编码后的字符串
// 			},
// 		},
// 	}
//
// 	// 通过 IPC 调用 Python 服务处理文件
// 	pythonResult, err := callPythonIpcService("python.service.fileProcess", ipcParams)
// 	if err != nil {
// 		http.Error(w, "IPC 调用失败: "+err.Error(), http.StatusInternalServerError)
// 		return
// 	}
//
// 	if pythonResult == nil {
// 		http.Error(w, "Python 服务返回空结果", http.StatusInternalServerError)
// 		return
// 	}
//
// 	resultMap, ok := pythonResult.(map[string]interface{})
// 	if !ok {
// 		http.Error(w, "Python 响应类型错误，非预期的 map 类型", http.StatusInternalServerError)
// 		return
// 	}
//
// 	processedFileVal := resultMap["processed_file"]
// 	if processedFileVal == nil {
// 		http.Error(w, "Python 响应中未包含 processed_file", http.StatusInternalServerError)
// 		return
// 	}
//
// 	processedFile, ok := processedFileVal.(map[string]interface{})
// 	if !ok {
// 		http.Error(w, "Python 响应中 processed_file 非预期的 map 类型", http.StatusInternalServerError)
// 		return
// 	}
//
// 	contentStr, ok := processedFile["content"].(string)
// 	if !ok {
// 		http.Error(w, "Python 响应中 content 非预期的字符串类型", http.StatusInternalServerError)
// 		return
// 	}
//
// 	fileContent, err := base64.StdEncoding.DecodeString(contentStr)
// 	if err != nil {
// 		http.Error(w, fmt.Sprintf("Base64 解码失败: %v", err), http.StatusInternalServerError)
// 		return
// 	}
//
// 	mimetype, ok := processedFile["mimetype"].(string)
// 	if !ok {
// 		http.Error(w, "Python 响应中 mimetype 非预期的字符串", http.StatusInternalServerError)
// 		return
// 	}
// 	newName, ok := processedFile["new_name"].(string)
// 	if !ok {
// 		http.Error(w, "Python 响应中 new_name 非预期的字符串", http.StatusInternalServerError)
// 		return
// 	}
//
// 	w.Header().Set("Content-Type", mimetype)
// 	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", newName))
// 	w.Write(fileContent)
// }

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
