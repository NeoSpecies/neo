package types

import "fmt"

// 错误类型定义
const (
    ErrorTypeReadFailed       = "read_failed"
    ErrorTypeWriteFailed      = "write_failed"
    ErrorTypeInvalidData      = "invalid_data"
    ErrorTypeConnection       = "connection_error"
    ErrorTypeConnectionClosed = "connection_closed"
)

// ConnectionError 连接错误结构体
type ConnectionError struct {
    Type    string
    Message string
    Err     error
}

// 实现error接口
func (e *ConnectionError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}