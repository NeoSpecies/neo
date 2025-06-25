package config

import (
	"time"
	"neo/internal/types"
)

// GlobalConfig 全局配置结构体
type GlobalConfig struct {
	IPC      types.IPCConfig      `yaml:"ipc"`
	Protocol types.ProtocolConfig `yaml:"protocol"`
	Metrics  types.MetricsConfig  `yaml:"metrics"`
}

// 默认配置实例
var defaultConfig = &GlobalConfig{
	IPC: types.IPCConfig{
		Host:              "127.0.0.1",
		Port:              9090,
		MaxConnections:    1000,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      10 * time.Second,
		ConnectionTimeout: 60 * time.Second,
		WorkerCount:       10,
	},
	Protocol: types.ProtocolConfig{
		CompressionType: "none",
		MaxMessageSize:  4194304,
		ChecksumEnabled: true,
	},
	Metrics: types.MetricsConfig{
		Enabled:           false,
		PrometheusAddress: ":9091",
	},
}

// GetGlobalConfig 获取全局配置
func GetGlobalConfig() *GlobalConfig {
	// 实际项目中应从配置文件加载，此处返回默认配置
	return defaultConfig
}
