package types

// 压缩算法类型
type CompressionType uint8

const (
	CompressNone CompressionType = iota
	CompressGzip
	CompressZstd
	CompressLZ4
)

// 压缩器接口
type Compressor interface {
	Compress(data []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}
