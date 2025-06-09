package loader

import (
	"go-ipc/config"
	"os"
	"strings"
)

const envPrefix = "IPC_"

func LoadFromEnv(cfg *config.GlobalConfig) error {
	// 发现配置
	cfg.Discovery.ETCDEndpoints = strings.Split(
		getEnv(envPrefix+"ETCD_ENDPOINTS", "localhost:2379"), ",")

	// 协议配置
	cfg.Protocol.CompressionType = getEnv(
		envPrefix+"COMPRESSION_TYPE", "gzip")

	// 其他配置加载...
	return nil
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
