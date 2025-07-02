package ipcprotocol

import (
	"neo/internal/types"
)

// 创建压缩器实例
func NewCompressor(typ types.CompressionType) types.Compressor {
	switch typ {
	case types.CompressGzip:
		return &types.GzipCompressor{}
	case types.CompressZstd:
		return &types.ZstdCompressor{}
	case types.CompressLZ4:
		return &types.LZ4Compressor{}
	default:
		return &types.NoCompressor{}
	}
}
