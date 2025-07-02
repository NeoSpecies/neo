package discovery

import (
	"log"
	"neo/internal/types"
)

// HandleMessage 处理服务发现协议消息
func HandleMessage(msg *types.Message) {
	log.Printf("收到消息类型: %v, 服务信息: %+v", msg.Type, msg.Service)
	// 现有处理逻辑...
}
