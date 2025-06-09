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
	Server    ServerConfig
}

// DiscoveryConfig 服务发现配置
type DiscoveryConfig struct {
	ETCDEndpoints   []string `config:"etcd_endpoints"`
	ServicePrefix   string   `config:"service_prefix"`
	RefreshInterval string   `config:"refresh_interval"`
}

// ProtocolConfig 协议配置
type ProtocolConfig struct {
	CompressionType string `config:"compression_type"`
	MaxMessageSize  int    `config:"max_message_size"`
	ChecksumEnabled bool   `config:"checksum_enabled"`
}

// 其他模块配置结构...

// 添加缺失的结构体定义
type MetricsConfig struct {
	EnablePrometheus bool `config:"enable_prometheus"`
	PrometheusPort   int  `config:"prometheus_port"`
}

type PoolConfig struct {
	MinSize     int `config:"min_size"`
	MaxSize     int `config:"max_size"`
	IdleTimeout int `config:"idle_timeout"`
}

type ServerConfig struct {
	Host string `config:"host"`
	Port int    `config:"port"`
}

// 新增原子操作接口
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
