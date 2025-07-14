package utils

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	// DEBUG 调试级别
	DEBUG LogLevel = iota
	// INFO 信息级别
	INFO
	// WARN 警告级别
	WARN
	// ERROR 错误级别
	ERROR
)

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// Logger 日志接口
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	WithFields(fields ...Field) Logger
	SetLevel(level LogLevel)
}

// StructuredLogger 结构化日志实现
type StructuredLogger struct {
	mu       sync.RWMutex
	output   io.Writer
	level    LogLevel
	fields   []Field
	prefix   string
	colored  bool
	location bool // 是否记录调用位置
}

// NewLogger 创建新的日志记录器
func NewLogger(options ...LoggerOption) Logger {
	logger := &StructuredLogger{
		output:  os.Stdout,
		level:   INFO,
		fields:  make([]Field, 0),
		colored: true,
		location: false,
	}

	for _, opt := range options {
		opt(logger)
	}

	return logger
}

// LoggerOption 日志选项
type LoggerOption func(*StructuredLogger)

// WithOutput 设置输出
func WithOutput(output io.Writer) LoggerOption {
	return func(l *StructuredLogger) {
		l.output = output
	}
}

// WithLevel 设置日志级别
func WithLevel(level LogLevel) LoggerOption {
	return func(l *StructuredLogger) {
		l.level = level
	}
}

// WithPrefix 设置前缀
func WithPrefix(prefix string) LoggerOption {
	return func(l *StructuredLogger) {
		l.prefix = prefix
	}
}

// WithoutColor 禁用颜色
func WithoutColor() LoggerOption {
	return func(l *StructuredLogger) {
		l.colored = false
	}
}

// WithLocation 启用位置记录
func WithLocation() LoggerOption {
	return func(l *StructuredLogger) {
		l.location = true
	}
}

// SetLevel 设置日志级别
func (l *StructuredLogger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// WithFields 创建带字段的新日志器
func (l *StructuredLogger) WithFields(fields ...Field) Logger {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLogger := &StructuredLogger{
		output:   l.output,
		level:    l.level,
		prefix:   l.prefix,
		colored:  l.colored,
		location: l.location,
		fields:   make([]Field, len(l.fields)+len(fields)),
	}

	copy(newLogger.fields, l.fields)
	copy(newLogger.fields[len(l.fields):], fields)

	return newLogger
}

// Debug 记录调试日志
func (l *StructuredLogger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields...)
}

// Info 记录信息日志
func (l *StructuredLogger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields...)
}

// Warn 记录警告日志
func (l *StructuredLogger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields...)
}

// Error 记录错误日志
func (l *StructuredLogger) Error(msg string, fields ...Field) {
	l.log(ERROR, msg, fields...)
}

// log 内部日志记录方法
func (l *StructuredLogger) log(level LogLevel, msg string, fields ...Field) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if level < l.level {
		return
	}

	// 构建日志消息
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")
	levelStr := l.getLevelString(level)
	
	// 获取调用位置
	location := ""
	if l.location {
		_, file, line, ok := runtime.Caller(2)
		if ok {
			short := file
			for i := len(file) - 1; i > 0; i-- {
				if file[i] == '/' || file[i] == '\\' {
					short = file[i+1:]
					break
				}
			}
			location = fmt.Sprintf(" [%s:%d]", short, line)
		}
	}

	// 合并字段
	allFields := make([]Field, len(l.fields)+len(fields))
	copy(allFields, l.fields)
	copy(allFields[len(l.fields):], fields)

	// 格式化字段
	fieldsStr := ""
	if len(allFields) > 0 {
		fieldsStr = " {"
		for i, f := range allFields {
			if i > 0 {
				fieldsStr += ", "
			}
			fieldsStr += fmt.Sprintf("%s=%v", f.Key, f.Value)
		}
		fieldsStr += "}"
	}

	// 构建最终消息
	prefix := ""
	if l.prefix != "" {
		prefix = fmt.Sprintf("[%s] ", l.prefix)
	}

	logMsg := fmt.Sprintf("%s %s%s%s %s%s\n", timestamp, prefix, levelStr, location, msg, fieldsStr)

	// 输出日志
	fmt.Fprint(l.output, logMsg)
}

// getLevelString 获取级别字符串
func (l *StructuredLogger) getLevelString(level LogLevel) string {
	switch level {
	case DEBUG:
		if l.colored {
			return "\033[36m[DEBUG]\033[0m" // 青色
		}
		return "[DEBUG]"
	case INFO:
		if l.colored {
			return "\033[32m[INFO]\033[0m" // 绿色
		}
		return "[INFO]"
	case WARN:
		if l.colored {
			return "\033[33m[WARN]\033[0m" // 黄色
		}
		return "[WARN]"
	case ERROR:
		if l.colored {
			return "\033[31m[ERROR]\033[0m" // 红色
		}
		return "[ERROR]"
	default:
		return "[UNKNOWN]"
	}
}

// String 实现 Field 的字符串字段辅助函数
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int 实现 Field 的整数字段辅助函数
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Error 实现 Field 的错误字段辅助函数
func ErrorField(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Duration 实现 Field 的时间字段辅助函数
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// DefaultLogger 默认日志器
var DefaultLogger = NewLogger()

// SetDefaultLogger 设置默认日志器
func SetDefaultLogger(logger Logger) {
	DefaultLogger = logger
}