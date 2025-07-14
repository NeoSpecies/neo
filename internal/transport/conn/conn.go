package conn

import (
	"context"
	"net"
)

// Conn 定义连接接口
type Conn interface {
	Send(ctx context.Context, msg []byte) error
	Receive(ctx context.Context) ([]byte, error)
	Close() error
	Validate() error // 检查连接是否有效
}

// tcpConn TCP连接具体实现
type tcpConn struct {
	conn net.Conn
}

// newTCPConn 创建新TCP连接（带错误处理）
func newTCPConn(addr string) (*tcpConn, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err // 返回错误而非忽略
	}
	return &tcpConn{conn: conn}, nil
}

// Send 发送消息
func (c *tcpConn) Send(ctx context.Context, msg []byte) error {
	_, err := c.conn.Write(msg)
	return err
}

// Receive 接收消息
func (c *tcpConn) Receive(ctx context.Context) ([]byte, error) {
	buf := make([]byte, 1024)
	n, err := c.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Close 关闭连接
func (c *tcpConn) Close() error {
	return c.conn.Close()
}

// Validate 检查连接有效性
func (c *tcpConn) Validate() error {
	// 示例：通过写1字节测试连接
	_, err := c.conn.Write([]byte{0})
	return err
}
