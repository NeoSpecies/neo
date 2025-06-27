package types

import "time"

// GlobalConfig 全局配置结构
type GlobalConfig struct {
	IPC      IPCConfig      `yaml:"ipc"`
	Protocol ProtocolConfig `yaml:"protocol"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Pool     PoolConfig     `yaml:"pool"`
}

// IPCConfig IPC服务器配置
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
type ProtocolConfig struct {
	CompressionType string        `yaml:"compression_type"`
	MaxMessageSize  int           `yaml:"max_message_size"`
	ChecksumEnabled bool          `yaml:"checksum_enabled"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled            bool          `yaml:"enabled"`
	PrometheusAddress  string        `yaml:"prometheus_address"`
	CollectionInterval time.Duration `yaml:"collection_interval"`
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MinSize             int           `yaml:"min_size"`
	MaxSize             int           `yaml:"max_size"`
	IdleTimeout         time.Duration `yaml:"idle_timeout"`
	MaxLifetime         time.Duration `yaml:"max_lifetime"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
}
