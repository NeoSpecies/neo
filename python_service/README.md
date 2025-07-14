# Neo Python Service Integration

本目录包含了Neo框架的Python服务集成实现，支持通过IPC与Go网关进行异步通信。

## 架构概述

```
HTTP客户端 → Go HTTP网关 (28080) → IPC服务器 (29999) → Python服务
             ↑                                           ↓
             ← ← ← ← ← ← 异步响应 ← ← ← ← ← ← ← ← ← ← ←
```

## 组件说明

### Python端组件

1. **neo_client.py** - IPC客户端库
   - 支持服务注册
   - 异步消息监听
   - 请求处理和响应

2. **example_service.py** - 示例Python服务
   - 数学运算服务
   - 展示如何使用neo_client

### Go端组件

1. **HTTP网关** (`internal/gateway/http_gateway.go`)
   - 接收HTTP请求
   - 转发到IPC服务

2. **IPC服务器** (`internal/ipc/server.go`)
   - 管理服务注册
   - 处理进程间通信

3. **异步传输** (`internal/transport/async_ipc_transport.go`)
   - 支持异步请求-响应

## 快速开始

### 1. 启动Neo Framework

```bash
cd C:\Dev\neo
go run cmd/neo/main.go
```

服务将在以下端口启动：
- HTTP网关: 28080
- IPC服务器: 29999

### 2. 启动Python服务

```bash
cd C:\Dev\neo\python_service
python example_service.py
```

### 3. 测试API

使用测试客户端：

```bash
cd C:\Dev\neo\test

# 测试加法
python test_client.py add 10 20

# 测试乘法
python test_client.py multiply 5 6

# 测试表达式计算
python test_client.py calculate "2 * (3 + 4)"

# 健康检查
python test_client.py health
```

或使用curl：

```bash
# 加法
curl -X POST http://localhost:28080/api/python.math/add \
  -H "Content-Type: application/json" \
  -d '{"a": 10, "b": 20}'

# 乘法
curl -X POST http://localhost:28080/api/python.math/multiply \
  -H "Content-Type: application/json" \
  -d '{"a": 5, "b": 6}'
```

## 运行集成测试

```bash
cd C:\Dev\neo\test
python integration_test.py
```

## 开发自己的Python服务

1. 导入neo_client：

```python
from neo_client import NeoIPCClient
```

2. 创建服务类：

```python
class MyService:
    def __init__(self):
        self.client = NeoIPCClient()
    
    async def start(self):
        await self.client.connect()
        await self.client.register_service("my.service")
        
        # 注册处理器
        self.client.register_handler("method1", self.handle_method1)
        
        # 开始监听
        await self.client.listen()
    
    async def handle_method1(self, data):
        # 处理请求
        return {"result": "success"}
```

3. 运行服务：

```python
import asyncio

async def main():
    service = MyService()
    await service.start()

if __name__ == "__main__":
    asyncio.run(main())
```

## 注意事项

1. 确保Go网关服务在Python服务之前启动
2. 服务名称应遵循域名风格（如 `python.math`）
3. 所有通信都是异步的，支持并发请求
4. 请求超时时间为30秒

## 故障排查

1. **连接错误**: 检查端口29999是否被占用
2. **服务未找到**: 确保Python服务已注册
3. **超时错误**: 检查Python服务是否正常处理请求
4. **健康检查失败**: 确保实现心跳机制或延长健康检查超时

## 性能优化

1. 使用连接池减少连接开销
2. 批量处理请求提高吞吐量
3. 合理设置超时时间
4. 使用适当的并发级别