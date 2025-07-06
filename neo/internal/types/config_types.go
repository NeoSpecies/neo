/*
 * 描述: 定义系统各模块的配置结构体，包括全局配置、IPC配置、协议配置、指标配置和连接池配置
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import "time"

// GlobalConfig 全局配置结构
// 包含系统所有核心模块的配置信息
// +----------------+----------------------+-------------------------------+
// | 字段名         | 类型                 | 描述                          |
// +----------------+----------------------+-------------------------------+
// | IPC            | IPCConfig            | IPC服务器配置                 |
// | Protocol       | ProtocolConfig       | 协议配置                      |
// | Metrics        | MetricsConfig        | 指标收集配置                  |
// | Pool           | PoolConfig           | 连接池配置                    |
// +----------------+----------------------+-------------------------------+
type GlobalConfig struct {
	IPC      IPCConfig      `yaml:"ipc"`
	Protocol ProtocolConfig `yaml:"protocol"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Pool     PoolConfig     `yaml:"pool"`
}

// IPCConfig IPC服务器配置
// 定义IPC服务器的网络参数和资源限制
// +---------------------+-------------------+-----------------------------------+
// | 字段名              | 类型              | 描述                              |
// +---------------------+-------------------+-----------------------------------+
// | Host                | string            | 服务器绑定主机地址                |
// | Port                | int               | 服务器监听端口号                  |
// | MaxConnections      | int               | 最大并发连接数限制                |
// | ReadTimeout         | time.Duration     | 读取操作超时时间                  |
// | WriteTimeout        | time.Duration     | 写入操作超时时间                  |
// | ConnectionTimeout   | time.Duration     | 连接建立超时时间                  |
// | WorkerCount         | int               | 处理请求的工作协程数量            |
// +---------------------+-------------------+-----------------------------------+
type IPCConfig struct {
	Host              string        `yaml:"host"`
	Port              int           `yaml:"port"`
	MaxConnections    int           `yaml:"max_connections"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	ConnectionTimeout time.Duration `yaml:"connection_timeout"`
	WorkerCount       int           `yaml:"worker_count"`
}

// ProtocolConfig 协议配置
// 定义通信协议的相关参数
// +---------------------+-------------------+-----------------------------------+
// | 字段名              | 类型              | 描述                              |
// +---------------------+-------------------+-----------------------------------+
// | CompressionType     | string            | 压缩算法类型(Gzip/Zstd/LZ4)       |
// | MaxMessageSize      | int               | 最大消息大小限制(字节)            |
// | ChecksumEnabled     | bool              | 是否启用校验和验证                |
// | ReadTimeout         | time.Duration     | 协议层读取超时时间                |
// | WriteTimeout        | time.Duration     | 协议层写入超时时间                |
// +---------------------+-------------------+-----------------------------------+
type ProtocolConfig struct {
	CompressionType string        `yaml:"compression_type"`
	MaxMessageSize  int           `yaml:"max_message_size"`
	ChecksumEnabled bool          `yaml:"checksum_enabled"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
}

// MetricsConfig 指标配置
// 定义性能指标收集和暴露的相关参数
// +---------------------+-------------------+-----------------------------------+
// | 字段名              | 类型              | 描述                              |
// +---------------------+-------------------+-----------------------------------+
// | Enabled             | bool              | 是否启用指标收集                  |
// | PrometheusAddress   | string            | Prometheus暴露地址(如:localhost:9090) |
// | CollectionInterval  | time.Duration     | 指标收集间隔时间                  |
// +---------------------+-------------------+-----------------------------------+
type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled"`
	PrometheusAddress  string        `yaml:"prometheus_address"`
	CollectionInterval time.Duration `yaml:"collection_interval"`
}

// PoolConfig 连接池配置
// 定义连接池的大小、超时和健康检查参数
// +---------------------+-------------------+-----------------------------------+
// | 字段名              | 类型              | 描述                              |
// +---------------------+-------------------+-----------------------------------+
// | MinSize             | int               | 最小保持连接数                    |
// | MaxSize             | int               | 最大连接数限制                    |
// | IdleTimeout         | time.Duration     | 空闲连接超时时间                  |
// | MaxLifetime         | time.Duration     | 连接最大生存时间                  |
// | HealthCheckInterval | time.Duration     | 健康检查执行间隔                  |
// +---------------------+-------------------+-----------------------------------+
type PoolConfig struct {
	MinSize             int           `yaml:"min_size"`
	MaxSize             int           `yaml:"max_size"`
	IdleTimeout         time.Duration `yaml:"idle_timeout"`
	MaxLifetime         time.Duration `yaml:"max_lifetime"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}
