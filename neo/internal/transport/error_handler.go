package transport

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/google/uuid"
	"neo/internal/ipcprotocol"
)

// ErrorHandler 统一错误响应处理
type ErrorHandler struct{}

// Send 发送标准化错误响应
// 参数:
//   conn - 网络连接
//   errorCode - 错误代码
//   message - 错误描述信息
// 返回:
//   发送过程中的错误，如果有的话
func (e *ErrorHandler) Send(conn net.Conn, errorCode int, message string) error {
	// 创建错误响应数据结构
	errorResponse := map[string]interface{}{
		"code":    errorCode,
		"message": message,
	}

	// 序列化为JSON
	jsonData, err := json.Marshal(errorResponse)
	if err != nil {
		return fmt.Errorf("序列化错误响应失败: %v", err)
	}

	// 生成唯一消息ID（遵循系统中server.go的实现方式）
	msgID := uuid.New().String()

	// 使用ipcprotocol构造错误消息
	// 注意：NewMessage需要三个参数：消息类型、消息ID、负载数据
	msg := ipcprotocol.NewMessage(ipcprotocol.TypeError, msgID, jsonData)

	// 使用Bytes()方法获取序列化后的字节（而非Serialize()）
	if _, err := conn.Write(msg.Bytes()); err != nil {
		return fmt.Errorf("发送错误响应失败: %v", err)
	}

	return nil
}
