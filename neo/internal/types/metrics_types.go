package types

import (
	"context"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics 监控指标管理器
type Metrics struct {
	Mu       sync.RWMutex         // 读写锁保护并发访问
	Registry *prometheus.Registry // 指标注册表
	Server   *http.Server         // 监控服务器实例
}

// NewMetrics 创建新的指标管理器实例
func NewMetrics(registry *prometheus.Registry) *Metrics {
	return &Metrics{
		Registry: registry,
	}
}

// Close 关闭指标服务器
func (m *Metrics) Close(ctx context.Context) error {
	if m.Server != nil {
		return m.Server.Shutdown(ctx)
	}
	return nil
}
