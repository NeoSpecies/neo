package loader

import (
	"neo/internal/types"
	"os"

	"gopkg.in/yaml.v3"
)

// FileLoader 从文件加载配置
type FileLoader struct {
	path string
}

// NewFileLoader 创建新的文件配置加载器
func NewFileLoader(path string) *FileLoader {
	return &FileLoader{path: path}
}

func (l *FileLoader) Load() (*types.GlobalConfig, error) {
	// 读取文件内容
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}

	// 解析YAML
	var cfg types.GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
