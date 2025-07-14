package pkg

import (
	"context"
	"neo/internal/config"
	"neo/internal/types"
)

// Client 定义公开的客户端接口，供外部调用
type Client interface {
	// Call 执行远程调用
	Call(ctx context.Context, req types.Request) (types.Response, error)
}

// 内部实现结构体，隐藏实现细节
type client struct {
	config config.Config
	// 可以添加其他内部依赖
}

// NewClient 创建客户端实例
// 参数: config - 客户端配置
// 返回: Client接口实例
func NewClient(config config.Config) Client {
	return &client{
		config: config,
	}
}

// Call 实现Client接口的Call方法
func (c *client) Call(ctx context.Context, req types.Request) (types.Response, error) {
	// 移除结构体比较，避免编译错误
	return types.Response{}, nil
}
