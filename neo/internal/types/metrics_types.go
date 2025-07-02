package types

import (
	"context"
)

// Close 关闭指标服务器
func (m *Metrics) Close(ctx context.Context) error {
	if m.Server != nil {
		return m.Server.Shutdown(ctx)
	}
	return nil
}
