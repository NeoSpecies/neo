package types

import "time"

// Request 表示服务间通信的请求结构
// 支持扩展元数据字段，用于异步请求响应匹配
type Request struct {
	ID       string            `json:"id"`                 // 请求唯一标识
	Service  string            `json:"service"`            // 目标服务名称
	Method   string            `json:"method"`             // 请求方法
	Body     []byte            `json:"body"`               // 请求正文内容
	Metadata map[string]string `json:"metadata,omitempty"` // 元数据（如认证信息、跟踪信息等）
	Timeout  time.Duration     `json:"-"` // 请求超时时间（不序列化）
}

// NewRequest 创建新请求
func NewRequest(service, method string, body []byte) *Request {
	return &Request{
		ID:       GenerateID(),
		Service:  service,
		Method:   method,
		Body:     body,
		Metadata: make(map[string]string),
	}
}

// SetTimeout 设置请求超时时间
func (r *Request) SetTimeout(timeout time.Duration) {
	r.Timeout = timeout
}

// GetTimeout 获取请求超时时间
func (r *Request) GetTimeout() time.Duration {
	if r.Timeout == 0 {
		return 30 * time.Second // 默认30秒超时
	}
	return r.Timeout
}

// Validate 验证请求结构
func (r *Request) Validate() error {
	if r.ID == "" {
		return ErrInvalidRequestID
	}
	if r.Service == "" {
		return ErrInvalidService
	}
	if r.Method == "" {
		return ErrInvalidMethod
	}
	return nil
}

// ToMessage 转换为消息结构
func (r *Request) ToMessage() *Message {
	return &Message{
		ID:        r.ID,
		Type:      REQUEST,
		Service:   r.Service,
		Method:    r.Method,
		Metadata:  r.Metadata,
		Body:      r.Body,
		Timestamp: time.Now(),
	}
}