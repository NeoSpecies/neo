package protocol

import (
	"bytes"
	"compress/gzip"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4"
)

// CompressionType 定义压缩算法类型
type CompressionType uint8

const (
	CompressNone CompressionType = iota
	CompressGzip
	CompressZstd
	CompressLZ4
)

// Compressor 压缩器接口
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

// 创建压缩器实例
func NewCompressor(typ CompressionType) Compressor {
	switch typ {
	case CompressGzip:
		return &GzipCompressor{}
	case CompressZstd:
		return &ZstdCompressor{}
	case CompressLZ4:
		return &LZ4Compressor{}
	default:
		return &NoCompressor{}
	}
}

// NoCompressor 不进行压缩
type NoCompressor struct{}

func (c *NoCompressor) Compress(data []byte) ([]byte, error) {
	return data, nil
}

func (c *NoCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}

// GzipCompressor Gzip压缩实现
type GzipCompressor struct{}

func (c *GzipCompressor) Compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c *GzipCompressor) Decompress(data []byte) ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ZstdCompressor Zstd压缩实现
type ZstdCompressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

func (c *ZstdCompressor) Compress(data []byte) ([]byte, error) {
	if c.encoder == nil {
		var err error
		c.encoder, err = zstd.NewWriter(nil)
		if err != nil {
			return nil, err
		}
	}
	return c.encoder.EncodeAll(data, nil), nil
}

func (c *ZstdCompressor) Decompress(data []byte) ([]byte, error) {
	if c.decoder == nil {
		var err error
		c.decoder, err = zstd.NewReader(nil)
		if err != nil {
			return nil, err
		}
	}
	return c.decoder.DecodeAll(data, nil)
}

// LZ4Compressor LZ4压缩实现
type LZ4Compressor struct{}

func (c *LZ4Compressor) Compress(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, buf, nil)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func (c *LZ4Compressor) Decompress(data []byte) ([]byte, error) {
	buf := make([]byte, len(data)*3) // 预估解压后大小
	n, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
} 