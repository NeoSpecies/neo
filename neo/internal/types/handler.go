package types

import "net"

// TCPHandler 连接处理器接口
type TCPHandler interface {
	Start() error
	Stop()
}

// TCPHandlerFactory 创建处理器的工厂函数类型
type TCPHandlerFactory func(conn net.Conn) TCPHandler
