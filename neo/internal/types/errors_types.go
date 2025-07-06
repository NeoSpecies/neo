/*
 * 描述: 定义系统中各类错误类型和错误常量，包括连接错误结构体和通用错误定义
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"errors"
	"fmt"
)

// 错误类型定义常量
// 用于标识不同类型的错误，便于错误分类和处理
// +---------------------------+-----------------------------------+
// | 常量名                    | 描述                              |
// +---------------------------+-----------------------------------+
// | ErrorTypeReadFailed       | 读取操作失败                      |
// | ErrorTypeWriteFailed      | 写入操作失败                      |
// | ErrorTypeInvalidData      | 数据格式无效                      |
// | ErrorTypeConnection       | 连接相关错误                      |
// | ErrorTypeConnectionClosed | 连接已关闭                        |
// +---------------------------+-----------------------------------+
const (
	ErrorTypeReadFailed       = "read_failed"
	ErrorTypeWriteFailed      = "write_failed"
	ErrorTypeInvalidData      = "invalid_data"
	ErrorTypeConnection       = "connection_error"
	ErrorTypeConnectionClosed = "connection_closed"
)

// ConnectionError 连接错误结构体
// 封装连接相关的错误信息，包括错误类型、描述和底层错误
// +----------------+------------------+-----------------------------------+
// | 字段名         | 类型             | 描述                              |
// +----------------+------------------+-----------------------------------+
// | Type           | string           | 错误类型（如read_failed）         |
// | Message        | string           | 错误描述信息                      |
// | Err            | error            | 底层错误对象                      |
// +----------------+------------------+-----------------------------------+
type ConnectionError struct {
	Type    string
	Message string
	Err     error
}

// Error 实现error接口
// 返回格式化的错误信息，包含错误类型、描述和底层错误（如有）
// +----------------+-----------------------------------+
// | 返回值         | 描述                              |
// +----------------+-----------------------------------+
// | string         | 格式化的错误字符串                |
// +----------------+-----------------------------------+
func (e *ConnectionError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// 通用错误变量定义
// 系统中常用的错误实例，避免重复创建相同错误
// +---------------------------+-----------------------------------+
// | 变量名                    | 描述                              |
// +---------------------------+-----------------------------------+
// | ErrInvalidMessage         | 消息格式无效                      |
// | ErrChecksumMismatch       | 校验和不匹配                      |
// | ErrMessageTooLarge        | 消息大小超出限制                  |
// | ErrCompressionFailed      | 压缩操作失败                      |
// | ErrDecompressionFailed    | 解压缩操作失败                    |
// | ErrInvalidCompression     | 无效的压缩类型                    |
// | ErrMaxRetryExceeded       | 超出最大重试次数                  |
// | ErrNetworkError           | 网络错误                          |
// | ErrBusinessError          | 业务逻辑错误                      |
// | ErrTimeout                | 请求超时                          |
// | ErrCircuitBreakerOpen     | 熔断器已打开                      |
// +---------------------------+-----------------------------------+
var (
	ErrInvalidMessage      = errors.New("invalid message format")
	ErrChecksumMismatch    = errors.New("checksum mismatch")
	ErrMessageTooLarge     = errors.New("message size exceeds limit")
	ErrCompressionFailed   = errors.New("compression failed")
	ErrDecompressionFailed = errors.New("decompression failed")
	ErrInvalidCompression  = errors.New("invalid compression type")
	ErrMaxRetryExceeded    = errors.New("max retry count exceeded")
	ErrNetworkError        = errors.New("network error")
	ErrBusinessError       = errors.New("business logic error")
	ErrTimeout             = errors.New("request timeout")
	ErrCircuitBreakerOpen  = errors.New("circuit breaker is open")
)
