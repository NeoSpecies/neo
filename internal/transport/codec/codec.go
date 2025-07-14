package codec

import (
	"context"
	"neo/internal/protocol"
	"neo/internal/types"
)

// Codec 调整接口参数类型为 types.Message，与 protocol 包对齐
type Codec interface {
	Encode(ctx context.Context, msg types.Message) ([]byte, error)
	Decode(ctx context.Context, data []byte) (types.Message, error)
}

// NewCodec 使用 protocol.NewCodec 获取具体编解码器实例
func NewCodec(protocolType string) Codec {
	return protocol.NewCodec(protocolType)
}
