# Neo Framework Gateway Services

## 概述

Neo框架的网关服务提供了对外的访问接入点，允许外部客户端通过HTTP或TCP协议访问框架内部的服务。这些网关服务本身也是注册到Neo IPC服务器的服务，实现了服务间的请求路由和转发。

## 架构设计

```
外部客户端
    ↓
┌─────────────────────────────────────────┐
│         Gateway Services (对外)          │
├──────────────────┬──────────────────────┤
│   HTTP Gateway   │    TCP Gateway       │
│   (端口:8080)    │    (端口:7777)       │
└──────────┬───────┴───────┬──────────────┘
           ↓               ↓
           └───────┬───────┘
                   ↓ (IPC Protocol)
┌──────────────────────────────────────────┐
│         Neo IPC Server (内部)            │
│            (端口:9999)                    │
└──────────┬──────────┬────────────────────┘
           ↓          ↓
    ┌──────────┐ ┌──────────┐
    │业务服务1 │ │业务服务2 │
    └──────────┘ └──────────┘
```

## 服务说明

### HTTP Gateway

HTTP网关服务提供RESTful API接口，接收HTTP请求并通过IPC转发到内部服务。

**特性：**
- RESTful API风格
- JSON请求/响应格式
- 健康检查端点
- 服务信息查询

**端点：**
- `/api/{service}/{method}` - API调用端点
- `/health` - 健康检查
- `/info` - 网关信息

### TCP Gateway

TCP网关服务提供原生TCP连接，支持JSON协议的消息通信。

**特性：**
- 长连接支持
- JSON协议（长度前缀）
- 连接管理
- 统计信息

**协议格式：**
```
[消息长度:4字节][JSON消息内容]
```

## 快速开始

### 1. 启动Neo核心

首先确保Neo核心（IPC Server）正在运行：

```bash
# 在项目根目录
./neo.exe -ipc :9999
```

### 2. 启动网关服务

#### 方式一：一键启动所有服务

```bash
cd services
start_all.bat
```

#### 方式二：单独启动

启动HTTP网关：
```bash
cd services
start_http.bat
# 或手动指定参数
cd http-gateway
./http-gateway.exe -ipc localhost:9999 -http :8080
```

启动TCP网关：
```bash
cd services
start_tcp.bat
# 或手动指定参数
cd tcp-gateway
./tcp-gateway.exe -ipc localhost:9999 -tcp :7777
```

### 3. 注册业务服务

启动示例服务（如Go服务）：
```bash
cd examples-ipc/go
go run service.go
```

## 使用示例

### HTTP Gateway 使用

#### 使用curl调用服务

```bash
# 调用hello方法
curl -X POST http://localhost:8080/api/demo-service-go/hello \
  -H "Content-Type: application/json" \
  -d '{"name":"World"}'

# 调用calculate方法
curl -X POST http://localhost:8080/api/demo-service-go/calculate \
  -H "Content-Type: application/json" \
  -d '{"a":10,"b":20,"operation":"add"}'

# 健康检查
curl http://localhost:8080/health

# 获取网关信息
curl http://localhost:8080/info
```

#### 使用Python客户端

```python
import requests
import json

# API基础URL
base_url = "http://localhost:8080/api"

# 调用hello方法
response = requests.post(
    f"{base_url}/demo-service-go/hello",
    json={"name": "Python Client"}
)
print(response.json())

# 调用calculate方法
response = requests.post(
    f"{base_url}/demo-service-go/calculate",
    json={"a": 10, "b": 20, "operation": "multiply"}
)
print(response.json())
```

### TCP Gateway 使用

#### 使用测试客户端

```bash
cd services/tcp-gateway
go build -o test_client.exe test_client.go
./test_client.exe localhost:7777
```

#### 自定义TCP客户端（Python示例）

```python
import socket
import struct
import json

def send_tcp_request(host, port, service, method, data):
    # 创建连接
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.connect((host, port))
    
    try:
        # 构建消息
        message = {
            "service": service,
            "method": method,
            "data": data
        }
        
        # 序列化为JSON
        msg_bytes = json.dumps(message).encode('utf-8')
        
        # 发送长度（4字节，大端序）
        sock.send(struct.pack('>I', len(msg_bytes)))
        
        # 发送消息
        sock.send(msg_bytes)
        
        # 读取响应长度
        resp_len_bytes = sock.recv(4)
        resp_len = struct.unpack('>I', resp_len_bytes)[0]
        
        # 读取响应
        resp_bytes = sock.recv(resp_len)
        response = json.loads(resp_bytes.decode('utf-8'))
        
        return response
        
    finally:
        sock.close()

# 使用示例
response = send_tcp_request(
    'localhost', 7777,
    'demo-service-go', 'hello',
    {'name': 'TCP Python Client'}
)
print(response)
```

## 配置说明

### HTTP Gateway 配置

命令行参数：
- `-ipc` : IPC服务器地址（默认：localhost:9999）
- `-http` : HTTP监听地址（默认：:8080）

### TCP Gateway 配置

命令行参数：
- `-ipc` : IPC服务器地址（默认：localhost:9999）
- `-tcp` : TCP监听地址（默认：:7777）
- `-protocol` : 协议类型，json或binary（默认：json）

## 开发指南

### 添加新的处理器

网关服务可以注册自己的IPC处理器，供其他服务调用：

```go
// 在registerHandlers方法中添加
s.ipcClient.AddHandler("myMethod", func(msg *common.Message) (*common.Message, error) {
    // 处理逻辑
    result := map[string]interface{}{
        "status": "success",
    }
    
    data, _ := json.Marshal(result)
    return &common.Message{
        Metadata: map[string]string{},
        Data:     data,
    }, nil
})
```

### 扩展协议支持

TCP网关预留了协议扩展接口，可以添加新的协议支持：

1. 在`readBinaryMessage`和`writeBinaryMessage`方法中实现新协议
2. 在命令行参数中添加新的协议类型
3. 更新协议选择逻辑

## 监控和调试

### 查看服务状态

通过HTTP网关的info端点查看状态：
```bash
curl http://localhost:8080/info
```

通过IPC调用TCP网关的getStats方法：
```bash
# 需要通过HTTP网关调用
curl -X POST http://localhost:8080/api/tcp-gateway/getStats
```

### 日志输出

两个网关服务都会输出详细的日志信息：
- 连接建立/断开
- 请求处理
- 错误信息
- 心跳状态

## 性能优化

1. **连接复用**：TCP网关支持长连接，减少连接建立开销
2. **并发处理**：每个连接都在独立的goroutine中处理
3. **超时控制**：HTTP和TCP都设置了合理的超时时间
4. **缓冲优化**：使用缓冲读写提高性能

## 故障排查

### 常见问题

1. **无法连接到IPC服务器**
   - 检查Neo核心是否运行
   - 确认端口9999是否被占用
   - 查看防火墙设置

2. **服务调用失败**
   - 确认目标服务已注册
   - 检查服务名和方法名是否正确
   - 查看请求数据格式

3. **TCP连接断开**
   - 检查消息格式是否正确
   - 确认没有超过最大消息大小（1MB）
   - 查看超时设置

## 安全建议

1. **生产环境**：
   - 使用防火墙限制访问
   - 实施认证机制
   - 启用TLS加密

2. **输入验证**：
   - 验证请求大小
   - 检查JSON格式
   - 防止注入攻击

## 版本历史

- v1.0.0 - 初始版本
  - HTTP网关服务
  - TCP网关服务（JSON协议）
  - IPC集成
  - 请求路由

## 许可证

本项目遵循Neo Framework的许可证。

---

*文档编写：Neo Framework Team*
*最后更新：2024*