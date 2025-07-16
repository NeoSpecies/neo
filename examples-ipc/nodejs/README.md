# Node.js IPC 客户端示例

这是一个 Node.js 的 IPC 客户端示例，展示如何连接到 Neo Framework。

## 运行要求

- Node.js >= 12.0
- 无需额外依赖（仅使用内置模块）

## 快速开始

```bash
# 直接运行
node service.js

# 或添加执行权限后运行（Linux/Mac）
chmod +x service.js
./service.js
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
  -d '{"name": "Node.js"}'

# 测试计算
curl -X POST http://localhost:8080/api/demo-service/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 20, "b": 4, "operation": "divide"}'

# 测试回显
curl -X POST http://localhost:8080/api/demo-service/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from Node.js!"}'

# 获取时间
curl -X POST http://localhost:8080/api/demo-service/getTime \
  -H "Content-Type: application/json" \
  -d '{"format": "readable"}'

# 获取服务信息
curl -X POST http://localhost:8080/api/demo-service/getInfo
```

## 代码结构

- `service.js` - 主服务文件，包含：
  - `NeoIPCClient` - IPC 客户端类
  - 二进制协议编解码
  - 异步处理器支持

## 扩展开发

1. 添加新的处理器：
```javascript
client.addHandler('myMethod', async (params) => {
    // 你的业务逻辑
    return { result: 'success' };
});
```

2. 修改服务名称：
```javascript
await client.registerService('my-service-name', {
    version: '1.0.0'
});
```

## 特点

- 使用 Node.js 内置的 `net` 模块
- 原生支持异步处理
- 适合 I/O 密集型任务
- 事件驱动架构

## 注意事项

- 使用小端序（Little Endian）进行二进制编码
- 心跳间隔为 30 秒
- 支持 Promise/async-await 模式