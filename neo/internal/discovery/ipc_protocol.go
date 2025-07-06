package discovery

import (
	"encoding/json"
	types "neo/internal/types"
)

// IPCRequest IPC请求结构
type IPCRequest struct {
	Type    string         `json:"type"` // 关键修复：添加JSON标签映射
	Action  string         `json:"action,omitempty"`
	Service *types.Service `json:"service,omitempty"`
	Name    string         `json:"name,omitempty"`
	ID      string         `json:"id,omitempty"`
}

// IPCResponse IPC响应结构
type IPCResponse struct {
	Success bool       `json:"success"`
	Data    []*Service `json:"data"`
	Error   string     `json:"error"`
}

// Marshal 序列化请求
func (r *IPCRequest) Marshal() ([]byte, error) {
	return json.Marshal(r)
}

// Unmarshal 反序列化响应
func (r *IPCResponse) Unmarshal(data []byte) error {
	return json.Unmarshal(data, r)
}
