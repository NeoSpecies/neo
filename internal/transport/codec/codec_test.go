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
			Body: []byte("HTTP测试消息内容"),
		},
	}, {
		name:         "IPC协议测试",
		protocolType: "ipc",
		originalMsg: types.Message{
			ID:      "test-ipc-456",
			Body: []byte("IPC测试消息内容"),
		},
	}}

	// 遍历测试用例
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建编解码器
			codec, err := NewCodec(tc.protocolType)
			if err != nil {
				t.Fatalf("创建%s编解码器失败: %v", tc.protocolType, err)
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
			if string(decodedMsg.Body) != string(tc.originalMsg.Body) {
				t.Errorf("%s 内容不匹配: 期望%s, 实际%s", tc.protocolType, string(tc.originalMsg.Body), string(decodedMsg.Body))
			}
		})
	}
}

// TestNewCodec_UnsupportedProtocol 测试不支持的协议类型
func TestNewCodec_UnsupportedProtocol(t *testing.T) {
	// 尝试创建不支持的协议编解码器
	_, err := NewCodec("unsupported")
	if err == nil {
		t.Error("预期不支持的协议会返回错误，但没有")
	}
}

// 测试超大消息编码/解码
func TestCodec_LargeMessage(t *testing.T) {
	largeBody := make([]byte, 1024*1024) // 1MB测试数据
	for i := range largeBody {
		largeBody[i] = byte(i % 256)
	}

	// 移除未使用的name字段或使用匿名结构体
	testCase := struct {
		protocolType string
		originalMsg  types.Message
	}{"http", types.Message{ID: "large-msg-001", Body: largeBody}}

	codec, err := NewCodec(testCase.protocolType)
	if err != nil {
		t.Fatalf("创建%s编解码器失败: %v", testCase.protocolType, err)
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
	if !bytes.Equal(decodedMsg.Body, testCase.originalMsg.Body) {
		t.Error("超大消息内容解码不一致")
	}
}

// 测试损坏数据解码错误处理
func TestCodec_CorruptedData(t *testing.T) {
	codec, err := NewCodec("http")
	if err != nil {
		t.Fatal("创建HTTP编解码器失败: ", err)
	}

	// 测试损坏的JSON数据
	corruptedData := []byte(`{"id":"test","content":[invalid]}`)
	_, err = codec.Decode(context.Background(), corruptedData)
	if err == nil {
		t.Error("预期损坏数据解码失败，但未返回错误")
	}
}

// 编解码性能基准测试
func BenchmarkCodec_Performance(b *testing.B) {
	msg := types.Message{ID: "bench-msg", Body: []byte("基准测试消息内容重复多次以增加长度基准测试消息内容")}
	codec, err := NewCodec("http")
	if err != nil {
		b.Fatal("创建编解码器失败: ", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		encoded, _ := codec.Encode(context.Background(), msg)
		codec.Decode(context.Background(), encoded)
	}
}
