package tcp

import (
	"io"
	"neo/internal/types"
)

// 类型别名迁移（保留兼容）
// Deprecated: 请使用 types.ConnectionError
type ConnectionError = types.ConnectionError

// Deprecated: 请使用 types.Codec
type Codec = types.Codec

func NewCodec(reader io.Reader, writer io.Writer) *types.Codec {
	return types.NewCodec(reader, writer)
}

// 写入IPC消息
// 添加协议常量定义（确保使用固定大小类型）
const (
	MAGIC_NUMBER uint16 = 0xAEBD  // 2字节大端魔数
	VERSION      uint8  = 0x01    // 1字节协议版本
	MaxFrameSize uint32 = 1 << 20 // 1MB 最大帧大小限制
)
