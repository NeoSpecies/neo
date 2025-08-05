package retry

import (
	"context"
	"errors"
	"fmt"
	"neo/internal/config"
	"strings"
	"testing"
	"time"
)

// TestExecute_SuccessOnFirstAttempt 测试首次尝试即成功的情况
func TestExecute_SuccessOnFirstAttempt(t *testing.T) {
	cfg := &config.Config{
		Transport: config.TransportConfig{
			RetryCount:     3,
			InitialBackoff: config.Duration(100 * time.Millisecond),
			MaxBackoff:     config.Duration(5 * time.Second),
		},
	}
	policy := NewRetryPolicy(cfg)
	ctx := context.Background()

	attemptCount := 0
	err := policy.Execute(ctx, func() error {
		attemptCount++
		return nil // 首次执行成功
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("expected 1 attempt, got %d", attemptCount)
	}
}

// TestExecute_RetryUntilSuccess 测试重试直到成功的情况
func TestExecute_RetryUntilSuccess(t *testing.T) {
	cfg := &config.Config{
		Transport: config.TransportConfig{
			RetryCount:     3,
			InitialBackoff: config.Duration(100 * time.Millisecond),
			MaxBackoff:     config.Duration(5 * time.Second),
		},
	}
	policy := NewRetryPolicy(cfg)
	ctx := context.Background()

	attemptCount := 0
	maxFailures := 2 // 前2次失败，第3次成功

	err := policy.Execute(ctx, func() error {
		attemptCount++
		if attemptCount <= maxFailures {
			return errors.New("temporary failure")
		}
		return nil
	})

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if attemptCount != maxFailures+1 {
		t.Errorf("expected %d attempts, got %d", maxFailures+1, attemptCount)
	}
}

// TestExecute_MaxRetriesExceeded 测试达到最大重试次数的情况
func TestExecute_MaxRetriesExceeded(t *testing.T) {
	cfg := &config.Config{
		Transport: config.TransportConfig{
			RetryCount:     3,
			InitialBackoff: config.Duration(100 * time.Millisecond),
			MaxBackoff:     config.Duration(5 * time.Second),
		},
	}
	policy := NewRetryPolicy(cfg)
	ctx := context.Background()

	attemptCount := 0
	expectedErr := errors.New("persistent failure")

	err := policy.Execute(ctx, func() error {
		attemptCount++
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
	// 修复断言：预期尝试次数 = RetryCount + 1
	if attemptCount != cfg.Transport.RetryCount+1 {
		t.Errorf("expected %d attempts, got %d", cfg.Transport.RetryCount+1, attemptCount)
	}

	// 删除以下错误引入的代码块
	/*
	   // 调整退避时间计算（4次重试间隔：10+20+40+80=150ms）
	   expectedMinElapsed := time.Duration(10+20+40+80) * time.Millisecond  // 150ms
	   expectedMaxElapsed := time.Duration(10+20+40+100) * time.Millisecond // 170ms

	   if elapsed < expectedMinElapsed || elapsed > expectedMaxElapsed {
	       t.Errorf("elapsed time %v not in range [%v, %v]", elapsed, expectedMinElapsed, expectedMaxElapsed)
	   }
	*/
}

// TestExecute_ContextCancelled 测试上下文取消的情况
func TestExecute_ContextCancelled(t *testing.T) {
	cfg := &config.Config{
		Transport: config.TransportConfig{
			RetryCount:     5,
			InitialBackoff: config.Duration(100 * time.Millisecond),
			MaxBackoff:     config.Duration(5 * time.Second),
		},
	}
	policy := NewRetryPolicy(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	attemptCount := 0
	go func() {
		// 延长睡眠时间，确保至少完成第一次尝试并进入等待阶段
		// 初始间隔为10ms（cfg.Transport.InitialBackoff=10），等待时间需超过初始间隔
		time.Sleep(150 * time.Millisecond)
		cancel()
	}()

	err := policy.Execute(ctx, func() error {
		attemptCount++
		return errors.New("temporary failure")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// 调整判断条件：尝试次数应大于1且小于最大重试次数
	if attemptCount <= 1 || attemptCount >= cfg.Transport.RetryCount {
		t.Errorf("expected attempts between 2 and %d, got %d", cfg.Transport.RetryCount-1, attemptCount)
	}
}

// TestExponentialBackoff 测试指数退避逻辑
func TestExponentialBackoff(t *testing.T) {
	// 使用正确的配置字段名（与retry.go中保持一致）
	cfg := &config.Config{
		Transport: config.TransportConfig{
			RetryCount:     4,
			InitialBackoff: config.Duration(10 * time.Millisecond),
			MaxBackoff:     config.Duration(100 * time.Millisecond),
		},
	}

	policy := NewRetryPolicy(cfg)

	var attempts int
	startTime := time.Now()

	// 修复函数签名：移除ctx参数（Execute方法已接收ctx）
	err := policy.Execute(context.Background(), func() error {
		attempts++
		return fmt.Errorf("temporary failure")
	})

	elapsed := time.Since(startTime)

	if err == nil || !strings.Contains(err.Error(), "temporary failure") {
		t.Errorf("Expected 'temporary failure' error, got: %v", err)
	}

	if attempts != 5 {
		t.Errorf("expected 5 attempts, got %d", attempts)
	}

	if elapsed < 150*time.Millisecond || elapsed > 170*time.Millisecond {
		t.Errorf("Unexpected total backoff time: %v", elapsed)
	}
}

// TestExecute_SingleAttempt 测试仅执行单次尝试（无重试）的场景
func TestExecute_SingleAttempt(t *testing.T) {
	// 导入必要的errors包

	// 配置零重试（实际执行1次尝试）
	cfg := &config.Config{
		Transport: config.TransportConfig{
			RetryCount:     0, // 零重试配置
			InitialBackoff: config.Duration(100 * time.Millisecond),
			MaxBackoff:     config.Duration(500 * time.Millisecond),
		},
	}
	policy := NewRetryPolicy(cfg)

	attemptCount := 0
	// 确保操作函数返回预期错误
	err := policy.Execute(context.Background(), func() error {
		attemptCount++
		return errors.New("temporary failure") // 明确返回错误
	})

	// 验证错误和尝试次数
	if err == nil || err.Error() != "temporary failure" {
		t.Errorf("预期错误 'temporary failure'，实际得到 %v", err)
	}
	if attemptCount != 1 {
		t.Errorf("零重试时应执行1次，实际执行了%d次", attemptCount)
	}
}
