package http

import (
	"net/http"
)

// Router HTTP路由管理器
type Router struct {
	handlers map[string]http.HandlerFunc
}

// NewRouter 创建新的路由实例
func NewRouter() *Router {
	return &Router{
		handlers: make(map[string]http.HandlerFunc),
	}
}

// Handle 注册HTTP处理函数
func (r *Router) Handle(pattern string, handler func(w http.ResponseWriter, req *http.Request)) {
	r.handlers[pattern] = handler
}

// ServeHTTP 实现http.Handler接口
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if handler, ok := r.handlers[req.URL.Path]; ok {
		handler(w, req)
		return
	}

	http.Error(w, "Not found", http.StatusNotFound)
}
