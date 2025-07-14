package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// 预编译的正则表达式
var (
	serviceNameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	methodNameRegex  = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emailRegex       = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateServiceName 验证服务名称
func ValidateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("service name too long (max 255 characters)")
	}
	if !serviceNameRegex.MatchString(name) {
		return fmt.Errorf("invalid service name: must contain only letters, numbers, dots, underscores, and hyphens")
	}
	return nil
}

// ValidateMethodName 验证方法名称
func ValidateMethodName(name string) error {
	if name == "" {
		return fmt.Errorf("method name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("method name too long (max 100 characters)")
	}
	if !methodNameRegex.MatchString(name) {
		return fmt.Errorf("invalid method name: must contain only letters, numbers, and underscores")
	}
	return nil
}

// FormatEndpoint 格式化端点
func FormatEndpoint(service, method string) string {
	return fmt.Sprintf("%s/%s", service, method)
}

// ParseEndpoint 解析端点
func ParseEndpoint(endpoint string) (service, method string, err error) {
	parts := strings.Split(endpoint, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid endpoint format: %s", endpoint)
	}
	return parts[0], parts[1], nil
}

// ToCamelCase 转换为驼峰命名
func ToCamelCase(s string) string {
	if s == "" {
		return ""
	}
	
	// 分割单词
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	// 转换每个单词
	for i, word := range words {
		if i == 0 {
			words[i] = strings.ToLower(word)
		} else {
			words[i] = strings.Title(strings.ToLower(word))
		}
	}
	
	return strings.Join(words, "")
}

// ToPascalCase 转换为帕斯卡命名
func ToPascalCase(s string) string {
	if s == "" {
		return ""
	}
	
	// 分割单词
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	
	// 转换每个单词
	for i, word := range words {
		words[i] = strings.Title(strings.ToLower(word))
	}
	
	return strings.Join(words, "")
}

// ToSnakeCase 转换为蛇形命名
func ToSnakeCase(s string) string {
	if s == "" {
		return ""
	}
	
	var result strings.Builder
	for i, r := range s {
		if i > 0 && unicode.IsUpper(r) && (i+1 < len(s) && unicode.IsLower(rune(s[i+1])) || unicode.IsLower(rune(s[i-1]))) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	
	return result.String()
}

// ToKebabCase 转换为烤串命名
func ToKebabCase(s string) string {
	if s == "" {
		return ""
	}
	
	snake := ToSnakeCase(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// Truncate 截断字符串
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// IsEmpty 检查字符串是否为空（包括空白字符）
func IsEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// SanitizeString 清理字符串（移除控制字符）
func SanitizeString(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, s)
}

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) bool {
	return emailRegex.MatchString(email)
}

// JoinPath 连接路径部分
func JoinPath(parts ...string) string {
	var nonEmpty []string
	for _, part := range parts {
		part = strings.Trim(part, "/")
		if part != "" {
			nonEmpty = append(nonEmpty, part)
		}
	}
	return "/" + strings.Join(nonEmpty, "/")
}

// EscapeHTML 转义HTML特殊字符
func EscapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}