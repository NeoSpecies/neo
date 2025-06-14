package loader

import (
	"go-ipc/config"
	"os"
)

const envPrefix = "IPC_"

func LoadFromEnv(cfg *config.GlobalConfig) error {
	// 发现配置
	// 删除以下行或注释掉
	// if endpoints := os.Getenv("DISCOVERY_ETCD_ENDPOINTS"); endpoints != "" {
	//     cfg.Discovery.ETCDEndpoints = strings.Split(endpoints, ",")
	// }
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
