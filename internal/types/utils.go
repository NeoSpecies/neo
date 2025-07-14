package types

import (
	"crypto/rand"
	"encoding/hex"
	"sync/atomic"
	"time"
)

// ID 生成器
var (
	idCounter uint64
)

// GenerateID 生成唯一ID
// 格式：时间戳-随机数-计数器
func GenerateID() string {
	// 获取当前时间戳
	timestamp := time.Now().UnixNano()

	// 生成4字节随机数
	randBytes := make([]byte, 4)
	rand.Read(randBytes)

	// 原子递增计数器
	counter := atomic.AddUint64(&idCounter, 1)

	// 组合成ID
	return hex.EncodeToString([]byte{
		byte(timestamp >> 56), byte(timestamp >> 48), byte(timestamp >> 40), byte(timestamp >> 32),
		byte(timestamp >> 24), byte(timestamp >> 16), byte(timestamp >> 8), byte(timestamp),
		randBytes[0], randBytes[1], randBytes[2], randBytes[3],
		byte(counter >> 24), byte(counter >> 16), byte(counter >> 8), byte(counter),
	})
}

// ValidateServiceName 验证服务名称
func ValidateServiceName(name string) error {
	if name == "" {
		return ErrInvalidService
	}
	// 服务名必须是字母、数字、点号、下划线或短横线
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == '_' || ch == '-') {
			return ErrInvalidService
		}
	}
	return nil
}

// ValidateMethodName 验证方法名称
func ValidateMethodName(name string) error {
	if name == "" {
		return ErrInvalidMethod
	}
	// 方法名必须是字母、数字或下划线
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '_') {
			return ErrInvalidMethod
		}
	}
	return nil
}