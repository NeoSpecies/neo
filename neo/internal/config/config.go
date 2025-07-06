package config

import (
	"io/ioutil"
	"neo/internal/types"
	"time"
	"log"
	"gopkg.in/yaml.v2"
)

// 修改默认配置引用
var defaultConfig = &types.GlobalConfig{
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
	// 新增HTTP默认配置
	HTTP: types.HTTPConfig{
		Host:        "0.0.0.0",
		Port:        8000,
		EnableHTTPS: false,
	},
}

// GetGlobalConfig 获取全局配置
// 修改函数返回类型
func GetGlobalConfig() *types.GlobalConfig {
	// 尝试从配置文件加载
	configPath := "/www/neo/neo/configs/default.yml"
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Printf("警告: 无法读取配置文件 %s, 使用默认配置: %v", configPath, err)
		return defaultConfig
	}

	// 合并配置文件到默认配置
	if err := yaml.Unmarshal(data, defaultConfig); err != nil {
		log.Printf("警告: 解析配置文件失败, 使用默认配置: %v", err)
		return defaultConfig
	}

	return defaultConfig
}
