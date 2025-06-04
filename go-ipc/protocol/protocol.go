package protocol

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"time"

	"github.com/google/uuid"
)

const (
	// Protocol versions
	ProtocolVersion1 = 1

	// Message types
	TypeRequest   = 1
	TypeResponse  = 2
	TypeHeartbeat = 3
	TypeError     = 4

	// Max message size
	MaxMessageSize = 1024 * 1024 * 10 // 10MB

	// Header size
	HeaderSize = 32 // Fixed header size
)

// MessageHeader represents the protocol header
type MessageHeader struct {
	Version         uint8           // Protocol version
	Type            uint8           // Message type
	CompressionType CompressionType // Compression algorithm
	RequestID       uint64          // Unique request ID
	PayloadSize     uint32          // Original payload size
	CompressedSize  uint32          // Compressed payload size
	Timestamp       int64           // Message timestamp
	Priority        uint8           // Message priority
	Checksum        uint32          // CRC32 checksum
	TraceID         [16]byte        // UUID for tracing
	RetryCount      uint8           // Retry count
}

// Message represents a complete protocol message
type Message struct {
	Header  MessageHeader
	Payload []byte
}

// NewMessage creates a new protocol message
func NewMessage(msgType uint8, payload []byte) *Message {
	traceID, _ := uuid.New().MarshalBinary()
	var traceBytes [16]byte
	copy(traceBytes[:], traceID)

	msg := &Message{
		Header: MessageHeader{
			Version:     ProtocolVersion1,
			Type:        msgType,
			RequestID:   generateRequestID(),
			PayloadSize: uint32(len(payload)),
			Timestamp:   time.Now().UnixNano(),
			Priority:    0,
			TraceID:     traceBytes,
			RetryCount:  0,
		},
		Payload: payload,
	}

	// Calculate checksum
	msg.Header.Checksum = msg.calculateChecksum()

	return msg
}

// 定义协议头结构体（替代手动字节解析）
type ProtocolHeader struct {
	Magic         uint32 // 魔数（固定值）
	Version       uint8  // 协议版本
	MsgIDLen      uint16 // 消息ID长度
	MethodNameLen uint16 // 方法名长度
	ParamLen      uint32 // 参数内容长度
	FileCount     uint8  // 文件数量（扩展字段）
	// 预留扩展字段（如压缩标识、追踪ID）
	CompressionAlg uint8  // 压缩算法标识（0:无，1:gzip，2:zstd）
	TraceIDLen     uint16 // 追踪ID长度（若enable_tracing=on）
}

// 编码协议头（替代手动写字节）
func EncodeHeader(header ProtocolHeader) ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, header) // 自动序列化结构体
	return buf.Bytes(), err
}

// 解码协议头（替代手动读字节）
func DecodeHeader(data []byte) (ProtocolHeader, error) {
	var header ProtocolHeader
	err := binary.Read(bytes.NewReader(data), binary.BigEndian, &header) // 自动反序列化
	return header, err
}

// calculateChecksum calculates CRC32 checksum of the message
func (m *Message) calculateChecksum() uint32 {
	// Create a copy of the header with checksum field zeroed
	headerCopy := m.Header
	// headerCopy.Checksum = 0

	// Convert header to bytes（修复：单字节直接赋值）
	headerBytes := make([]byte, HeaderSize)
	headerBytes[0] = headerCopy.Version
	headerBytes[1] = headerCopy.Type
	headerBytes[2] = uint8(headerCopy.CompressionType)
	binary.BigEndian.PutUint64(headerBytes[3:11], headerCopy.RequestID)
	binary.BigEndian.PutUint32(headerBytes[11:15], headerCopy.PayloadSize)
	binary.BigEndian.PutUint32(headerBytes[15:19], headerCopy.CompressedSize)
	binary.BigEndian.PutUint64(headerBytes[19:27], uint64(headerCopy.Timestamp))
	headerBytes[27] = headerCopy.Priority
	binary.BigEndian.PutUint32(headerBytes[28:32], 0) // Checksum field (zeroed)
	copy(headerBytes[32:48], headerCopy.TraceID[:])
	headerBytes[48] = headerCopy.RetryCount

	// Calculate checksum of header and payload
	checksum := crc32.NewIEEE()
	checksum.Write(headerBytes)
	checksum.Write(m.Payload)
	return checksum.Sum32()
}

// generateRequestID generates a unique request ID
func generateRequestID() uint64 {
	return uint64(time.Now().UnixNano())
}

// GetTraceID returns the trace ID as UUID
func (m *Message) GetTraceID() uuid.UUID {
	var id uuid.UUID
	copy(id[:], m.Header.TraceID[:])
	return id
}

// SetCompression sets the compression type for the message
func (m *Message) SetCompression(typ CompressionType) {
	m.Header.CompressionType = typ
}
