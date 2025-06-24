package config

import (
	"sync"
	"sync/atomic"
)

var globalConfig atomic.Value
var configLock sync.RWMutex

// GlobalConfig 聚合所有模块配置
type GlobalConfig struct {
	Discovery DiscoveryConfig
	Protocol  ProtocolConfig
	Metrics   MetricsConfig
	Pool      PoolConfig
	IPC       IPCConfig  // 新增IPC专用配置
	HTTP      HTTPConfig // 新增HTTP专用配置
}

// DiscoveryConfig 服务发现配置
type DiscoveryConfig struct {
	Storage         string `yaml:"storage"`
	TTL             int    `yaml:"ttl"`
	RefreshInterval int    `yaml:"refresh_interval"`
	ServicePrefix   string `yaml:"service_prefix"`
	ServiceKey      string `yaml:"service_key"`
	FilePath        string `yaml:"file_path"`
	MulticastAddr   string `yaml:"multicast_addr"`
}

// ProtocolConfig 协议配置
type ProtocolConfig struct {
	CompressionType string `yaml:"compression_type"`
	MaxMessageSize  int    `yaml:"max_message_size"`
	ChecksumEnabled bool   `yaml:"checksum_enabled"`
}

// 添加缺失的结构体定义
type MetricsConfig struct {
	EnablePrometheus bool `yaml:"enable_prometheus"`
	PrometheusPort   int  `yaml:"prometheus_port"`
}

// PoolConfig 连接池配置
type PoolConfig struct {
	InitialSize int `yaml:"initial_size"` // 新增初始连接数字段
	MinSize     int `yaml:"min_size"`
	MaxSize     int `yaml:"max_size"`
	IdleTimeout int `yaml:"idle_timeout"`
	KeepAliveInterval int `yaml:"keep_alive_interval"` // 新增保持连接间隔字段
}

// 合并后的包初始化函数，设置默认配置
func init() {
	globalConfig.Store(GlobalConfig{
		Pool: PoolConfig{
			InitialSize: 6,  // 初始连接数默认值
			MinSize:     5,  // 最小空闲连接数
			MaxSize:     20, // 最大连接数
			IdleTimeout: 60, // 空闲连接超时时间(秒)
		},
		IPC: IPCConfig{
			Host:           "127.0.0.1",
			Port:           9090,
			MaxConnections: 1000,
		},
	})
}

// 原子操作接口
func Get() GlobalConfig {
	return globalConfig.Load().(GlobalConfig)
}

func Update(newConfig GlobalConfig) {
	configLock.Lock()
	defer configLock.Unlock()
	globalConfig.Store(newConfig)
}

// 线程安全的配置获取方法示例
func GetDiscoveryConfig() DiscoveryConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return Get().Discovery
}

// IPC配置
type IPCConfig struct {
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	MaxConnections int    `yaml:"max_connections"`
}

// HTTP配置
type HTTPConfig struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	EnableHTTPS bool   `yaml:"enable_https"`
	CertFile    string `yaml:"cert_file"`
	KeyFile     string `yaml:"key_file"`
}
