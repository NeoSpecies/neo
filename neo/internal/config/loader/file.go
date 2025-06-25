package loader

import (
	"io/ioutil"
	"gopkg.in/yaml.v3"
	"neo/internal/config"
)

// FileLoader 从文件加载配置
type FileLoader struct {
	path string
}

// NewFileLoader 创建新的文件配置加载器
func NewFileLoader(path string) *FileLoader {
	return &FileLoader{path: path}
}

// Load 加载并解析配置文件
func (l *FileLoader) Load() (*config.GlobalConfig, error) {
	// 读取文件内容
	data, err := ioutil.ReadFile(l.path)
	if err != nil {
		return nil, err
	}

	// 解析YAML
	var cfg config.GlobalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
