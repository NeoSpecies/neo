# Python IPC 客户端示例

这是一个简单的 Python IPC 客户端示例，展示如何连接到 Neo Framework。

## 运行要求

- Python >= 3.7
- 无需额外依赖（仅使用标准库）

## 快速开始

```bash
# 直接运行
python service.py

# 或添加执行权限后运行（Linux/Mac）
chmod +x service.py
./service.py
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
  -d '{"name": "Python"}'

# 测试计算
curl -X POST http://localhost:8080/api/demo-service/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 10, "b": 5, "operation": "multiply"}'

# 测试回显
curl -X POST http://localhost:8080/api/demo-service/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from Python!"}'

# 获取时间
curl -X POST http://localhost:8080/api/demo-service/getTime \
  -H "Content-Type: application/json" \
  -d '{"format": "readable"}'

# 获取服务信息
curl -X POST http://localhost:8080/api/demo-service/getInfo
```

## 代码结构

- `service.py` - 主服务文件，包含：
  - `NeoIPCClient` - 简化的 IPC 客户端类
  - 消息处理逻辑
  - 示例业务处理器

## 扩展开发

1. 添加新的处理器：
```python
@client.handler("myMethod")
async def my_method(params):
    # 你的业务逻辑
    return {"result": "success"}
```

2. 修改服务名称：
```python
await client.register_service("my-service-name", {
    "version": "1.0.0"
})
```

## 注意事项

- 使用小端序（Little Endian）进行二进制编码
- 心跳间隔为 30 秒
- 支持异步处理器（async/await）