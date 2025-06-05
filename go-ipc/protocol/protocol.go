package protocol

import (
	"bytes"
	"encoding/binary"
	"errors" // 添加 errors 包导入
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
	Magic         uint32 
	Version       uint8  
	MsgIDLen      uint16 
	MethodNameLen uint16
	ParamLen      uint32 
	// 新增回调标识字段
	CallbackFlag  uint8  // 0:无回调 1:需要回调
	CallbackIDLen uint16 // 回调ID长度
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

// IsHeartbeatResponse 判断是否为心跳响应消息
func IsHeartbeatResponse(msg *Message) bool {
	return msg.Header.Type == TypeHeartbeat
}

// UnmarshalMessage 将字节数据反序列化为Message对象
func UnmarshalMessage(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, errors.New("invalid message data: too short")
	}

	msg := &Message{}
	// 解析头部
	msg.Header.Version = data[0]
	msg.Header.Type = data[1]
	msg.Header.CompressionType = CompressionType(data[2])
	msg.Header.RequestID = binary.BigEndian.Uint64(data[3:11])
	msg.Header.PayloadSize = binary.BigEndian.Uint32(data[11:15])
	msg.Header.CompressedSize = binary.BigEndian.Uint32(data[15:19])
	msg.Header.Timestamp = int64(binary.BigEndian.Uint64(data[19:27]))
	msg.Header.Priority = data[27]
	msg.Header.Checksum = binary.BigEndian.Uint32(data[28:32])
	copy(msg.Header.TraceID[:], data[32:48])
	msg.Header.RetryCount = data[48]

	// 解析负载
	if len(data) > HeaderSize {
		msg.Payload = data[HeaderSize:]
	}

	// 校验和验证
	if msg.calculateChecksum() != msg.Header.Checksum {
		return nil, errors.New("checksum mismatch")
	}

	return msg, nil
}

// Bytes 返回消息的完整字节表示（头部+负载）
func (m *Message) Bytes() []byte {
	headerBytes := make([]byte, HeaderSize)
	headerBytes[0] = m.Header.Version
	headerBytes[1] = m.Header.Type
	headerBytes[2] = uint8(m.Header.CompressionType)
	binary.BigEndian.PutUint64(headerBytes[3:11], m.Header.RequestID)
	binary.BigEndian.PutUint32(headerBytes[11:15], m.Header.PayloadSize)
	binary.BigEndian.PutUint32(headerBytes[15:19], m.Header.CompressedSize)
	binary.BigEndian.PutUint64(headerBytes[19:27], uint64(m.Header.Timestamp))
	headerBytes[27] = m.Header.Priority
	binary.BigEndian.PutUint32(headerBytes[28:32], m.Header.Checksum)
	copy(headerBytes[32:48], m.Header.TraceID[:])
	headerBytes[48] = m.Header.RetryCount
	return append(headerBytes, m.Payload...)
}
