package tcp

import (
	"fmt"
	"io"
	"neo/internal/types"
)

// 类型别名迁移（保留兼容）
// Deprecated: 请使用 types.ConnectionError
type ConnectionError = types.ConnectionError

// Deprecated: 请使用 types.Codec
type Codec = types.Codec

// 修改NewCodec函数，显式传递协议配置
// 修改NewCodec函数，仅传递reader和writer参数
func NewCodec(reader io.Reader, writer io.Writer) *types.Codec {
	// 只传递reader和writer，协议常量在types包中定义或通过其他方式配置
	return types.NewCodec(reader, writer)
}

// 添加魔数验证函数（用于调试）
func ValidateMagicNumber(magic uint16) error {
	if magic != MAGIC_NUMBER {
		return fmt.Errorf("魔数不匹配: 0x%04X (期望: 0x%04X)", magic, MAGIC_NUMBER)
	}
	return nil
}

// 写入IPC消息
// 添加协议常量定义（确保使用固定大小类型）
// 协议常量定义（如果types包中未定义，需确保这些常量在Codec使用时被正确引用）
// 修复：移除嵌套的const声明
const (
    MAGIC_NUMBER uint16 = 0xAEBD  // 2字节大端魔数
    VERSION      uint8  = 0x01    // 1字节协议版本
    MaxFrameSize uint32 = 1 << 20 // 1MB 最大帧大小限制
)
