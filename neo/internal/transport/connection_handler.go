package transport

import (
	"net"
)

// ConnectionHandler 专注于连接生命周期管理
type ConnectionHandler struct {
	protocol     ProtocolHandler
	registry     *ServiceRegistry
	errorHandler *ErrorHandler
}

// 处理连接的完整生命周期
func (h *ConnectionHandler) Handle(conn net.Conn) {
	// 原handleConnection函数逻辑迁移至此
}
