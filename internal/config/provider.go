package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ConfigProvider 配置提供者接口
type ConfigProvider interface {
	Load(source string) error
	Get(key string) interface{}
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetConfig() *Config
	Watch(key string, callback func(value interface{})) error
}

// FileConfigProvider 文件配置提供者
type FileConfigProvider struct {
	mu       sync.RWMutex
	config   *Config
	rawData  map[string]interface{}
	filePath string
}

// NewFileConfigProvider 创建文件配置提供者
func NewFileConfigProvider() *FileConfigProvider {
	return &FileConfigProvider{
		config:  DefaultConfig(),
		rawData: make(map[string]interface{}),
	}
}

// Load 加载配置文件
func (p *FileConfigProvider) Load(source string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查文件是否存在
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", source)
	}

	// 读取文件内容
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	// 根据文件扩展名解析
	ext := strings.ToLower(filepath.Ext(source))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &p.rawData); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
		// 同时解析到配置结构体
		if err := yaml.Unmarshal(data, p.config); err != nil {
			return fmt.Errorf("failed to parse YAML to config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, &p.rawData); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
		// 同时解析到配置结构体
		if err := json.Unmarshal(data, p.config); err != nil {
			return fmt.Errorf("failed to parse JSON to config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	p.filePath = source
	return nil
}

// Get 获取配置值
func (p *FileConfigProvider) Get(key string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 支持嵌套键，如 "transport.timeout"
	keys := strings.Split(key, ".")
	value := interface{}(p.rawData)

	for _, k := range keys {
		switch v := value.(type) {
		case map[string]interface{}:
			value = v[k]
		case map[interface{}]interface{}:
			value = v[k]
		default:
			return nil
		}
		if value == nil {
			return nil
		}
	}

	return value
}

// GetString 获取字符串值
func (p *FileConfigProvider) GetString(key string) string {
	value := p.Get(key)
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// GetInt 获取整数值
func (p *FileConfigProvider) GetInt(key string) int {
	value := p.Get(key)
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		var i int
		fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

// GetBool 获取布尔值
func (p *FileConfigProvider) GetBool(key string) bool {
	value := p.Get(key)
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return strings.ToLower(v) == "true"
	default:
		return false
	}
}

// GetConfig 获取完整配置
func (p *FileConfigProvider) GetConfig() *Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// Watch 监听配置变化（文件提供者暂不支持）
func (p *FileConfigProvider) Watch(key string, callback func(value interface{})) error {
	return fmt.Errorf("file provider does not support watch")
}

// EnvConfigProvider 环境变量配置提供者
type EnvConfigProvider struct {
	mu      sync.RWMutex
	config  *Config
	prefix  string
	envVars map[string]string
}

// NewEnvConfigProvider 创建环境变量配置提供者
func NewEnvConfigProvider(prefix string) *EnvConfigProvider {
	return &EnvConfigProvider{
		config:  DefaultConfig(),
		prefix:  prefix,
		envVars: make(map[string]string),
	}
}

// Load 加载环境变量
func (p *EnvConfigProvider) Load(source string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 获取所有环境变量
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		// 如果有前缀，只加载匹配的变量
		if p.prefix != "" && !strings.HasPrefix(key, p.prefix) {
			continue
		}

		p.envVars[key] = value
	}

	// 应用环境变量到配置
	return LoadFromEnv(p.config)
}

// Get 获取配置值
func (p *EnvConfigProvider) Get(key string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// 转换键名为环境变量格式
	envKey := strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	if p.prefix != "" {
		envKey = p.prefix + "_" + envKey
	}

	return p.envVars[envKey]
}

// GetString 获取字符串值
func (p *EnvConfigProvider) GetString(key string) string {
	value := p.Get(key)
	if value == nil {
		return ""
	}
	return value.(string)
}

// GetInt 获取整数值
func (p *EnvConfigProvider) GetInt(key string) int {
	value := p.GetString(key)
	if value == "" {
		return 0
	}
	var i int
	fmt.Sscanf(value, "%d", &i)
	return i
}

// GetBool 获取布尔值
func (p *EnvConfigProvider) GetBool(key string) bool {
	value := p.GetString(key)
	return strings.ToLower(value) == "true"
}

// GetConfig 获取完整配置
func (p *EnvConfigProvider) GetConfig() *Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.config
}

// Watch 监听配置变化（环境变量提供者不支持）
func (p *EnvConfigProvider) Watch(key string, callback func(value interface{})) error {
	return fmt.Errorf("env provider does not support watch")
}

// MemoryConfigProvider 内存配置提供者（用于测试）
type MemoryConfigProvider struct {
	mu       sync.RWMutex
	config   *Config
	data     map[string]interface{}
	watchers map[string][]func(value interface{})
}

// NewMemoryConfigProvider 创建内存配置提供者
func NewMemoryConfigProvider() *MemoryConfigProvider {
	return &MemoryConfigProvider{
		config:   DefaultConfig(),
		data:     make(map[string]interface{}),
		watchers: make(map[string][]func(value interface{})),
	}
}

// Load 加载配置（内存提供者不需要实际加载）
func (p *MemoryConfigProvider) Load(source string) error {
	return nil
}

// Set 设置配置值
func (p *MemoryConfigProvider) Set(key string, value interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.data[key] = value

	// 通知观察者
	if watchers, ok := p.watchers[key]; ok {
		for _, callback := range watchers {
			go callback(value)
		}
	}
}

// Get 获取配置值
func (p *MemoryConfigProvider) Get(key string) interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.data[key]
}

// GetString 获取字符串值
func (p *MemoryConfigProvider) GetString(key string) string {
	value := p.Get(key)
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	default:
		return fmt.Sprint(v)
	}
}

// GetInt 获取整数值
func (p *MemoryConfigProvider) GetInt(key string) int {
	value := p.Get(key)
	if value == nil {
		return 0
	}
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

// GetBool 获取布尔值
func (p *MemoryConfigProvider) GetBool(key string) bool {
	value := p.Get(key)
	if value == nil {
		return false
	}
	switch v := value.(type) {
	case bool:
		return v
	default:
		return false
	}
}

// GetConfig 获取完整配置
func (p *MemoryConfigProvider) GetConfig() *Config {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// 返回配置的副本，避免外部修改
	if p.config == nil {
		return nil
	}
	return p.config
}

// Watch 监听配置变化
func (p *MemoryConfigProvider) Watch(key string, callback func(value interface{})) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.watchers[key] = append(p.watchers[key], callback)
	return nil
}

// SetConfig 设置完整配置
func (p *MemoryConfigProvider) SetConfig(config *Config) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
}