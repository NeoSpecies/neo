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
func NewCodec(protocolType string) (Codec, error) {
	codec, err := protocol.NewCodec(protocolType)
	if err != nil {
		return nil, err
	}
	// 包装为兼容的 Codec 接口
	return &codecWrapper{codec: codec}, nil
}

// codecWrapper 包装 protocol.Codec 为本地 Codec 接口
type codecWrapper struct {
	codec protocol.Codec
}

func (c *codecWrapper) Encode(ctx context.Context, msg types.Message) ([]byte, error) {
	return c.codec.Encode(msg)
}

func (c *codecWrapper) Decode(ctx context.Context, data []byte) (types.Message, error) {
	return c.codec.Decode(data)
}
