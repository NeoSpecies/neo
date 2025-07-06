package tcp

import (
	"context"
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
// 修复NewServer函数参数传递
func NewServer(config *types.TCPConfig, callback types.MessageCallback) (*types.TCPServer, error) {
	// 初始化连接池
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

	// 创建上下文
	ctx := context.Background()

	// 创建TCP处理器工厂
	factory := func(conn net.Conn) types.TCPHandler {
		return NewTCPHandler(
			conn,
			callback,
			metrics,
			config.MaxMsgSize,
			config.ReadTimeout,
			config.WriteTimeout,
		)
	}

	// 使用正确参数调用NewTCPServer
	return types.NewTCPServer(config, callback, metrics, pool, ctx, factory), nil
}

// TCPHandler 处理单个TCP连接
// TCPHandler 现在实现types.TCPHandler接口
type TCPHandler struct {
	conn         net.Conn
	callback     types.MessageCallback
	metrics      *types.Metrics
	maxMsgSize   int
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// Stop 实现types.TCPHandler接口的Stop方法
// 修改Stop方法，添加上下文信息
func (h *TCPHandler) Stop() {
	if h.conn != nil {
		addr := h.conn.RemoteAddr().String()
		if err := h.conn.Close(); err != nil {
			fmt.Printf("[ERROR] 关闭连接失败 (客户端: %s): %v\n", addr, err)
		} else {
			fmt.Printf("[INFO] 连接已关闭 (客户端: %s)\n", addr)
		}
	}
}

// Start 实现types.TCPHandler接口的Start方法
// 修改Start方法，移除重复的响应发送逻辑
func (h *TCPHandler) Start() error {
	codec := NewCodec(h.conn, h.conn)

	for {
		// 设置读取超时
		h.conn.SetReadDeadline(time.Now().Add(h.readTimeout))

		// 使用codec读取并解析IPC消息
		msg, err := codec.ReadIPCMessage()
		if err != nil {
			if connErr, ok := err.(*types.ConnectionError); ok {
				if connErr.Type == types.ErrorTypeConnectionClosed {
					fmt.Printf("[INFO] 客户端断开连接: %s\n", h.conn.RemoteAddr())
					return nil // 正常关闭，不返回错误
				}
			}
			return fmt.Errorf("读取消息失败: %w", err)
		}

		// 记录原始消息数据
		fmt.Printf("[DEBUG] 原始消息数据: %+v\n", msg)
		fmt.Printf("[DEBUG] 收到客户端消息: Type=%s, Payload=%s\n", msg.Type, string(msg.Payload))

		// 将解析后的消息帧转换为JSON字节
		respData, callbackErr := h.callback(msg.Payload)
		// 新增响应数据详细日志
		fmt.Printf("[DEBUG] 回调返回的响应数据: 长度=%d, 内容=%s\n", len(respData), string(respData))
		fmt.Printf("[DEBUG] 回调返回的错误: %v\n", callbackErr)

		if callbackErr != nil {
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
				return marshalErr
			}
			// 发送错误响应
			responseFrame := &types.MessageFrame{
				Type:    string(types.MessageTypeResponse),
				Payload: errorResponse,
			}
			if writeErr := codec.WriteIPCMessage(responseFrame); writeErr != nil {
				fmt.Printf("[ERROR] 发送错误响应失败: %v\n", writeErr)
				return writeErr
			}
			continue
		} else if respData != nil {
			// 使用回调返回的响应数据
			responseFrame := &types.MessageFrame{
				Type:    string(types.MessageTypeResponse),
				Payload: respData,
			}
			if writeErr := codec.WriteIPCMessage(responseFrame); writeErr != nil {
				return writeErr
			}
			// 发送响应后退出循环，避免重复处理
			return nil
		}

		// 设置写入超时
		h.conn.SetWriteDeadline(time.Now().Add(h.writeTimeout))

		// 解析请求消息Payload
		var payloadMap map[string]interface{}
		if unmarshalErr := json.Unmarshal(msg.Payload, &payloadMap); unmarshalErr != nil {
			fmt.Printf("[ERROR] 解析Payload失败: %v\n", unmarshalErr)
			return unmarshalErr // 添加错误返回
		}

		// 提取服务信息并添加错误处理
		serviceID, ok := payloadMap["service_id"].(string)
		if !ok {
			fmt.Printf("[ERROR] 无效的service_id格式\n")
			return fmt.Errorf("无效的service_id格式") // 添加错误返回
		}

		name, ok := payloadMap["name"].(string)
		if !ok {
			fmt.Printf("[ERROR] 无效的name格式\n")
			return fmt.Errorf("无效的name格式") // 添加错误返回
		}

		address, ok := payloadMap["address"].(string)
		if !ok {
			fmt.Printf("[ERROR] 无效的address格式\n")
			return fmt.Errorf("无效的address格式") // 添加错误返回
		}

		portVal, ok := payloadMap["port"].(float64)
		if !ok {
			fmt.Printf("[ERROR] 无效的port格式\n")
			return fmt.Errorf("无效的port格式") // 添加错误返回
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
			return err // 添加错误返回
		}

		// 构建并发送响应帧
		responseFrame := &types.MessageFrame{
			Type:    string(types.MessageTypeResponse),
			Payload: normalResponse,
		}
		if err := codec.WriteIPCMessage(responseFrame); err != nil {
			fmt.Printf("[ERROR] 发送响应失败: %v\n", err)
			return err
		}
	}
}

// NewTCPHandler 创建新的TCP连接处理器
func NewTCPHandler(
	conn net.Conn,
	callback types.MessageCallback,
	metrics *types.Metrics,
	maxMsgSize int,
	readTimeout, writeTimeout time.Duration,
) *TCPHandler {
	// 新增：记录新客户端连接
	fmt.Printf("[INFO] 新客户端连接: %s -> %s\n", conn.RemoteAddr(), conn.LocalAddr())
	return &TCPHandler{
		conn:         conn,
		callback:     callback,
		metrics:      metrics,
		maxMsgSize:   maxMsgSize,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
	}
}
