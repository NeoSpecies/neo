package discovery

import "encoding/json"

// IPCRequest IPC请求结构
type IPCRequest struct {
	Action  string   `json:"action"`  // register/deregister/list
	Service *Service `json:"service"` // 服务数据（register/deregister时使用）
	Name    string   `json:"name"`    // 服务名称（list时使用）
	ID      string   `json:"id"`      // 服务ID（deregister时使用）
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
