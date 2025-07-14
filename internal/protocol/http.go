package protocol

import (
	"encoding/json"
	"fmt"
	"neo/internal/types"
	"strconv"
	"time"
)

const (
	// HTTP 协议版本
	HTTPVersion = "1.0"
)

// HTTPCodec HTTP/JSON协议编解码器
type HTTPCodec struct{}

// NewHTTPCodec 创建HTTP编解码器
func NewHTTPCodec() *HTTPCodec {
	return &HTTPCodec{}
}

// Version 返回协议版本
func (c *HTTPCodec) Version() string {
	return HTTPVersion
}

// HTTPMessage HTTP消息的JSON表示
type HTTPMessage struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"`
	Service   string            `json:"service"`
	Method    string            `json:"method"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Body      json.RawMessage   `json:"body"`
	Timestamp string            `json:"timestamp"`
}

// Encode 编码消息为JSON格式
func (c *HTTPCodec) Encode(msg types.Message) ([]byte, error) {
	// 转换Body为JSON
	var bodyJSON json.RawMessage
	if len(msg.Body) > 0 {
		// 检查Body是否已经是有效的JSON
		if json.Valid(msg.Body) {
			bodyJSON = json.RawMessage(msg.Body)
		} else {
			// 如果不是JSON，尝试将其作为字符串编码
			bodyBytes, err := json.Marshal(string(msg.Body))
			if err != nil {
				return nil, fmt.Errorf("failed to encode body as JSON: %w", err)
			}
			bodyJSON = json.RawMessage(bodyBytes)
		}
	} else {
		bodyJSON = json.RawMessage("null")
	}
	
	httpMsg := HTTPMessage{
		ID:        msg.ID,
		Type:      messageTypeToString(msg.Type),
		Service:   msg.Service,
		Method:    msg.Method,
		Metadata:  msg.Metadata,
		Body:      bodyJSON,
		Timestamp: msg.Timestamp.Format(time.RFC3339Nano),
	}
	
	data, err := json.Marshal(httpMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal HTTP message: %w", err)
	}
	
	return data, nil
}

// Decode 解码JSON数据为消息
func (c *HTTPCodec) Decode(data []byte) (types.Message, error) {
	var httpMsg HTTPMessage
	if err := json.Unmarshal(data, &httpMsg); err != nil {
		return types.Message{}, fmt.Errorf("failed to unmarshal HTTP message: %w", err)
	}
	
	// 解析时间戳
	timestamp, err := time.Parse(time.RFC3339Nano, httpMsg.Timestamp)
	if err != nil {
		// 如果解析失败，使用当前时间
		timestamp = time.Now()
	}
	
	// 将Body从JSON转换回字节数组
	var body []byte
	if len(httpMsg.Body) > 0 && string(httpMsg.Body) != "null" {
		body = []byte(httpMsg.Body)
	}
	
	return types.Message{
		ID:        httpMsg.ID,
		Type:      stringToMessageType(httpMsg.Type),
		Service:   httpMsg.Service,
		Method:    httpMsg.Method,
		Metadata:  httpMsg.Metadata,
		Body:      body,
		Timestamp: timestamp,
	}, nil
}

// messageTypeToString 将消息类型转换为字符串
func messageTypeToString(t types.MessageType) string {
	switch t {
	case types.REQUEST:
		return "REQUEST"
	case types.RESPONSE:
		return "RESPONSE"
	case types.REGISTER:
		return "REGISTER"
	case types.HEARTBEAT:
		return "HEARTBEAT"
	default:
		return strconv.Itoa(int(t))
	}
}

// stringToMessageType 将字符串转换为消息类型
func stringToMessageType(s string) types.MessageType {
	switch s {
	case "REQUEST":
		return types.REQUEST
	case "RESPONSE":
		return types.RESPONSE
	case "REGISTER":
		return types.REGISTER
	case "HEARTBEAT":
		return types.HEARTBEAT
	default:
		// 尝试解析为数字
		if i, err := strconv.Atoi(s); err == nil {
			return types.MessageType(i)
		}
		return types.REQUEST // 默认值
	}
}