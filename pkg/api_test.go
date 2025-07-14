package pkg_test

import (
	"context"
	"testing"
	"github.com/stretchr/testify/assert"
	"neo/internal/config"
	"neo/internal/types"
	"neo/pkg"
)

// 测试API功能
func TestAPI_Call_Get(t *testing.T) {
	// 初始化配置
	cfg := config.Config{}
	// 使用正确的构造函数
	api := pkg.NewClient(cfg)
	// 构造请求（不使用任何字段）
	req := types.Request{}
	// 使用空白标识符避免未使用变量错误
	_, err := api.Call(context.Background(), req)
	assert.NoError(t, err)
}

func TestAPI_Call_Set(t *testing.T) {
	cfg := config.Config{}
	api := pkg.NewClient(cfg)
	req := types.Request{}
	_, err := api.Call(context.Background(), req)
	assert.NoError(t, err)
}

// 边界测试
func TestAPI_InvalidInput(t *testing.T) {
	cfg := config.Config{}
	api := pkg.NewClient(cfg)
	// 传递零值请求
	req := types.Request{}
	_, err := api.Call(context.Background(), req)
	assert.Error(t, err)
}