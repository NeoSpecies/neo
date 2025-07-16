# Go IPC 客户端示例

这是一个 Go 语言的 IPC 客户端示例，展示如何连接到 Neo Framework。

## 运行要求

- Go >= 1.16

## 快速开始

```bash
# 直接运行
go run service.go

# 或编译后运行
go build -o service
./service
```

## 环境变量配置

```bash
# 设置 IPC 服务器地址（默认: localhost）
export NEO_IPC_HOST=localhost

# 设置 IPC 服务器端口（默认: 9999）
export NEO_IPC_PORT=9999
```

## 测试服务

```bash
# 测试 hello
curl -X POST http://localhost:8080/api/demo-service/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Go"}'

# 测试计算
curl -X POST http://localhost:8080/api/demo-service/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 15, "b": 3, "operation": "divide"}'

# 测试回显
curl -X POST http://localhost:8080/api/demo-service/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from Go!"}'

# 获取时间
curl -X POST http://localhost:8080/api/demo-service/getTime \
  -H "Content-Type: application/json" \
  -d '{"format": "unix"}'

# 获取服务信息
curl -X POST http://localhost:8080/api/demo-service/getInfo
```

## 代码结构

- `service.go` - 主服务文件，包含：
  - `NeoIPCClient` - IPC 客户端结构
  - 消息编解码逻辑
  - 示例处理器实现

## 扩展开发

1. 添加新的处理器：
```go
client.AddHandler("myMethod", func(params map[string]interface{}) (interface{}, error) {
    // 你的业务逻辑
    return map[string]interface{}{
        "result": "success",
    }, nil
})
```

2. 修改服务名称：
```go
err = client.RegisterService("my-service-name", map[string]string{
    "version": "1.0.0",
})
```

## 性能优势

- Go 的并发模型非常适合处理高并发请求
- 原生支持协程，每个请求都在独立的 goroutine 中处理
- 编译型语言，性能优异

## 注意事项

- 使用小端序（Little Endian）进行二进制编码
- 心跳间隔为 30 秒
- 支持并发请求处理