package config_test

import (
	"neo/internal/config"
	"neo/internal/utils"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	// 传输配置
	assert.Equal(t, config.Duration(30*time.Second), cfg.Transport.Timeout)
	assert.Equal(t, 3, cfg.Transport.RetryCount)
	assert.Equal(t, 100, cfg.Transport.MaxConnections)
	assert.Equal(t, 10, cfg.Transport.MinConnections)
	assert.Equal(t, config.Duration(5*time.Minute), cfg.Transport.MaxIdleTime)
	assert.Equal(t, config.Duration(30*time.Second), cfg.Transport.HealthCheckInterval)
	assert.Equal(t, config.Duration(100*time.Millisecond), cfg.Transport.InitialBackoff)
	assert.Equal(t, config.Duration(5*time.Second), cfg.Transport.MaxBackoff)
	assert.Equal(t, 2.0, cfg.Transport.Multiplier)

	// 注册中心配置
	assert.Equal(t, "inmemory", cfg.Registry.Type)
	assert.Equal(t, "default", cfg.Registry.Namespace)
	assert.Equal(t, config.Duration(30*time.Second), cfg.Registry.TTL)
	assert.Equal(t, config.Duration(10*time.Second), cfg.Registry.RefreshInterval)

	// 网关配置
	assert.Equal(t, ":8080", cfg.Gateway.Address)
	assert.Equal(t, config.Duration(30*time.Second), cfg.Gateway.ReadTimeout)
	assert.Equal(t, config.Duration(30*time.Second), cfg.Gateway.WriteTimeout)
	assert.Equal(t, 1<<20, cfg.Gateway.MaxHeaderBytes)
	assert.Equal(t, config.Duration(10*time.Second), cfg.Gateway.ShutdownTimeout)

	// IPC配置
	assert.Equal(t, ":9999", cfg.IPC.Address)
	assert.Equal(t, 1000, cfg.IPC.MaxClients)
	assert.Equal(t, 4096, cfg.IPC.BufferSize)
	assert.Equal(t, config.Duration(30*time.Second), cfg.IPC.ReadTimeout)
	assert.Equal(t, config.Duration(30*time.Second), cfg.IPC.WriteTimeout)

	// 日志配置
	assert.Equal(t, "info", cfg.Log.Level)
	assert.Equal(t, "stdout", cfg.Log.Output)
	assert.Equal(t, "text", cfg.Log.Format)
	assert.True(t, cfg.Log.WithColor)
	assert.False(t, cfg.Log.WithLocation)

	// 模式
	assert.Equal(t, "debug", cfg.Mode)
}

// 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*config.Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "有效配置",
			modify:  func(c *config.Config) {},
			wantErr: false,
		},
		{
			name: "无效传输超时",
			modify: func(c *config.Config) {
				c.Transport.Timeout = 0
			},
			wantErr: true,
			errMsg:  "transport timeout must be positive",
		},
		{
			name: "无效重试次数",
			modify: func(c *config.Config) {
				c.Transport.RetryCount = -1
			},
			wantErr: true,
			errMsg:  "transport retry count cannot be negative",
		},
		{
			name: "无效最大连接数",
			modify: func(c *config.Config) {
				c.Transport.MaxConnections = 0
			},
			wantErr: true,
			errMsg:  "transport max connections must be positive",
		},
		{
			name: "无效最小连接数",
			modify: func(c *config.Config) {
				c.Transport.MinConnections = 200
				c.Transport.MaxConnections = 100
			},
			wantErr: true,
			errMsg:  "transport min connections must be between 0 and max connections",
		},
		{
			name: "无效注册中心类型",
			modify: func(c *config.Config) {
				c.Registry.Type = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid registry type",
		},
		{
			name: "无效日志级别",
			modify: func(c *config.Config) {
				c.Log.Level = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid log level",
		},
		{
			name: "无效模式",
			modify: func(c *config.Config) {
				c.Mode = "invalid"
			},
			wantErr: true,
			errMsg:  "invalid mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.modify(cfg)
			err := cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// 测试从文件加载配置
func TestLoadFromFile(t *testing.T) {
	t.Run("YAML文件", func(t *testing.T) {
		// 创建临时YAML文件
		tmpDir := t.TempDir()
		yamlFile := filepath.Join(tmpDir, "config.yaml")
		yamlContent := `
transport:
  timeout: 60s
  retry_count: 5
  max_connections: 200
registry:
  type: etcd
  address: localhost:2379
gateway:
  address: :8088
log:
  level: debug
  with_color: false
mode: release
`
		require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

		// 加载配置
		cfg, err := config.LoadFromFile(yamlFile)
		require.NoError(t, err)

		// 验证配置
		assert.Equal(t, config.Duration(60*time.Second), cfg.Transport.Timeout)
		assert.Equal(t, 5, cfg.Transport.RetryCount)
		assert.Equal(t, 200, cfg.Transport.MaxConnections)
		assert.Equal(t, "etcd", cfg.Registry.Type)
		assert.Equal(t, "localhost:2379", cfg.Registry.Address)
		assert.Equal(t, ":8088", cfg.Gateway.Address)
		assert.Equal(t, "debug", cfg.Log.Level)
		assert.False(t, cfg.Log.WithColor)
		assert.Equal(t, "release", cfg.Mode)
	})

	t.Run("JSON文件", func(t *testing.T) {
		// 创建临时JSON文件
		tmpDir := t.TempDir()
		jsonFile := filepath.Join(tmpDir, "config.json")
		jsonContent := `{
			"transport": {
				"timeout": "45s",
				"retry_count": 2
			},
			"registry": {
				"type": "consul"
			},
			"log": {
				"level": "warn"
			}
		}`
		require.NoError(t, os.WriteFile(jsonFile, []byte(jsonContent), 0644))

		// 加载配置
		cfg, err := config.LoadFromFile(jsonFile)
		require.NoError(t, err)

		// 验证配置
		assert.Equal(t, config.Duration(45*time.Second), cfg.Transport.Timeout)
		assert.Equal(t, 2, cfg.Transport.RetryCount)
		assert.Equal(t, "consul", cfg.Registry.Type)
		assert.Equal(t, "warn", cfg.Log.Level)
	})

	t.Run("文件不存在", func(t *testing.T) {
		_, err := config.LoadFromFile("nonexistent.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})

	t.Run("不支持的文件格式", func(t *testing.T) {
		tmpDir := t.TempDir()
		txtFile := filepath.Join(tmpDir, "config.txt")
		require.NoError(t, os.WriteFile(txtFile, []byte("test"), 0644))

		_, err := config.LoadFromFile(txtFile)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported config file format")
	})
}

// 测试环境变量加载
func TestLoadFromEnv(t *testing.T) {
	// 设置环境变量
	envVars := map[string]string{
		"NEO_TRANSPORT_TIMEOUT":        "120s",
		"NEO_TRANSPORT_RETRY_COUNT":    "10",
		"NEO_TRANSPORT_MAX_CONNECTIONS": "500",
		"NEO_TRANSPORT_MULTIPLIER":     "3.5",
		"NEO_REGISTRY_TYPE":            "etcd",
		"NEO_GATEWAY_ADDRESS":          ":9090",
		"NEO_LOG_LEVEL":                "error",
		"NEO_LOG_WITH_COLOR":           "false",
		"NEO_MODE":                     "test",
	}

	// 设置环境变量
	for k, v := range envVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	// 加载配置
	cfg := config.DefaultConfig()
	err := config.LoadFromEnv(cfg)
	require.NoError(t, err)

	// 验证配置
	assert.Equal(t, config.Duration(120*time.Second), cfg.Transport.Timeout)
	assert.Equal(t, 10, cfg.Transport.RetryCount)
	assert.Equal(t, 500, cfg.Transport.MaxConnections)
	assert.Equal(t, 3.5, cfg.Transport.Multiplier)
	assert.Equal(t, "etcd", cfg.Registry.Type)
	assert.Equal(t, ":9090", cfg.Gateway.Address)
	assert.Equal(t, "error", cfg.Log.Level)
	assert.False(t, cfg.Log.WithColor)
	assert.Equal(t, "test", cfg.Mode)
}

// 测试配置管理器
func TestConfigManager(t *testing.T) {
	logger := utils.NewLogger(utils.WithOutput(&testWriter{}))
	
	t.Run("基本操作", func(t *testing.T) {
		cm := config.NewConfigManager(logger)
		
		// 获取默认配置
		cfg := cm.Get()
		assert.NotNil(t, cfg)
		assert.Equal(t, "debug", cfg.Mode)

		// 添加内存提供者
		provider := config.NewMemoryConfigProvider()
		testConfig := config.DefaultConfig()
		testConfig.Mode = "test"
		provider.SetConfig(testConfig)
		
		cm.AddProvider(provider)
		err := cm.Load("")
		assert.NoError(t, err)

		// 验证配置已更新
		cfg = cm.Get()
		assert.Equal(t, "test", cfg.Mode)
	})

	t.Run("配置监听", func(t *testing.T) {
		cm := config.NewConfigManager(logger)
		
		var notified bool
		var wg sync.WaitGroup
		wg.Add(1)

		// 添加监听器
		cm.Watch(func(cfg *config.Config) {
			notified = true
			wg.Done()
		})

		// 触发配置更新
		provider := config.NewMemoryConfigProvider()
		cm.AddProvider(provider)
		err := cm.Load("")
		assert.NoError(t, err)

		// 等待通知
		wg.Wait()
		assert.True(t, notified)
	})

	t.Run("多提供者", func(t *testing.T) {
		cm := config.NewConfigManager(logger)

		// 添加多个提供者
		provider1 := config.NewMemoryConfigProvider()
		cfg1 := config.DefaultConfig()
		cfg1.Mode = "test"
		cfg1.Log.Level = "debug"
		provider1.SetConfig(cfg1)

		provider2 := config.NewMemoryConfigProvider()
		cfg2 := &config.Config{
			Log: config.LogConfig{
				Level: "error", // 只设置要覆盖的字段
			},
		}
		provider2.SetConfig(cfg2)

		cm.AddProvider(provider1)
		cm.AddProvider(provider2)

		err := cm.Load("")
		assert.NoError(t, err)

		// 验证配置合并结果
		cfg := cm.Get()
		assert.Equal(t, "test", cfg.Mode) // 来自provider1
		assert.Equal(t, "error", cfg.Log.Level) // 被provider2覆盖
	})
}

// 测试文件配置提供者
func TestFileConfigProvider(t *testing.T) {
	t.Run("加载YAML文件", func(t *testing.T) {
		tmpDir := t.TempDir()
		yamlFile := filepath.Join(tmpDir, "test.yaml")
		yamlContent := `
transport:
  timeout: 45s
  retry_count: 3
log:
  level: warn
nested:
  key: value
`
		require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

		provider := config.NewFileConfigProvider()
		err := provider.Load(yamlFile)
		require.NoError(t, err)

		// 测试Get方法
		assert.Equal(t, "value", provider.GetString("nested.key"))
		assert.Equal(t, 0, provider.GetInt("nonexistent"))
		assert.False(t, provider.GetBool("nonexistent"))

		// 测试GetConfig
		cfg := provider.GetConfig()
		assert.Equal(t, config.Duration(45*time.Second), cfg.Transport.Timeout)
		assert.Equal(t, 3, cfg.Transport.RetryCount)
		assert.Equal(t, "warn", cfg.Log.Level)
	})

	t.Run("不支持Watch", func(t *testing.T) {
		provider := config.NewFileConfigProvider()
		err := provider.Watch("key", func(value interface{}) {})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not support watch")
	})
}

// 测试内存配置提供者
func TestMemoryConfigProvider(t *testing.T) {
	t.Run("基本操作", func(t *testing.T) {
		provider := config.NewMemoryConfigProvider()

		// 设置值
		provider.Set("key1", "value1")
		provider.Set("key2", 42)
		provider.Set("key3", true)

		// 获取值
		assert.Equal(t, "value1", provider.GetString("key1"))
		assert.Equal(t, 42, provider.GetInt("key2"))
		assert.True(t, provider.GetBool("key3"))

		// 不存在的键
		assert.Equal(t, "", provider.GetString("nonexistent"))
		assert.Equal(t, 0, provider.GetInt("nonexistent"))
		assert.False(t, provider.GetBool("nonexistent"))
	})

	t.Run("Watch功能", func(t *testing.T) {
		provider := config.NewMemoryConfigProvider()

		var notified bool
		var notifiedValue interface{}
		var wg sync.WaitGroup
		wg.Add(1)

		// 注册监听器
		err := provider.Watch("test.key", func(value interface{}) {
			notified = true
			notifiedValue = value
			wg.Done()
		})
		assert.NoError(t, err)

		// 触发变化
		provider.Set("test.key", "new value")

		// 等待通知
		wg.Wait()
		assert.True(t, notified)
		assert.Equal(t, "new value", notifiedValue)
	})
}

// 测试环境变量提供者
func TestEnvConfigProvider(t *testing.T) {
	// 设置测试环境变量
	os.Setenv("TEST_KEY1", "value1")
	os.Setenv("TEST_KEY2", "123")
	os.Setenv("TEST_KEY3", "true")
	defer func() {
		os.Unsetenv("TEST_KEY1")
		os.Unsetenv("TEST_KEY2")
		os.Unsetenv("TEST_KEY3")
	}()

	t.Run("带前缀", func(t *testing.T) {
		provider := config.NewEnvConfigProvider("TEST")
		err := provider.Load("")
		assert.NoError(t, err)

		assert.Equal(t, "value1", provider.GetString("key1"))
		assert.Equal(t, 123, provider.GetInt("key2"))
		assert.True(t, provider.GetBool("key3"))
	})

	t.Run("不支持Watch", func(t *testing.T) {
		provider := config.NewEnvConfigProvider("")
		err := provider.Watch("key", func(value interface{}) {})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not support watch")
	})
}

// 测试配置合并
func TestConfigMerge(t *testing.T) {
	tmpDir := t.TempDir()
	yamlFile := filepath.Join(tmpDir, "partial.yaml")
	yamlContent := `
transport:
  timeout: 60s
log:
  level: debug
`
	require.NoError(t, os.WriteFile(yamlFile, []byte(yamlContent), 0644))

	// 设置环境变量（会覆盖文件配置）
	os.Setenv("NEO_LOG_LEVEL", "error")
	defer os.Unsetenv("NEO_LOG_LEVEL")

	// 加载配置
	cfg, err := config.LoadFromFile(yamlFile)
	require.NoError(t, err)

	// 验证文件配置被加载
	assert.Equal(t, config.Duration(60*time.Second), cfg.Transport.Timeout)
	
	// 验证环境变量覆盖了文件配置
	assert.Equal(t, "error", cfg.Log.Level)
	
	// 验证其他字段使用默认值
	assert.Equal(t, 3, cfg.Transport.RetryCount) // 默认值
}

// 性能基准测试
func BenchmarkConfigLoad(b *testing.B) {
	tmpDir := b.TempDir()
	yamlFile := filepath.Join(tmpDir, "bench.yaml")
	yamlContent := `
transport:
  timeout: 30s
  retry_count: 3
registry:
  type: inmemory
log:
  level: info
`
	os.WriteFile(yamlFile, []byte(yamlContent), 0644)

	b.Run("LoadFromFile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			config.LoadFromFile(yamlFile)
		}
	})

	b.Run("Validate", func(b *testing.B) {
		cfg := config.DefaultConfig()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cfg.Validate()
		}
	})
}

func BenchmarkConfigManager(b *testing.B) {
	cm := config.NewConfigManager(nil)
	provider := config.NewMemoryConfigProvider()
	cm.AddProvider(provider)

	b.Run("Get", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cm.Get()
		}
	})

	b.Run("Load", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			cm.Load("")
		}
	})
}

// 测试辅助类型
type testWriter struct{}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}