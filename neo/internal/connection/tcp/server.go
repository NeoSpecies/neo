package tcp

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"neo/internal/connection"
	"neo/internal/types"
)

// 类型别名迁移（保留兼容）
// Deprecated: 请使用 types.ServerConfig
type ServerConfig = types.ServerConfig

// Deprecated: 请使用 types.TCPServer
type TCPServer = types.TCPServer

// NewServer 创建新的TCP服务器
func NewServer(config *types.TCPConfig, callback types.MessageCallback) (*types.TCPServer, error) {
	// 初始化连接池 - 使用正确的包和函数
	pool, err := connection.NewTCPConnectionPool(types.Config{
		MaxSize:           config.MaxConnections,
		ConnectTimeout:    config.ConnectionTimeout,
		IdleTimeout:       config.ReadTimeout,
		KeepAliveInterval: 30 * time.Second,
	}, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("创建连接池失败: %v", err)
	}

	// 初始化指标收集器
	metrics := types.NewMetrics(prometheus.NewRegistry())

	// 使用types包的构造函数创建TCPServer
	return types.NewTCPServer(config, callback, metrics, pool), nil
}

// TCPHandler 处理单个TCP连接
type TCPHandler struct {
	Conn         net.Conn
	callback     types.MessageCallback
	metrics      *types.Metrics
	maxMsgSize   int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// NewTCPHandler 创建新的TCP连接处理器
func NewTCPHandler(
	conn net.Conn,
	callback types.MessageCallback,
	metrics *types.Metrics,
	maxMsgSize int,
	readTimeout, writeTimeout time.Duration,
) *TCPHandler {
	return &TCPHandler{
		Conn:         conn,
		callback:     callback,
		metrics:      metrics,
		maxMsgSize:   maxMsgSize,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}

// Start 开始处理连接
func (h *TCPHandler) Start() {
	codec := NewCodec(h.Conn, h.Conn)

	for {
		// 设置读取超时
		h.Conn.SetReadDeadline(time.Now().Add(h.readTimeout))

		// 使用codec读取并解析IPC消息
		msg, err := codec.ReadIPCMessage()
		if err != nil {
			if connErr, ok := err.(*types.ConnectionError); ok {
				if connErr.Type == types.ErrorTypeConnectionClosed {
					fmt.Printf("[INFO] 客户端断开连接")
				} else {
					fmt.Printf("[ERROR] 未知错误: %v\n", err)
				}
				return
			}

			// 将解析后的消息帧转换为JSON字节
			requestData, err := json.Marshal(msg)
			if err != nil {
				fmt.Printf("[ERROR] 消息序列化失败: %v\n", err)
				return
			}

			// 调用回调处理解析后的消息
			// 将err重命名为callbackErr以避免遮蔽上层作用域的err变量
			if _, callbackErr := h.callback(requestData); callbackErr != nil {
				fmt.Printf("[ERROR] 消息处理失败: %v\n", callbackErr)
				// 返回结构化错误响应
				errorResp := map[string]interface{}{
					"error": map[string]string{
						"code":    "PROCESSING_ERROR",
						"message": callbackErr.Error(),
					},
				}
				// 使用不同变量名避免遮蔽
				errorResponse, marshalErr := json.Marshal(errorResp)
				if marshalErr != nil {
					fmt.Printf("[ERROR] 错误响应序列化失败: %v\n", marshalErr)
					return
				}
				// 发送错误响应
				responseFrame := &types.MessageFrame{
					Type:    string(types.MessageTypeResponse),
					Payload: errorResponse,
				}
				if writeErr := codec.WriteIPCMessage(responseFrame); writeErr != nil {
					fmt.Printf("[ERROR] 发送错误响应失败: %v\n", writeErr)
					return
				}
				continue
			}

			// 设置写入超时
			h.Conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))

			// 解析请求消息Payload
			var payloadMap map[string]interface{}
			if unmarshalErr := json.Unmarshal(msg.Payload, &payloadMap); unmarshalErr != nil {
				fmt.Printf("[ERROR] 解析Payload失败: %v\n", unmarshalErr)
				return
			}

			// 提取服务信息并添加错误处理
			serviceID, ok := payloadMap["service_id"].(string)
			if !ok {
				fmt.Printf("[ERROR] 无效的service_id格式\n")
				return
			}

			name, ok := payloadMap["name"].(string)
			if !ok {
				fmt.Printf("[ERROR] 无效的name格式\n")
				return
			}

			address, ok := payloadMap["address"].(string)
			if !ok {
				fmt.Printf("[ERROR] 无效的address格式\n")
				return
			}

			portVal, ok := payloadMap["port"].(float64)
			if !ok {
				fmt.Printf("[ERROR] 无效的port格式\n")
				return
			}
			port := int(portVal)

			// 构建响应数据
			responseData := map[string]interface{}{
				"result": map[string]interface{}{
					"id":      serviceID,
					"name":    name,
					"address": address,
					"port":    port,
				},
			}

			// 序列化正常响应
			normalResponse, err := json.Marshal(responseData)
			if err != nil {
				fmt.Printf("[ERROR] 响应序列化失败: %v\n", err)
				return
			}

			// 构建并发送响应帧
			responseFrame := &types.MessageFrame{
				Type:    string(types.MessageTypeResponse),
				Payload: normalResponse,
			}
			if err := codec.WriteIPCMessage(responseFrame); err != nil {
				fmt.Printf("[ERROR] 发送响应失败: %v\n", err)
				return
			}
		}
	}
}
