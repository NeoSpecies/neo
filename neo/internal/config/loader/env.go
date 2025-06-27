package loader

import (
	"neo/internal/types"
	"os"
)

const envPrefix = "NEO_IPC_"

func LoadFromEnv(cfg *types.GlobalConfig) error {
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
