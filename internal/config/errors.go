package config

import "errors"

// 定义配置相关的错误变量
var (
    ErrConfigFileNotFound = errors.New("config file not found")
    ErrConfigParseFailed  = errors.New("failed to parse config")
    ErrConfigKeyNotFound  = errors.New("config key not found")
    ErrInvalidConfig      = errors.New("invalid config value")
)