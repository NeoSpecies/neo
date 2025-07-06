package http

import (
	"net/http"
)

// 新增: 内部响应接口
type ResponseWriter interface {
	WriteHeader(statusCode int)
	Write(data []byte) (int, error)
	Header() http.Header
}

// 新增: 内部请求结构
type Request struct {
	*http.Request
}

// 新增: 内部处理器类型
type HandlerFunc func(w ResponseWriter, r *Request)

// Router HTTP路由管理器
type Router struct {
	// 修改: 使用内部处理器类型
	handlers map[string]HandlerFunc
}

// NewRouter 创建新的路由实例
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]HandlerFunc),
	}
}

// Handle 注册HTTP处理函数
// 修改: 接受内部处理器类型
func (r *Router) Handle(pattern string, handler HandlerFunc) {
	r.handlers[pattern] = handler
}

// ServeHTTP 实现http.Handler接口
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if handler, ok := r.handlers[req.URL.Path]; ok {
		// 适配标准库类型到内部接口
		internalW := &responseWriterAdapter{w}
		internalReq := &Request{req}
		handler(internalW, internalReq)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}

// 新增: 响应适配器
type responseWriterAdapter struct {
	http.ResponseWriter
}
