package retry

import (
	"context"
	"neo/internal/config"
	"time"
)

// RetryPolicy 重试策略接口
type RetryPolicy interface {
	// Execute 执行带重试的操作
	Execute(ctx context.Context, operation func() error) error
}

// exponentialBackoff 指数退避策略实现
type exponentialBackoff struct {
	maxAttempts     int           // 最大重试次数
	initialInterval time.Duration // 初始间隔
	maxInterval     time.Duration // 最大间隔
}

// NewRetryPolicy 根据配置创建重试策略
func NewRetryPolicy(cfg *config.Config) RetryPolicy {
    // 从配置中读取重试参数
    return &exponentialBackoff{
        maxAttempts:     cfg.Transport.RetryCount,
        initialInterval: time.Duration(cfg.Transport.InitialBackoff),
        maxInterval:     time.Duration(cfg.Transport.MaxBackoff),
    }
}

// Execute 执行带重试的操作
func (e *exponentialBackoff) Execute(ctx context.Context, operation func() error) error {
	var err error
	interval := e.initialInterval

	// 修复：至少执行一次操作
	for attempt := 0; attempt <= e.maxAttempts; attempt++ {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// 尝试执行操作
		err = operation()
		if err == nil {
			// 操作成功，直接返回
			return nil
		}

		// 如果达到最大重试次数，不再重试
		if attempt >= e.maxAttempts {
			break
		}

		// 等待后重试
		timer := time.NewTimer(interval)
		select {
		case <-timer.C:
			// 增加间隔，但不超过最大间隔
			interval *= 2
			if interval > e.maxInterval {
				interval = e.maxInterval
			}
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	}

	// 返回最后一次错误
	return err
}
