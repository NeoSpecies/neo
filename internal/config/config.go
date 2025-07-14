package config

import (
	"encoding/json"
	"fmt"
	"neo/internal/utils"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration 自定义Duration类型，支持JSON序列化
type Duration time.Duration

// UnmarshalJSON 实现JSON反序列化
func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		*d = Duration(time.Duration(value))
		return nil
	case string:
		dur, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		*d = Duration(dur)
		return nil
	default:
		return fmt.Errorf("invalid duration")
	}
}

// MarshalJSON 实现JSON序列化
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalYAML 实现YAML反序列化
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	dur, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = Duration(dur)
	return nil
}

// MarshalYAML 实现YAML序列化
func (d Duration) MarshalYAML() (interface{}, error) {
	return time.Duration(d).String(), nil
}

// TransportConfig 传输层配置
type TransportConfig struct {
	Timeout         Duration          `yaml:"timeout" json:"timeout" env:"NEO_TRANSPORT_TIMEOUT"`
	RetryCount      int               `yaml:"retry_count" json:"retry_count" env:"NEO_TRANSPORT_RETRY_COUNT"`
	MaxConnections  int               `yaml:"max_connections" json:"max_connections" env:"NEO_TRANSPORT_MAX_CONNECTIONS"`
	MinConnections  int               `yaml:"min_connections" json:"min_connections" env:"NEO_TRANSPORT_MIN_CONNECTIONS"`
	MaxIdleTime     Duration          `yaml:"max_idle_time" json:"max_idle_time" env:"NEO_TRANSPORT_MAX_IDLE_TIME"`
	HealthCheckInterval Duration      `yaml:"health_check_interval" json:"health_check_interval" env:"NEO_TRANSPORT_HEALTH_CHECK_INTERVAL"`
	InitialBackoff  Duration          `yaml:"initial_backoff" json:"initial_backoff" env:"NEO_TRANSPORT_INITIAL_BACKOFF"`
	MaxBackoff      Duration          `yaml:"max_backoff" json:"max_backoff" env:"NEO_TRANSPORT_MAX_BACKOFF"`
	Multiplier      float64           `yaml:"multiplier" json:"multiplier" env:"NEO_TRANSPORT_MULTIPLIER"`
}

// RegistryConfig 注册中心配置
type RegistryConfig struct {
	Type            string   `yaml:"type" json:"type" env:"NEO_REGISTRY_TYPE"`
	Address         string   `yaml:"address" json:"address" env:"NEO_REGISTRY_ADDRESS"`
	Namespace       string   `yaml:"namespace" json:"namespace" env:"NEO_REGISTRY_NAMESPACE"`
	TTL             Duration `yaml:"ttl" json:"ttl" env:"NEO_REGISTRY_TTL"`
	RefreshInterval Duration `yaml:"refresh_interval" json:"refresh_interval" env:"NEO_REGISTRY_REFRESH_INTERVAL"`
}

// GatewayConfig HTTP网关配置
type GatewayConfig struct {
	Address         string   `yaml:"address" json:"address" env:"NEO_GATEWAY_ADDRESS"`
	ReadTimeout     Duration `yaml:"read_timeout" json:"read_timeout" env:"NEO_GATEWAY_READ_TIMEOUT"`
	WriteTimeout    Duration `yaml:"write_timeout" json:"write_timeout" env:"NEO_GATEWAY_WRITE_TIMEOUT"`
	MaxHeaderBytes  int      `yaml:"max_header_bytes" json:"max_header_bytes" env:"NEO_GATEWAY_MAX_HEADER_BYTES"`
	ShutdownTimeout Duration `yaml:"shutdown_timeout" json:"shutdown_timeout" env:"NEO_GATEWAY_SHUTDOWN_TIMEOUT"`
}

// IPCConfig IPC配置
type IPCConfig struct {
	Address      string   `yaml:"address" json:"address" env:"NEO_IPC_ADDRESS"`
	MaxClients   int      `yaml:"max_clients" json:"max_clients" env:"NEO_IPC_MAX_CLIENTS"`
	BufferSize   int      `yaml:"buffer_size" json:"buffer_size" env:"NEO_IPC_BUFFER_SIZE"`
	ReadTimeout  Duration `yaml:"read_timeout" json:"read_timeout" env:"NEO_IPC_READ_TIMEOUT"`
	WriteTimeout Duration `yaml:"write_timeout" json:"write_timeout" env:"NEO_IPC_WRITE_TIMEOUT"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level" json:"level" env:"NEO_LOG_LEVEL"`
	Output     string `yaml:"output" json:"output" env:"NEO_LOG_OUTPUT"`
	Format     string `yaml:"format" json:"format" env:"NEO_LOG_FORMAT"`
	WithColor  bool   `yaml:"with_color" json:"with_color" env:"NEO_LOG_WITH_COLOR"`
	WithLocation bool `yaml:"with_location" json:"with_location" env:"NEO_LOG_WITH_LOCATION"`
}

// Config 主配置结构
type Config struct {
	Transport TransportConfig `yaml:"transport" json:"transport"`
	Registry  RegistryConfig  `yaml:"registry" json:"registry"`
	Gateway   GatewayConfig   `yaml:"gateway" json:"gateway"`
	IPC       IPCConfig       `yaml:"ipc" json:"ipc"`
	Log       LogConfig       `yaml:"log" json:"log"`
	Mode      string          `yaml:"mode" json:"mode" env:"NEO_MODE"` // debug/release/test
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Transport: TransportConfig{
			Timeout:             Duration(30 * time.Second),
			RetryCount:          3,
			MaxConnections:      100,
			MinConnections:      10,
			MaxIdleTime:         Duration(5 * time.Minute),
			HealthCheckInterval: Duration(30 * time.Second),
			InitialBackoff:      Duration(100 * time.Millisecond),
			MaxBackoff:          Duration(5 * time.Second),
			Multiplier:          2.0,
		},
		Registry: RegistryConfig{
			Type:            "inmemory",
			Address:         "",
			Namespace:       "default",
			TTL:             Duration(30 * time.Second),
			RefreshInterval: Duration(10 * time.Second),
		},
		Gateway: GatewayConfig{
			Address:         ":8080",
			ReadTimeout:     Duration(30 * time.Second),
			WriteTimeout:    Duration(30 * time.Second),
			MaxHeaderBytes:  1 << 20, // 1MB
			ShutdownTimeout: Duration(10 * time.Second),
		},
		IPC: IPCConfig{
			Address:      ":9999",
			MaxClients:   1000,
			BufferSize:   4096,
			ReadTimeout:  Duration(30 * time.Second),
			WriteTimeout: Duration(30 * time.Second),
		},
		Log: LogConfig{
			Level:        "info",
			Output:       "stdout",
			Format:       "text",
			WithColor:    true,
			WithLocation: false,
		},
		Mode: "debug",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证传输配置
	if time.Duration(c.Transport.Timeout) <= 0 {
		return fmt.Errorf("transport timeout must be positive")
	}
	if c.Transport.RetryCount < 0 {
		return fmt.Errorf("transport retry count cannot be negative")
	}
	if c.Transport.MaxConnections <= 0 {
		return fmt.Errorf("transport max connections must be positive")
	}
	if c.Transport.MinConnections < 0 || c.Transport.MinConnections > c.Transport.MaxConnections {
		return fmt.Errorf("transport min connections must be between 0 and max connections")
	}

	// 验证注册中心配置
	validRegistryTypes := map[string]bool{"inmemory": true, "etcd": true, "consul": true}
	if !validRegistryTypes[c.Registry.Type] {
		return fmt.Errorf("invalid registry type: %s", c.Registry.Type)
	}

	// 验证日志配置
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}

	// 验证模式
	validModes := map[string]bool{"debug": true, "release": true, "test": true}
	if !validModes[c.Mode] {
		return fmt.Errorf("invalid mode: %s", c.Mode)
	}

	return nil
}

// ConfigManager 配置管理器
type ConfigManager struct {
	mu        sync.RWMutex
	config    *Config
	providers []ConfigProvider
	watchers  []func(*Config)
	logger    utils.Logger
}

// NewConfigManager 创建配置管理器
func NewConfigManager(logger utils.Logger) *ConfigManager {
	if logger == nil {
		logger = utils.DefaultLogger
	}
	return &ConfigManager{
		config:    DefaultConfig(),
		providers: make([]ConfigProvider, 0),
		watchers:  make([]func(*Config), 0),
		logger:    logger,
	}
}

// AddProvider 添加配置提供者
func (cm *ConfigManager) AddProvider(provider ConfigProvider) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.providers = append(cm.providers, provider)
}

// Load 加载配置
func (cm *ConfigManager) Load(source string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// 从默认配置开始
	newConfig := DefaultConfig()

	// 应用所有提供者
	for _, provider := range cm.providers {
		if err := provider.Load(source); err != nil {
			cm.logger.Warn("failed to load from provider",
				utils.String("source", source),
				utils.ErrorField(err))
			continue
		}

		// 合并配置
		providerConfig := provider.GetConfig()
		if err := mergeConfig(newConfig, providerConfig); err != nil {
			return fmt.Errorf("failed to merge config: %w", err)
		}
	}

	// 验证配置
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// 更新配置
	cm.config = newConfig

	// 通知观察者
	cm.notifyWatchers(newConfig)

	return nil
}

// Get 获取当前配置
func (cm *ConfigManager) Get() *Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.config
}

// Watch 监听配置变化
func (cm *ConfigManager) Watch(callback func(*Config)) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.watchers = append(cm.watchers, callback)
}

// notifyWatchers 通知观察者
func (cm *ConfigManager) notifyWatchers(config *Config) {
	for _, watcher := range cm.watchers {
		go watcher(config)
	}
}

// mergeConfig 合并配置
func mergeConfig(dst, src *Config) error {
	if src == nil {
		return nil
	}

	// 使用反射合并
	dstValue := reflect.ValueOf(dst).Elem()
	srcValue := reflect.ValueOf(src).Elem()

	return mergeStruct(dstValue, srcValue)
}

// mergeStruct 递归合并结构体
func mergeStruct(dst, src reflect.Value) error {
	if dst.Kind() != reflect.Struct || src.Kind() != reflect.Struct {
		return fmt.Errorf("both values must be structs")
	}

	for i := 0; i < src.NumField(); i++ {
		srcField := src.Field(i)
		dstField := dst.Field(i)

		if !srcField.IsValid() {
			continue
		}

		switch srcField.Kind() {
		case reflect.Struct:
			if err := mergeStruct(dstField, srcField); err != nil {
				return err
			}
		default:
			// 只有当src字段不是零值时才覆盖dst
			if dstField.CanSet() && !isZeroValue(srcField) {
				dstField.Set(srcField)
			}
		}
	}

	return nil
}

// isZeroValue 检查值是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Complex64, reflect.Complex128:
		return v.Complex() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Interface, reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return v.IsNil()
	default:
		return false
	}
}

// LoadFromFile 从文件加载配置
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	
	// 根据文件扩展名选择解析器
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported config file format: %s", ext)
	}

	// 应用环境变量覆盖
	if err := LoadFromEnv(config); err != nil {
		return nil, fmt.Errorf("failed to load from env: %w", err)
	}

	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return config, nil
}

// LoadFromEnv 从环境变量加载配置
func LoadFromEnv(config *Config) error {
	return loadStructFromEnv(reflect.ValueOf(config).Elem(), "")
}

// loadStructFromEnv 递归从环境变量加载结构体
func loadStructFromEnv(v reflect.Value, prefix string) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 处理嵌套结构体
		if field.Kind() == reflect.Struct && fieldType.Tag.Get("env") == "" {
			// 递归处理嵌套结构体
			if err := loadStructFromEnv(field, prefix); err != nil {
				return err
			}
			continue
		}

		// 获取环境变量名
		envTag := fieldType.Tag.Get("env")
		if envTag == "" {
			continue
		}

		// 获取环境变量值
		envValue := os.Getenv(envTag)
		if envValue == "" {
			continue
		}

		// 根据字段类型设置值
		switch field.Kind() {
		case reflect.String:
			field.SetString(envValue)
		case reflect.Int, reflect.Int64:
			if field.Type() == reflect.TypeOf(Duration(0)) {
				// 解析时间间隔
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return fmt.Errorf("invalid duration for %s: %w", envTag, err)
				}
				field.SetInt(int64(duration))
			} else if field.Type() == reflect.TypeOf(time.Duration(0)) {
				// 解析time.Duration
				duration, err := time.ParseDuration(envValue)
				if err != nil {
					return fmt.Errorf("invalid duration for %s: %w", envTag, err)
				}
				field.SetInt(int64(duration))
			} else {
				// 解析整数
				intValue, err := strconv.ParseInt(envValue, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid int for %s: %w", envTag, err)
				}
				field.SetInt(intValue)
			}
		case reflect.Float64:
			floatValue, err := strconv.ParseFloat(envValue, 64)
			if err != nil {
				return fmt.Errorf("invalid float for %s: %w", envTag, err)
			}
			field.SetFloat(floatValue)
		case reflect.Bool:
			boolValue, err := strconv.ParseBool(envValue)
			if err != nil {
				return fmt.Errorf("invalid bool for %s: %w", envTag, err)
			}
			field.SetBool(boolValue)
		}
	}

	return nil
}