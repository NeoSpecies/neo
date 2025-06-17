package core // 新增包声明

import (
	"sync"
)

var callbackMap = struct {
	sync.RWMutex
	m map[string]func(interface{}, error)
}{m: make(map[string]func(interface{}, error))}

func RegisterCallback(msgID string, cb func(interface{}, error)) {
	callbackMap.Lock()
	defer callbackMap.Unlock()
	callbackMap.m[msgID] = cb
}

func HandleResponse(msgID string, result interface{}, err error) {
	callbackMap.RLock()
	cb, exists := callbackMap.m[msgID]
	callbackMap.RUnlock()

	if exists {
		go cb(result, err) // 异步执行回调
		// 清理回调（需添加超时清理机制）
		callbackMap.Lock()
		delete(callbackMap.m, msgID)
		callbackMap.Unlock()
	}
}
