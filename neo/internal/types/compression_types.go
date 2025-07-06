/*
 * 描述: 定义压缩算法相关的类型和接口，支持Gzip、Zstd、LZ4等多种压缩方式
 * 作者: Cogito
 * 日期: 2025-06-18
 * 联系方式: neospecies@outlook.com
 */
package types

import (
	"bytes"
	"compress/gzip"

	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4"
)

// CompressionType 压缩算法类型枚举
// 定义系统支持的压缩算法类型常量
// +----------------+-----------------------------------+
// | 常量名         | 描述                              |
// +----------------+-----------------------------------+
// | CompressNone   | 不使用压缩                        |
// | CompressGzip   | Gzip压缩算法                      |
// | CompressZstd   | Zstd压缩算法                      |
// | CompressLZ4    | LZ4压缩算法                       |
// +----------------+-----------------------------------+
type CompressionType uint8

const (
	CompressNone CompressionType = iota
	CompressGzip
	CompressZstd
	CompressLZ4
)

// Compressor 压缩器接口
// 定义压缩与解压缩操作的标准接口
// +----------------+-----------------------------------+
// | 方法名         | 描述                              |
// +----------------+-----------------------------------+
// | Compress       | 对输入数据进行压缩处理            |
// | Decompress     | 对压缩数据进行解压缩处理          |
// +----------------+-----------------------------------+
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}

// NoCompressor 无压缩实现
// 实现Compressor接口的空压缩器，不执行任何压缩操作
// 用于不需要压缩的场景或作为默认实现

type NoCompressor struct{}

// Compress 不压缩直接返回原始数据
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 原始数据字节切片                  |
// | 返回值         | 未压缩的原始数据和nil错误         |
// +----------------+-----------------------------------+
func (c *NoCompressor) Compress(data []byte) ([]byte, error) {
	return data, nil
}

// Decompress 不解压直接返回原始数据
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 原始数据字节切片                  |
// | 返回值         | 未解压的原始数据和nil错误         |
// +----------------+-----------------------------------+
func (c *NoCompressor) Decompress(data []byte) ([]byte, error) {
	return data, nil
}

// GzipCompressor Gzip压缩实现
// 使用标准库实现的Gzip压缩算法
// 适用于需要广泛兼容性的场景

type GzipCompressor struct{}

// Compress 使用Gzip算法压缩数据
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 要压缩的原始数据                  |
// | 返回值         | 压缩后的数据和可能的错误          |
// +----------------+-----------------------------------+
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

// Decompress 使用Gzip算法解压缩数据
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 要解压的Gzip压缩数据              |
// | 返回值         | 解压后的原始数据和可能的错误      |
// +----------------+-----------------------------------+
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
// 使用klauspost/compress库实现的Zstd压缩算法
// 提供高压缩比和快速压缩速度

type ZstdCompressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

// Compress 使用Zstd算法压缩数据
// 延迟初始化编码器以提高性能
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 要压缩的原始数据                  |
// | 返回值         | 压缩后的数据和可能的错误          |
// +----------------+-----------------------------------+
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

// Decompress 使用Zstd算法解压缩数据
// 延迟初始化解码器以提高性能
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 要解压的Zstd压缩数据              |
// | 返回值         | 解压后的原始数据和可能的错误      |
// +----------------+-----------------------------------+
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
// 使用pierrec/lz4库实现的LZ4压缩算法
// 提供超高速压缩和解压缩性能

type LZ4Compressor struct{}

// Compress 使用LZ4算法压缩数据
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 要压缩的原始数据                  |
// | 返回值         | 压缩后的数据和可能的错误          |
// +----------------+-----------------------------------+
func (c *LZ4Compressor) Compress(data []byte) ([]byte, error) {
	buf := make([]byte, lz4.CompressBlockBound(len(data)))
	n, err := lz4.CompressBlock(data, buf, nil)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// Decompress 使用LZ4算法解压缩数据
// +----------------+-----------------------------------+
// | 参数           | 描述                              |
// +----------------+-----------------------------------+
// | data           | 要解压的LZ4压缩数据               |
// | 返回值         | 解压后的原始数据和可能的错误      |
// +----------------+-----------------------------------+
func (c *LZ4Compressor) Decompress(data []byte) ([]byte, error) {
	buf := make([]byte, len(data)*3) // 预估解压后大小
	n, err := lz4.UncompressBlock(data, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}
