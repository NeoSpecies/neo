package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

func main() {
	// 注册 Go 测试函数（供 Python 调用）
	RegisterService("go.service.test", func(params map[string]interface{}) (interface{}, error) {
		fmt.Printf("Go 服务接收到参数：%+v\n", params)
		return fmt.Sprintf("Go 测试函数返回：%v", params["input"]), nil
	})

	go StartIpcServer(":9090")

	http.HandleFunc("/trigger", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("接收到 HTTP 请求：%s %s\n", r.Method, r.URL.Path)

		var req struct{ Input string }
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			fmt.Printf("解码请求体失败：%v\n", err)
			http.Error(w, "无效的请求体", 400)
			return
		}

		pythonResult, err := callPythonService("python.service.demo", map[string]interface{}{"input": req.Input})
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// 关键修改：将 Python 返回的字符串解析为 JSON 对象（避免二次转义）
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(pythonResult.(string)), &result); err != nil {
			http.Error(w, "Python 响应格式错误", 500)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": "HTTP 请求处理完成",
			"result":  result, // 直接返回解析后的对象
		})
	})

	fmt.Printf("HTTP Server listening on :8080\n")
	http.ListenAndServe(":8080", nil)
}

func callPythonService(method string, params map[string]interface{}) (interface{}, error) {
	conn, err := net.Dial("tcp", "127.0.0.1:9091")
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	paramData, _ := json.Marshal(params)
	conn.Write([]byte(method + "|" + string(paramData)))

	resp, err := io.ReadAll(conn)
	if err != nil {
		return nil, err
	}
	return string(resp), nil
}
