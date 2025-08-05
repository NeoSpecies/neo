package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"neo/internal/types"
	"strconv"
	"time"
	"unicode/utf8"
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
			// 如果不是JSON，检查是否为有效的UTF-8文本
			if utf8.Valid(msg.Body) {
				// 如果是文本，作为字符串编码
				bodyBytes, err := json.Marshal(string(msg.Body))
				if err != nil {
					return nil, fmt.Errorf("failed to encode body as JSON: %w", err)
				}
				bodyJSON = json.RawMessage(bodyBytes)
			} else {
				// 如果是二进制数据，使用base64编码
				encoded := base64.StdEncoding.EncodeToString(msg.Body)
				bodyBytes, err := json.Marshal(map[string]string{
					"_base64": encoded,
				})
				if err != nil {
					return nil, fmt.Errorf("failed to encode binary body: %w", err)
				}
				bodyJSON = json.RawMessage(bodyBytes)
			}
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
		// 首先检查是否为base64编码的二进制数据
		var base64Map map[string]string
		if err := json.Unmarshal(httpMsg.Body, &base64Map); err == nil {
			if encoded, ok := base64Map["_base64"]; ok {
				// 解码base64数据
				decoded, err := base64.StdEncoding.DecodeString(encoded)
				if err == nil {
					body = decoded
				} else {
					body = []byte(httpMsg.Body)
				}
			} else {
				body = []byte(httpMsg.Body)
			}
		} else {
			// 尝试将Body作为JSON字符串解码
			var str string
			if err := json.Unmarshal(httpMsg.Body, &str); err == nil {
				// 如果成功解码为字符串，使用解码后的值
				body = []byte(str)
			} else {
				// 否则直接使用原始字节
				body = []byte(httpMsg.Body)
			}
		}
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