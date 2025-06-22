package ipcprotocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"

	"github.com/google/uuid"
)

const (
	// Protocol versions
	ProtocolVersion1 = 1

	// Message types - 客户端不使用此字段，暂时保留但不序列化
	TypeRequest   = 1
	TypeResponse  = 2
	TypeHeartbeat = 3
	TypeError     = 4

	// Max message size
	MaxMessageSize = 1024 * 1024 * 10 // 10MB

	// 协议常量与test.py完全一致
	MAGIC_NUMBER = 0xAEBD // 2字节大端魔数
	VERSION      = 0x01   // 1字节协议版本
	HeaderSize   = 15     // 修正为15字节 (2+1+2+2+4+4)
)

// MessageHeader 修正为与test.py完全一致的协议头部
type MessageHeader struct {
	Magic     uint16 // 协议魔数 (0xAEBD)
	Version   uint8  // 协议版本
	MsgIDLen  uint16 // 消息ID长度
	MethodLen uint16 // 方法名长度
	ParamLen  uint32 // 参数长度
	Checksum  uint32 // CRC32校验和
}

// Message represents a complete protocol message
type Message struct {
	Header  MessageHeader
	Payload []byte // 包含: 消息ID内容 + 方法名内容 + 参数内容
}

// NewMessage 创建符合test.py协议的消息
func NewMessage(msgType uint8, method string, payload []byte) *Message {
	// 生成UUID作为消息ID
	msgID := uuid.New().String()
	msgIDBytes := []byte(msgID)
	msgIDLen := uint16(len(msgIDBytes))

	// 方法名字节
	methodBytes := []byte(method)
	methodLen := uint16(len(methodBytes))

	// 参数长度
	paramLen := uint32(len(payload))

	// 构建完整Payload (消息ID内容 + 方法名内容 + 参数内容)
	// 修复：将uint16转换为uint32后再相加
	fullPayload := make([]byte, 0, uint32(msgIDLen)+uint32(methodLen)+paramLen)
	fullPayload = append(fullPayload, msgIDBytes...)
	fullPayload = append(fullPayload, methodBytes...)
	fullPayload = append(fullPayload, payload...)

	// 构建用于计算校验和的数据
	checksumData := make([]byte, 0)
	// 1. 魔数
	magicBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(magicBytes, MAGIC_NUMBER)
	checksumData = append(checksumData, magicBytes...)
	// 2. 版本
	checksumData = append(checksumData, VERSION)
	// 3. 消息ID长度
	msgIDLenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(msgIDLenBytes, msgIDLen)
	checksumData = append(checksumData, msgIDLenBytes...)
	// 4. 消息ID内容
	checksumData = append(checksumData, msgIDBytes...)
	// 5. 方法名长度
	methodLenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(methodLenBytes, methodLen)
	checksumData = append(checksumData, methodLenBytes...)
	// 6. 方法名内容
	checksumData = append(checksumData, methodBytes...)
	// 7. 参数长度
	paramLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(paramLenBytes, paramLen)
	checksumData = append(checksumData, paramLenBytes...)
	// 8. 参数内容
	checksumData = append(checksumData, payload...)

	// 计算校验和
	checksum := crc32.ChecksumIEEE(checksumData)

	return &Message{
		Header: MessageHeader{
			Magic:     MAGIC_NUMBER,
			Version:   VERSION,
			MsgIDLen:  msgIDLen,
			MethodLen: methodLen,
			ParamLen:  paramLen,
			Checksum:  checksum,
		},
		Payload: fullPayload,
	}
}

// Bytes 返回完整的协议消息字节流
func (m *Message) Bytes() []byte {
	headerBytes := make([]byte, HeaderSize)

	// 魔数 (2字节)
	binary.BigEndian.PutUint16(headerBytes[0:2], m.Header.Magic)
	// 版本 (1字节)
	headerBytes[2] = m.Header.Version
	// 消息ID长度 (2字节)
	binary.BigEndian.PutUint16(headerBytes[3:5], m.Header.MsgIDLen)
	// 方法名长度 (2字节)
	binary.BigEndian.PutUint16(headerBytes[5:7], m.Header.MethodLen)
	// 参数长度 (4字节)
	binary.BigEndian.PutUint32(headerBytes[7:11], m.Header.ParamLen)
	// 校验和 (4字节)
	binary.BigEndian.PutUint32(headerBytes[11:15], m.Header.Checksum)

	// 拼接头部和负载
	return append(headerBytes, m.Payload...)
}

// UnmarshalMessage 从字节流解析消息
func UnmarshalMessage(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, errors.New("invalid message data: too short")
	}

	msg := &Message{}
	msg.Header.Magic = binary.BigEndian.Uint16(data[0:2])
	msg.Header.Version = data[2]
	msg.Header.MsgIDLen = binary.BigEndian.Uint16(data[3:5])
	msg.Header.MethodLen = binary.BigEndian.Uint16(data[5:7])
	msg.Header.ParamLen = binary.BigEndian.Uint32(data[7:11])
	msg.Header.Checksum = binary.BigEndian.Uint32(data[11:15])

	// 提取负载数据
	if len(data) > HeaderSize {
		msg.Payload = data[HeaderSize:]
	}

	// 验证校验和
	if msg.CalculateChecksum() != msg.Header.Checksum {
		return nil, errors.New("checksum mismatch")
	}

	return msg, nil
}

// CalculateChecksum 计算校验和（与test.py完全一致）
func (m *Message) CalculateChecksum() uint32 {
	checksumData := make([]byte, 0)

	// 1. 魔数
	magicBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(magicBytes, m.Header.Magic)
	checksumData = append(checksumData, magicBytes...)

	// 2. 版本
	checksumData = append(checksumData, m.Header.Version)

	// 3. 消息ID长度
	msgIDLenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(msgIDLenBytes, m.Header.MsgIDLen)
	checksumData = append(checksumData, msgIDLenBytes...)

	// 4. 消息ID内容
	if len(m.Payload) >= int(m.Header.MsgIDLen) {
		checksumData = append(checksumData, m.Payload[:m.Header.MsgIDLen]...)
	} else {
		return 0
	}

	// 5. 方法名长度
	methodLenBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(methodLenBytes, m.Header.MethodLen)
	checksumData = append(checksumData, methodLenBytes...)

	// 6. 方法名内容
	methodStart := m.Header.MsgIDLen
	methodEnd := methodStart + m.Header.MethodLen
	if len(m.Payload) >= int(methodEnd) {
		checksumData = append(checksumData, m.Payload[methodStart:methodEnd]...)
	} else {
		return 0
	}

	// 7. 参数长度
	paramLenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(paramLenBytes, m.Header.ParamLen)
	checksumData = append(checksumData, paramLenBytes...)

	// 8. 参数内容
	paramStart := methodEnd
	paramEnd := paramStart + uint16(m.Header.ParamLen)
	if len(m.Payload) >= int(paramEnd) {
		checksumData = append(checksumData, m.Payload[paramStart:paramEnd]...)
	} else {
		return 0
	}

	return crc32.ChecksumIEEE(checksumData)
}

// IsHeartbeatResponse 判断是否为心跳响应消息
func IsHeartbeatResponse(msg *Message) bool {
	// 从负载中提取方法名，判断是否为心跳请求
	if msg.Header.MethodLen > 0 && len(msg.Payload) >= int(msg.Header.MsgIDLen+msg.Header.MethodLen) {
		method := string(msg.Payload[msg.Header.MsgIDLen : msg.Header.MsgIDLen+msg.Header.MethodLen])
		return method == "heartbeat"
	}
	return false
}

// NewResponse 创建客户端期望格式的响应消息
func NewResponse(body []byte) []byte {
	// 修复：正确计算缓冲区大小(2+1+4=7字节头部 + 消息体长度)
	buffer := make([]byte, 7+len(body))

	// 1. 魔数(2字节)
	binary.BigEndian.PutUint16(buffer[0:2], MAGIC_NUMBER)
	// 2. 版本(1字节)
	buffer[2] = VERSION
	// 3. 响应体长度(4字节) - 修复缓冲区越界
	binary.BigEndian.PutUint32(buffer[3:7], uint32(len(body)))
	// 4. 响应体内容
	copy(buffer[7:], body)

	return buffer
}

// ValidateResponseFormat 验证响应格式是否符合规范
func ValidateResponseFormat(response []byte) error {
	if len(response) < 7 {
		return errors.New("响应长度不足7字节头部")
	}

	// 验证魔数
	magic := binary.BigEndian.Uint16(response[0:2])
	if magic != MAGIC_NUMBER {
		return fmt.Errorf("响应魔数不匹配, 期望%#x, 实际%#x", MAGIC_NUMBER, magic)
	}

	// 验证版本
	version := response[2]
	if version != VERSION {
		return fmt.Errorf("响应版本不匹配, 期望%d, 实际%d", VERSION, version)
	}

	// 验证长度字段
	bodyLen := binary.BigEndian.Uint32(response[3:7])
	if uint32(len(response)-7) != bodyLen {
		return fmt.Errorf("响应长度字段不匹配, 头部声明%d字节, 实际%d字节", bodyLen, len(response)-7)
	}

	return nil
}
