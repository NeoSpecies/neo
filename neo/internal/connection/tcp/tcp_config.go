package tcp

import (
	"net"
	"strconv"
	"time"

	"neo/internal/config"
	"neo/internal/types"
)

// TCPConfig 本地包装结构体，用于实现ServerConfig接口
// 原问题：直接为types.TCPConfig添加方法导致跨包方法定义错误
type TCPConfig struct {
	types.TCPConfig // 嵌入原有配置结构体
}

// GetAddress 实现types.ServerConfig接口
func (c *TCPConfig) GetAddress() string {
	globalConfig := config.GetGlobalConfig()
	return net.JoinHostPort(globalConfig.IPC.Host, strconv.Itoa(globalConfig.IPC.Port))
}

// GetMaxConnections 实现types.ServerConfig接口
func (c *TCPConfig) GetMaxConnections() int {
	return c.MaxConnections
}

// GetConnectionTimeout 实现types.ServerConfig接口
func (c *TCPConfig) GetConnectionTimeout() time.Duration {
	return c.ConnectionTimeout
}

// GetHandlerConfig 实现types.ServerConfig接口
func (c *TCPConfig) GetHandlerConfig() interface{} {
	return nil
}
