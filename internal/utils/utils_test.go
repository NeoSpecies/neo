package utils_test

import (
	"bytes"
	"errors"
	"neo/internal/utils"
	"strings"
	"sync"
	"testing"
	_ "time"

	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
)

// writerFunc 实现 io.Writer 接口
type writerFunc struct {
	f func([]byte) (int, error)
}

func (w *writerFunc) Write(p []byte) (int, error) {
	return w.f(p)
}

// 测试日志器
func TestLogger(t *testing.T) {
	t.Run("基本日志记录", func(t *testing.T) {
		var buf bytes.Buffer
		logger := utils.NewLogger(
			utils.WithOutput(&buf),
			utils.WithoutColor(),
		)

		logger.Info("test message")
		assert.Contains(t, buf.String(), "[INFO]")
		assert.Contains(t, buf.String(), "test message")
	})

	t.Run("日志级别过滤", func(t *testing.T) {
		var buf bytes.Buffer
		logger := utils.NewLogger(
			utils.WithOutput(&buf),
			utils.WithLevel(utils.WARN),
			utils.WithoutColor(),
		)

		logger.Debug("debug message")
		logger.Info("info message")
		logger.Warn("warn message")
		logger.Error("error message")

		output := buf.String()
		assert.NotContains(t, output, "debug message")
		assert.NotContains(t, output, "info message")
		assert.Contains(t, output, "warn message")
		assert.Contains(t, output, "error message")
	})

	t.Run("结构化字段", func(t *testing.T) {
		var buf bytes.Buffer
		logger := utils.NewLogger(
			utils.WithOutput(&buf),
			utils.WithoutColor(),
		)

		logger.Info("test", utils.String("key", "value"), utils.Int("count", 42))
		output := buf.String()
		assert.Contains(t, output, "key=value")
		assert.Contains(t, output, "count=42")
	})

	t.Run("WithFields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := utils.NewLogger(
			utils.WithOutput(&buf),
			utils.WithoutColor(),
		)

		requestLogger := logger.WithFields(utils.String("request_id", "123"))
		requestLogger.Info("handling request")

		output := buf.String()
		assert.Contains(t, output, "request_id=123")
		assert.Contains(t, output, "handling request")
	})

	t.Run("前缀", func(t *testing.T) {
		var buf bytes.Buffer
		logger := utils.NewLogger(
			utils.WithOutput(&buf),
			utils.WithPrefix("MyService"),
			utils.WithoutColor(),
		)

		logger.Info("test")
		assert.Contains(t, buf.String(), "[MyService]")
	})

	t.Run("位置记录", func(t *testing.T) {
		var buf bytes.Buffer
		logger := utils.NewLogger(
			utils.WithOutput(&buf),
			utils.WithLocation(),
			utils.WithoutColor(),
		)

		logger.Info("test")
		output := buf.String()
		assert.Contains(t, output, "utils_test.go:")
	})

	t.Run("并发安全", func(t *testing.T) {
		var mu sync.Mutex
		var count int
		logger := utils.NewLogger(
			utils.WithOutput(&writerFunc{f: func(p []byte) (int, error) {
				mu.Lock()
				defer mu.Unlock()
				count += strings.Count(string(p), "\n")
				return len(p), nil
			}}),
			utils.WithoutColor(),
		)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				logger.Info("message", utils.Int("n", n))
			}(i)
		}
		wg.Wait()

		assert.Equal(t, 100, count)
	})
}

// 测试ID生成器
func TestIDGenerators(t *testing.T) {
	t.Run("UUID生成器", func(t *testing.T) {
		gen := utils.NewUUIDGenerator()
		ids := make(map[string]bool)

		for i := 0; i < 1000; i++ {
			id := gen.Generate()
			assert.NotEmpty(t, id)
			assert.False(t, ids[id], "UUID重复: %s", id)
			ids[id] = true
			
			// 验证UUID格式
			parts := strings.Split(id, "-")
			assert.Equal(t, 5, len(parts))
		}
	})

	t.Run("顺序ID生成器", func(t *testing.T) {
		gen := utils.NewSequentialIDGenerator("test")
		ids := make(map[string]bool)

		for i := 0; i < 100; i++ {
			id := gen.Generate()
			assert.NotEmpty(t, id)
			assert.True(t, strings.HasPrefix(id, "test-"))
			assert.False(t, ids[id])
			ids[id] = true
		}
	})

	t.Run("雪花ID生成器", func(t *testing.T) {
		gen := utils.NewSnowflakeIDGenerator(1, 1)
		ids := make(map[string]bool)

		for i := 0; i < 1000; i++ {
			id := gen.Generate()
			assert.NotEmpty(t, id)
			assert.False(t, ids[id], "雪花ID重复: %s", id)
			ids[id] = true
		}
	})

	t.Run("默认请求ID生成", func(t *testing.T) {
		id1 := utils.GenerateRequestID()
		id2 := utils.GenerateRequestID()
		assert.NotEmpty(t, id1)
		assert.NotEmpty(t, id2)
		assert.NotEqual(t, id1, id2)
	})

	t.Run("追踪ID生成", func(t *testing.T) {
		traceID := utils.GenerateTraceID()
		assert.Equal(t, 32, len(traceID)) // 16字节 = 32个hex字符

		spanID := utils.GenerateSpanID()
		assert.Equal(t, 16, len(spanID)) // 8字节 = 16个hex字符
	})

	t.Run("并发ID生成", func(t *testing.T) {
		gen := utils.NewUUIDGenerator()
		var wg sync.WaitGroup
		ids := sync.Map{}
		duplicates := 0

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					id := gen.Generate()
					if _, loaded := ids.LoadOrStore(id, true); loaded {
						duplicates++
					}
				}
			}()
		}

		wg.Wait()
		assert.Equal(t, 0, duplicates)
	})
}

// 测试字符串工具
func TestStringUtils(t *testing.T) {
	t.Run("ValidateServiceName", func(t *testing.T) {
		tests := []struct {
			name    string
			service string
			wantErr bool
		}{
			{"有效名称", "test.service", false},
			{"带下划线", "test_service", false},
			{"带短横线", "test-service", false},
			{"带数字", "service123", false},
			{"空名称", "", true},
			{"特殊字符", "test@service", true},
			{"太长", strings.Repeat("a", 256), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := utils.ValidateServiceName(tt.service)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ValidateMethodName", func(t *testing.T) {
		tests := []struct {
			name    string
			method  string
			wantErr bool
		}{
			{"有效名称", "testMethod", false},
			{"带下划线", "test_method", false},
			{"带数字", "method123", false},
			{"空名称", "", true},
			{"带点号", "test.method", true},
			{"太长", strings.Repeat("a", 101), true},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := utils.ValidateMethodName(tt.method)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("FormatEndpoint", func(t *testing.T) {
		endpoint := utils.FormatEndpoint("test.service", "testMethod")
		assert.Equal(t, "test.service/testMethod", endpoint)
	})

	t.Run("ParseEndpoint", func(t *testing.T) {
		service, method, err := utils.ParseEndpoint("test.service/testMethod")
		assert.NoError(t, err)
		assert.Equal(t, "test.service", service)
		assert.Equal(t, "testMethod", method)

		_, _, err = utils.ParseEndpoint("invalid")
		assert.Error(t, err)
	})

	t.Run("命名转换", func(t *testing.T) {
		// 驼峰命名
		assert.Equal(t, "helloWorld", utils.ToCamelCase("hello_world"))
		assert.Equal(t, "helloWorld", utils.ToCamelCase("hello-world"))
		assert.Equal(t, "helloWorld", utils.ToCamelCase("Hello World"))

		// 帕斯卡命名
		assert.Equal(t, "HelloWorld", utils.ToPascalCase("hello_world"))
		assert.Equal(t, "HelloWorld", utils.ToPascalCase("hello-world"))

		// 蛇形命名
		assert.Equal(t, "hello_world", utils.ToSnakeCase("HelloWorld"))
		assert.Equal(t, "hello_world", utils.ToSnakeCase("helloWorld"))

		// 烤串命名
		assert.Equal(t, "hello-world", utils.ToKebabCase("HelloWorld"))
		assert.Equal(t, "hello-world", utils.ToKebabCase("helloWorld"))
	})

	t.Run("Truncate", func(t *testing.T) {
		assert.Equal(t, "hello", utils.Truncate("hello", 10))
		assert.Equal(t, "hello...", utils.Truncate("hello world", 8))
		assert.Equal(t, "hel", utils.Truncate("hello", 3))
	})

	t.Run("IsEmpty", func(t *testing.T) {
		assert.True(t, utils.IsEmpty(""))
		assert.True(t, utils.IsEmpty("  "))
		assert.True(t, utils.IsEmpty("\t\n"))
		assert.False(t, utils.IsEmpty("hello"))
	})

	t.Run("ValidateEmail", func(t *testing.T) {
		assert.True(t, utils.ValidateEmail("test@example.com"))
		assert.True(t, utils.ValidateEmail("user.name+tag@example.co.uk"))
		assert.False(t, utils.ValidateEmail("invalid"))
		assert.False(t, utils.ValidateEmail("@example.com"))
		assert.False(t, utils.ValidateEmail("test@"))
	})

	t.Run("JoinPath", func(t *testing.T) {
		assert.Equal(t, "/api/v1/users", utils.JoinPath("api", "v1", "users"))
		assert.Equal(t, "/api/v1/users", utils.JoinPath("/api/", "/v1/", "/users/"))
		assert.Equal(t, "/api/users", utils.JoinPath("api", "", "users"))
	})

	t.Run("EscapeHTML", func(t *testing.T) {
		assert.Equal(t, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
			utils.EscapeHTML("<script>alert('xss')</script>"))
		assert.Equal(t, "&quot;hello&quot; &amp; &#39;world&#39;",
			utils.EscapeHTML("\"hello\" & 'world'"))
	})
}

// 测试错误处理
func TestErrors(t *testing.T) {
	t.Run("NewError", func(t *testing.T) {
		err := utils.NewError(utils.ErrCodeNotFound, "resource not found")
		assert.Equal(t, utils.ErrCodeNotFound, err.Code)
		assert.Equal(t, "resource not found", err.Message)
		assert.NotEmpty(t, err.StackTrace)
		assert.Contains(t, err.Error(), "[NOT_FOUND]")
	})

	t.Run("NewErrorf", func(t *testing.T) {
		err := utils.NewErrorf(utils.ErrCodeInvalidArgument, "invalid %s: %d", "count", 42)
		assert.Equal(t, "invalid count: 42", err.Message)
	})

	t.Run("WrapError", func(t *testing.T) {
		cause := errors.New("database error")
		err := utils.WrapError(cause, utils.ErrCodeInternal, "failed to query")
		assert.Equal(t, utils.ErrCodeInternal, err.Code)
		assert.Equal(t, cause, err.Unwrap())
		assert.Contains(t, err.Error(), "database error")
	})

	t.Run("WithDetail", func(t *testing.T) {
		err := utils.NewError(utils.ErrCodeInvalidArgument, "invalid input")
		err.WithDetail("field", "email").WithDetail("value", "invalid@")
		
		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "invalid@", err.Details["value"])
	})

	t.Run("IsError", func(t *testing.T) {
		err := utils.NewError(utils.ErrCodeNotFound, "not found")
		assert.True(t, utils.IsError(err, utils.ErrCodeNotFound))
		assert.False(t, utils.IsError(err, utils.ErrCodeInternal))
		assert.False(t, utils.IsError(nil, utils.ErrCodeNotFound))
	})

	t.Run("ErrorChain", func(t *testing.T) {
		chain := &utils.ErrorChain{}
		assert.False(t, chain.HasErrors())

		err1 := utils.NewError(utils.ErrCodeInvalidArgument, "invalid field1")
		err2 := utils.NewError(utils.ErrCodeInvalidArgument, "invalid field2")
		
		chain.Add(err1)
		chain.Add(err2)
		
		assert.True(t, chain.HasErrors())
		assert.Equal(t, 2, len(chain.Errors()))
		assert.Equal(t, err1, chain.First())
		assert.Contains(t, chain.Error(), "multiple errors")
	})
}

// 性能基准测试
func BenchmarkUUIDGenerate(b *testing.B) {
	gen := utils.NewUUIDGenerator()
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkSnowflakeGenerate(b *testing.B) {
	gen := utils.NewSnowflakeIDGenerator(1, 1)
	for i := 0; i < b.N; i++ {
		gen.Generate()
	}
}

func BenchmarkLogger(b *testing.B) {
	logger := utils.NewLogger(
		utils.WithOutput(bytes.NewBuffer(nil)),
		utils.WithoutColor(),
	)
	
	b.Run("Simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("test message")
		}
	})
	
	b.Run("WithFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("test", utils.String("key", "value"), utils.Int("count", i))
		}
	})
}

func BenchmarkValidateServiceName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utils.ValidateServiceName("test.service.name")
	}
}

func BenchmarkToCamelCase(b *testing.B) {
	for i := 0; i < b.N; i++ {
		utils.ToCamelCase("hello_world_test")
	}
}