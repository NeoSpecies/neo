package codec

import (
	"bytes"
	"context"
	"neo/internal/types"
	"testing"
)

// TestCodec_EncodeDecode 测试编解码器的编码和解码功能
func TestCodec_EncodeDecode(t *testing.T) {
	// 定义测试用例
	testCases := []struct {
		name         string
		protocolType string
		originalMsg  types.Message
	}{{
		name:         "HTTP协议测试",
		protocolType: "http",
		originalMsg: types.Message{
			ID:      "test-http-123",
			Content: []byte("HTTP测试消息内容"),
		},
	}, {
		name:         "IPC协议测试",
		protocolType: "ipc",
		originalMsg: types.Message{
			ID:      "test-ipc-456",
			Content: []byte("IPC测试消息内容"),
		},
	}}

	// 遍历测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建编解码器
			codec := NewCodec(tc.protocolType)
			if codec == nil {
				t.Fatalf("创建%s编解码器失败", tc.protocolType)
			}

			// 编码消息
			encodedData, err := codec.Encode(context.Background(), tc.originalMsg)
			if err != nil {
				t.Fatalf("%s编码失败: %v", tc.protocolType, err)
			}

			// 解码消息
			decodedMsg, err := codec.Decode(context.Background(), encodedData)
			if err != nil {
				t.Fatalf("%s解码失败: %v", tc.protocolType, err)
			}

			// 验证解码后的消息是否与原始消息一致
			if decodedMsg.ID != tc.originalMsg.ID {
				t.Errorf("%s ID不匹配: 期望%s, 实际%s", tc.protocolType, tc.originalMsg.ID, decodedMsg.ID)
			}
			if string(decodedMsg.Content) != string(tc.originalMsg.Content) {
				t.Errorf("%s 内容不匹配: 期望%s, 实际%s", tc.protocolType, string(tc.originalMsg.Content), string(decodedMsg.Content))
			}
		})
	}
}

// TestNewCodec_UnsupportedProtocol 测试不支持的协议类型
func TestNewCodec_UnsupportedProtocol(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("预期不支持的协议会触发panic，但没有")
		}
	}()

	// 尝试创建不支持的协议编解码器
	NewCodec("unsupported")
}

// 测试超大消息编码/解码
func TestCodec_LargeMessage(t *testing.T) {
	largeContent := make([]byte, 1024*1024) // 1MB测试数据
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	// 移除未使用的name字段或使用匿名结构体
	testCase := struct {
		protocolType string
		originalMsg  types.Message
	}{"http", types.Message{ID: "large-msg-001", Content: largeContent}}

	codec := NewCodec(testCase.protocolType)
	if codec == nil {
		t.Fatalf("创建%s编解码器失败", testCase.protocolType)
	}

	// 编码测试
	encodedData, err := codec.Encode(context.Background(), testCase.originalMsg)
	if err != nil {
		t.Fatalf("超大消息编码失败: %v", err)
	}

	// 解码测试
	decodedMsg, err := codec.Decode(context.Background(), encodedData)
	if err != nil {
		t.Fatalf("超大消息解码失败: %v", err)
	}

	// 验证数据完整性
	if !bytes.Equal(decodedMsg.Content, testCase.originalMsg.Content) {
		t.Error("超大消息内容解码不一致")
	}
}

// 测试损坏数据解码错误处理
func TestCodec_CorruptedData(t *testing.T) {
	codec := NewCodec("http")
	if codec == nil {
		t.Fatal("创建HTTP编解码器失败")
	}

	// 测试损坏的JSON数据
	corruptedData := []byte(`{"id":"test","content":[invalid]}`)
	_, err := codec.Decode(context.Background(), corruptedData)
	if err == nil {
		t.Error("预期损坏数据解码失败，但未返回错误")
	}
}

// 编解码性能基准测试
func BenchmarkCodec_Performance(b *testing.B) {
	msg := types.Message{ID: "bench-msg", Content: []byte("基准测试消息内容重复多次以增加长度基准测试消息内容")}
	codec := NewCodec("http")
	if codec == nil {
		b.Fatal("创建编解码器失败")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded, _ := codec.Encode(context.Background(), msg)
		codec.Decode(context.Background(), encoded)
	}
}
