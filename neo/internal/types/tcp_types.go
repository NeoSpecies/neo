package types

import (
	"context"
	"net"
	"sync"
	"time"
)

// TCPConfig TCP服务器配置结构体
// 仅包含字段定义，不包含方法实现（符合设计思路）
type TCPConfig struct {
	MaxConnections    int           `yaml:"max_connections"`
	MaxMsgSize        int           `yaml:"max_message_size"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	WorkerCount       int           `yaml:"worker_count"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
    Address           string        `yaml:"address"` // 添加Address字段
}

// 实现ServerConfig接口方法
func (c *TCPConfig) GetAddress() string {
    return c.Address
}

func (c *TCPConfig) GetMaxConnections() int {
    return c.MaxConnections
}

func (c *TCPConfig) GetConnectionTimeout() time.Duration {
    return c.ConnectionTimeout
}

func (c *TCPConfig) GetHandlerConfig() interface{} {
    return c
}

// MessageCallback 消息处理回调函数类型
type MessageCallback func([]byte) ([]byte, error)

// TCPServer TCP服务器结构体
type TCPServer struct {
	listener    net.Listener
	config      *TCPConfig
	metrics     *Metrics
	connections *TCPConnectionPool
	callback    MessageCallback
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
	taskChan    chan func()
}
