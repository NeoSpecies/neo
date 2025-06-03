package protocol

import (
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

// Encode serializes the message to bytes with optional compression
func (m *Message) Encode() ([]byte, error) {
	// Compress payload if compression is enabled
	var payload []byte
	var err error
	if m.Header.CompressionType != CompressNone {
		compressor := NewCompressor(m.Header.CompressionType)
		payload, err = compressor.Compress(m.Payload)
		if err != nil {
			return nil, err
		}
		m.Header.CompressedSize = uint32(len(payload))
	} else {
		payload = m.Payload
		m.Header.CompressedSize = m.Header.PayloadSize
	}

	// Allocate buffer for header and payload
	buffer := make([]byte, HeaderSize+len(payload))

	// Encode header
	buffer[0] = m.Header.Version                // Version (1 byte)
	buffer[1] = m.Header.Type                   // Type (1 byte)
	buffer[2] = uint8(m.Header.CompressionType) //
	binary.BigEndian.PutUint64(buffer[3:11], m.Header.RequestID)
	binary.BigEndian.PutUint32(buffer[11:15], m.Header.PayloadSize)
	binary.BigEndian.PutUint32(buffer[15:19], m.Header.CompressedSize)
	binary.BigEndian.PutUint64(buffer[19:27], uint64(m.Header.Timestamp)) // Timestamp (8 bytes, 替换 PutInt64)
	buffer[27] = m.Header.Priority
	binary.BigEndian.PutUint32(buffer[28:32], m.Header.Checksum)
	copy(buffer[32:48], m.Header.TraceID[:])
	buffer[48] = m.Header.RetryCount

	// Copy payload
	copy(buffer[HeaderSize:], payload)

	return buffer, nil
}

// Decode deserializes bytes to message with optional decompression
func Decode(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, ErrInvalidMessage
	}

	msg := &Message{
		Header: MessageHeader{
			Version:         data[0],
			Type:            data[1],
			CompressionType: CompressionType(data[2]),
			RequestID:       binary.BigEndian.Uint64(data[3:11]),
			PayloadSize:     binary.BigEndian.Uint32(data[11:15]),
			CompressedSize:  binary.BigEndian.Uint32(data[15:19]),
			Timestamp:       int64(binary.BigEndian.Uint64(data[19:27])), // 替换 Uint8 为直接读取
			Priority:        data[27],
			Checksum:        binary.BigEndian.Uint32(data[28:32]),
			RetryCount:      data[48],
		},
	}

	// Copy TraceID
	copy(msg.Header.TraceID[:], data[32:48])

	// Extract payload
	payload := data[HeaderSize:]

	// Verify checksum
	if msg.calculateChecksum() != msg.Header.Checksum {
		return nil, ErrChecksumMismatch
	}

	// Decompress if needed
	if msg.Header.CompressionType != CompressNone {
		compressor := NewCompressor(msg.Header.CompressionType)
		decompressed, err := compressor.Decompress(payload)
		if err != nil {
			return nil, err
		}
		msg.Payload = decompressed
	} else {
		msg.Payload = payload
	}

	return msg, nil
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
