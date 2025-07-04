package tcp

import (
	"neo/internal/types"
)

// 类型别名迁移（保留兼容）
type ConnectionHandler = types.ConnectionHandler

// NewConnectionHandler 创建新的连接处理器
func NewConnectionHandler(config *types.Config, pool *types.TCPConnectionPool) *ConnectionHandler {
	return &types.ConnectionHandler{
		Config:         config, // 使用导出字段
		ConnectionPool: pool,   // 使用导出字段
	}
}
