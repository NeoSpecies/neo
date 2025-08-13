# Neo Framework 网关服务文档

## 概述

Neo Framework 采用微服务架构，将HTTP和TCP网关作为独立服务运行。这些网关服务通过IPC协议注册到Neo核心，提供外部访问接入点，并将请求转发到内部微服务。

## 架构设计

### 设计原则

1. **服务独立性**：每个网关服务都是完全独立的，包含自己的IPC客户端代码
2. **无共享依赖**：服务之间不共享任何代码，确保独立部署和升级
3. **协议转换**：网关负责外部协议（HTTP/TCP）与内部IPC协议的转换
4. **服务注册**：网关作为IPC服务注册，可被其他服务调用

### 通信流程

```
外部请求 → 网关服务 → IPC协议 → Neo Core → 目标服务
         ↑                                    ↓
         ← 外部响应 ← IPC协议 ← Neo Core ← 服务响应
```

## HTTP 网关服务

### 概述
HTTP网关提供RESTful API接入点，将HTTP请求转换为IPC协议并转发到内部服务。

### 功能特性
- RESTful API 路由：`/api/{service}/{method}`
- 自动服务发现和路由
- 请求/响应转换
- 错误处理和状态码映射
- 支持JSON数据格式
- 自身作为IPC服务提供管理接口

### 配置参数
```bash
./http-gateway.exe -ipc <IPC地址> -http <HTTP监听地址>

参数说明：
  -ipc   Neo IPC服务器地址 (默认: localhost:9999)
  -http  HTTP监听地址 (默认: :8080)
```

### API 路由规则
- **路径格式**：`/api/{service}/{method}`
- **请求方法**：POST, GET, PUT, DELETE等
- **请求体**：JSON格式
- **响应体**：JSON格式

### 示例请求

#### 调用Python计算服务
```bash
curl -X POST http://localhost:8081/api/demo-service-python/calculate \
  -H "Content-Type: application/json" \
  -d '{"operation":"add","a":10,"b":20}'

响应：
{"result": 30, "operation": "add", "a": 10, "b": 20}
```

#### 获取网关自身信息
```bash
curl -X POST http://localhost:8081/api/http-gateway/getInfo \
  -H "Content-Type: application/json" \
  -d '{}'

响应：
{
  "service": "http-gateway",
  "type": "http-gateway",
  "httpAddress": ":8081",
  "status": "running",
  "version": "1.0.0"
}
```

### 源代码结构
```
services/http-gateway/
├── main.go         # 程序入口
├── service.go      # HTTP网关服务实现
├── ipc_client.go   # IPC客户端（自包含）
└── go.mod          # Go模块定义
```

## TCP 网关服务

### 概述
TCP网关提供原生TCP socket接入，支持JSON协议的消息交换，适用于需要持久连接的场景。

### 功能特性
- 持久TCP连接
- JSON消息协议（长度前缀）
- 连接池管理
- 心跳保活
- 并发连接处理
- 自身作为IPC服务提供管理接口

### 配置参数
```bash
./tcp-gateway.exe -ipc <IPC地址> -tcp <TCP监听地址> [-protocol <协议类型>]

参数说明：
  -ipc       Neo IPC服务器地址 (默认: localhost:9999)
  -tcp       TCP监听地址 (默认: :7777)
  -protocol  协议类型 json/binary (默认: json)
```

### TCP协议格式

#### 消息结构
```
[4字节长度(大端序)][JSON消息体]
```

#### 请求消息格式
```json
{
  "service": "目标服务名",
  "method": "方法名",
  "data": {
    // 请求参数
  }
}
```

#### 响应消息格式
```json
{
  "success": true/false,
  "data": {
    // 响应数据
  },
  "error": "错误信息（如果有）"
}
```

### 示例交互

#### 使用测试客户端
```bash
./test_client.exe localhost:7777
```

#### 编程接口示例（Go）
```go
// 连接TCP网关
conn, err := net.Dial("tcp", "localhost:7777")

// 构建请求
request := TCPMessage{
    Service: "demo-service-python",
    Method:  "calculate",
    Data: map[string]interface{}{
        "operation": "add",
        "a": 10,
        "b": 20,
    },
}

// 发送请求（带长度前缀）
data, _ := json.Marshal(request)
binary.Write(conn, binary.BigEndian, uint32(len(data)))
conn.Write(data)

// 读取响应
var respLen uint32
binary.Read(conn, binary.BigEndian, &respLen)
respData := make([]byte, respLen)
io.ReadFull(conn, respData)

var response TCPResponse
json.Unmarshal(respData, &response)
```

### 源代码结构
```
services/tcp-gateway/
├── main.go         # 程序入口
├── service.go      # TCP网关服务实现
├── ipc_client.go   # IPC客户端（自包含）
├── test_client.go  # 测试客户端
└── go.mod          # Go模块定义
```

## 网关服务的IPC接口

两个网关服务自身也注册为IPC服务，提供以下方法：

### HTTP网关IPC方法

#### getInfo
获取HTTP网关服务信息
```json
请求：{}
响应：{
  "service": "http-gateway",
  "type": "http-gateway",
  "httpAddress": ":8081",
  "status": "running",
  "version": "1.0.0"
}
```

### TCP网关IPC方法

#### getInfo
获取TCP网关服务信息
```json
请求：{}
响应：{
  "service": "tcp-gateway",
  "type": "tcp-gateway",
  "tcpAddress": ":7777",
  "protocol": "json",
  "connections": 5,
  "status": "running",
  "version": "1.0.0"
}
```

#### getStats
获取TCP网关统计信息
```json
请求：{}
响应：{
  "activeConnections": 5,
  "totalRequests": 1234,
  "protocol": "json"
}
```

## 部署建议

### 开发环境
```bash
# 编译所有服务
scripts/build/build-all.bat

# 启动所有服务
scripts/start-all.bat
```

### 生产环境

1. **独立部署**：每个网关可以独立部署在不同的服务器
2. **负载均衡**：可以部署多个网关实例并使用负载均衡器
3. **监控**：通过IPC接口监控网关状态
4. **日志**：每个网关有独立的日志输出

### 高可用配置

```
                    ┌──────────────┐
                    │Load Balancer │
                    └──────┬───────┘
                           │
        ┌──────────────────┼──────────────────┐
        ▼                  ▼                  ▼
  ┌────────────┐    ┌────────────┐    ┌────────────┐
  │HTTP Gateway│    │HTTP Gateway│    │HTTP Gateway│
  │  Instance 1│    │  Instance 2│    │  Instance 3│
  └─────┬──────┘    └─────┬──────┘    └─────┬──────┘
        │                 │                  │
        └─────────────────┼──────────────────┘
                          ▼
                    ┌──────────┐
                    │Neo Core  │
                    │IPC Server│
                    └──────────┘
```

## 扩展开发

### 添加新的网关服务

1. **创建服务目录**
   ```bash
   mkdir services/websocket-gateway
   ```

2. **实现IPC客户端**
   - 复制现有网关的 `ipc_client.go`
   - 根据需要调整

3. **实现协议转换**
   - 接收外部协议请求
   - 转换为IPC消息
   - 转发到目标服务
   - 将响应转换回外部协议

4. **注册为IPC服务**
   ```go
   client.RegisterService("websocket-gateway", metadata)
   ```

5. **更新构建脚本**
   在 `scripts/build/build-all.bat` 添加编译步骤

### 自定义协议支持

TCP网关预留了协议扩展接口，可以支持：
- 二进制协议
- Protobuf
- MessagePack
- 自定义协议

## 故障排除

### 常见问题

1. **网关无法连接到IPC服务器**
   - 检查Neo Core是否运行
   - 验证IPC端口（默认9999）
   - 检查防火墙设置

2. **服务未找到错误**
   - 确认目标服务已注册
   - 检查服务名称拼写
   - 查看Neo Core日志

3. **TCP连接断开**
   - 检查读取超时设置（默认5分钟）
   - 验证消息格式
   - 查看TCP网关日志

### 调试方法

1. **查看服务注册状态**
   ```bash
   curl -X POST http://localhost:8081/api/http-gateway/getInfo
   ```

2. **检查TCP连接数**
   ```bash
   curl -X POST http://localhost:8081/api/tcp-gateway/getStats
   ```

3. **查看服务日志**
   - Neo Core日志：查看IPC通信
   - 网关日志：查看请求转发
   - 服务日志：查看业务处理

## 性能优化

### HTTP网关
- 使用连接池
- 启用HTTP/2
- 调整超时参数
- 使用缓存

### TCP网关
- 调整缓冲区大小
- 优化消息序列化
- 使用连接池
- 实现批量处理

## 安全建议

1. **认证授权**：在网关层实现统一认证
2. **加密传输**：使用TLS/SSL
3. **限流熔断**：防止服务过载
4. **输入验证**：在网关层验证请求参数
5. **日志审计**：记录所有请求和响应