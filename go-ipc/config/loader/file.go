package loader

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadFromFile(path string, cfg interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("读取文件 %s 时出错: %v\n", path, err)
		return err
	}
	// fmt.Printf("成功读取文件 %s，内容长度: %d\n", path, len(data))

	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		fmt.Printf("解析 YAML 文件 %s 时出错: %v\n", path, err)
		return err
	}
	// fmt.Printf("成功解析 YAML 文件 %s\n", path)

	return nil
}
