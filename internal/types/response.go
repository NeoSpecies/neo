package types

import "time"

// Response 表示服务间通信的响应结构
// 包含状态码、正文和错误信息，支持异步响应匹配
type Response struct {
	ID       string            `json:"id"`                 // 响应ID，与请求ID对应
	Status   int               `json:"status"`             // 状态码（如 HTTP 的 200/404）
	Body     []byte            `json:"body"`               // 响应正文内容
	Error    string            `json:"error,omitempty"`    // 错误信息（仅当状态码非成功时有效）
	Metadata map[string]string `json:"metadata,omitempty"` // 元数据（如响应时间、服务器信息等）
}

// NewResponse 创建新响应
func NewResponse(requestID string, status int, body []byte) *Response {
	return &Response{
		ID:       requestID,
		Status:   status,
		Body:     body,
		Metadata: make(map[string]string),
	}
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(requestID string, status int, err string) *Response {
	return &Response{
		ID:       requestID,
		Status:   status,
		Error:    err,
		Metadata: make(map[string]string),
	}
}

// IsSuccess 判断响应是否成功
func (r *Response) IsSuccess() bool {
	return r.Status >= 200 && r.Status < 300
}

// Validate 验证响应结构
func (r *Response) Validate() error {
	if r.ID == "" {
		return ErrInvalidResponseID
	}
	if r.Status == 0 {
		return ErrInvalidStatus
	}
	return nil
}

// ToMessage 转换为消息结构
func (r *Response) ToMessage() *Message {
	return &Message{
		ID:        r.ID,
		Type:      RESPONSE,
		Metadata:  r.Metadata,
		Body:      r.Body,
		Timestamp: time.Now(),
	}
}