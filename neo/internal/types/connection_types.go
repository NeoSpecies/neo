package types

import (
	"errors"
	"net"
	"sync"
	"time"
)

// 连接池错误定义
var (
	// ErrPoolClosed 连接池已关闭
	ErrPoolClosed = errors.New("connection pool is closed")
	// ErrPoolExhausted 连接池耗尽
	ErrPoolExhausted = errors.New("connection pool exhausted")
	// ErrConnectionUnhealthy 连接不健康
	ErrConnectionUnhealthy = errors.New("connection is unhealthy")
)

// Config 统一连接池配置
type Config struct {
	// 基础配置
	MaxSize           int           // 最大连接数
	MinSize           int           // 最小连接数
	ConnectTimeout    time.Duration // 连接超时时间
	IdleTimeout       time.Duration // 空闲超时时间
	KeepAliveInterval time.Duration // 保持连接间隔

	// 扩展配置
	InitialSize         int                 // 初始连接数
	AutoScaling         bool                // 是否启用自动扩缩容
	ScaleUpThreshold    float64             // 扩容阈值（活跃连接比例）
	ScaleDownThreshold  float64             // 缩容阈值（空闲连接比例）
	ScaleStep           int                 // 每次扩缩容步长
	HealthCheck         bool                // 是否启用健康检查
	HealthCheckInterval time.Duration       // 健康检查间隔
	MaxErrorCount       int                 // 最大错误次数
	MaxLatency          time.Duration       // 最大延迟阈值
	LoadBalance         LoadBalanceStrategy // 负载均衡策略
	MaxRetryCount       int                 // 连接创建最大重试次数
	RetryInterval       time.Duration       // 连接重试间隔
}

// ScalingConfig 自动扩缩容配置
type ScalingConfig struct {
	MinSize   int // 最小连接数
	MaxSize   int // 最大连接数
	ExpandPct int // 扩容阈值百分比
	ShrinkPct int // 缩容阈值百分比
}

// Connection 统一连接结构体
type Connection struct {
	Conn       net.Conn           // 底层连接
	Pool       *TCPConnectionPool // 所属连接池
	Stats      *ConnectionStats   // 连接统计
	LastUsed   time.Time          // 最后使用时间
	LastCheck  time.Time          // 最后检查时间
	InUse      bool               // 是否正在使用
	ErrorCount int                // 错误次数
	Closed     bool               // 是否已关闭
}

// Balancer 负载均衡器接口
type Balancer interface {
	// Select 从连接列表中选择一个合适的连接
	Select(connections []*Connection) (*Connection, error)
}

// TCPConnectionPool 连接池结构体
// 修改TCPConnectionPool结构体中的Metrics字段引用
type TCPConnectionPool struct {
	MaxSize           int                      `json:"max_size"`
	MinSize           int                      `json:"min_size"`
	InitialSize       int                      `json:"initial_size"`
	IdleTimeout       time.Duration            `json:"idle_timeout"`
	KeepAliveInterval time.Duration            `json:"keep_alive_interval"`
	Config            Config                   `json:"config"` // 修改为Config类型
	Factory           func() (net.Conn, error) `json:"-"`
	Balancer          Balancer                 `json:"balancer"`
	Metrics           *Metrics      `json:"metrics"` // 确保Metrics引用的是types.Metrics
	Done              chan struct{}            `json:"-"`
	WaitConn          chan struct{}            `json:"-"`
	Connections       []*Connection            `json:"connections"`
	Mu                *sync.RWMutex            `json:"-"` // 修改为RWMutex
	Closed            bool                     `json:"closed"`
}

// 负载均衡策略常量
type LoadBalanceStrategy string

const (
	LoadBalanceRoundRobin       LoadBalanceStrategy = "round_robin"
	LoadBalanceLeastConnections LoadBalanceStrategy = "least_connections"
	LoadBalanceSourceIP         LoadBalanceStrategy = "source_ip"
)

// ConnectionPoolMetrics 连接池指标
type ConnectionPoolMetrics struct {
	ActiveConnections  int64 // 活跃连接数
	TotalConnections   int64 // 总连接数
	WaitingRequests    int64 // 等待请求数
	ConnectionErrors   int64 // 连接错误数
	ConnectionTimeouts int64 // 连接超时数
}

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	CreatedAt    time.Time // 创建时间
	LastUsed     time.Time // 最后使用时间
	UsageCount   int64     // 使用次数
	ReadBytes    int64     // 读取字节数
	WrittenBytes int64     // 写入字节数
	ErrorCount   int64     // 错误次数
	LastError    error     // 最后错误
}

// NewConnectionStats 创建新的连接统计信息
func NewConnectionStats() *ConnectionStats {
	return &ConnectionStats{
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}
}
