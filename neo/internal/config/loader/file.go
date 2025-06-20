package loader

import (
	"fmt"
	"os"
	"neo/internal/config"
	"gopkg.in/yaml.v3"
)

func LoadFromFile(path string, cfg *config.GlobalConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("读取文件 %s 时出错: %v\n", path, err)
		return err
	}

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		fmt.Printf("解析 YAML 文件 %s 时出错: %v\n", path, err)
		return err
	}

	return nil
}
