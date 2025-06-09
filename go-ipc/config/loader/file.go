package loader

import (
	"go-ipc/config"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadFromFile(path string, cfg *config.GlobalConfig) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}
