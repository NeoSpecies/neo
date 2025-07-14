package protocol

import (
	"fmt"
	"neo/internal/types"
)

// Codec 编解码器接口
type Codec interface {
	Encode(msg types.Message) ([]byte, error)
	Decode(data []byte) (types.Message, error)
	Version() string
}

// ProtocolFactory 协议工厂
func NewCodec(protocol string) (Codec, error) {
	switch protocol {
	case "http":
		return NewHTTPCodec(), nil
	case "ipc":
		return NewIPCCodec(), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}