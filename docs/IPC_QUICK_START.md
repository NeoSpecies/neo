# Neo Framework IPC 快速入门

本指南将帮助你在 5 分钟内将你的服务接入 Neo Framework。

## 前置要求

- Neo Framework 已启动并运行
- IPC Server 监听端口（默认 9999）
- 你选择的编程语言环境

## 快速开始步骤

### 1. 确认 Neo Framework 运行状态

```bash
# 检查服务是否运行
curl http://localhost:8080/health

# 检查 IPC 端口
telnet localhost 9999
```

### 2. 选择你的语言并创建客户端

#### Python 快速示例

创建文件 `my_service.py`：

```python
import asyncio
import json
from neo_ipc_client import NeoIPCClient  # 使用项目提供的客户端库

async def main():
    # 创建客户端
    client = NeoIPCClient(host="localhost", port=9999)
    
    # 定义服务处理函数
    @client.handler("greet")
    async def greet(params):
        name = params.get("name", "Guest")
        return {"message": f"Hello, {name}! Welcome to Neo Framework."}
    
    @client.handler("calculate")
    async def calculate(params):
        a = params.get("a", 0)
        b = params.get("b", 0)
        operation = params.get("operation", "add")
        
        if operation == "add":
            return {"result": a + b}
        elif operation == "multiply":
            return {"result": a * b}
        else:
            return {"error": "Unknown operation"}
    
    # 连接并注册服务
    await client.connect()
    await client.register_service("python-demo", {
        "version": "1.0.0",
        "author": "demo"
    })
    
    print("Service 'python-demo' is running...")
    
    # 开始处理请求
    await client.start()

if __name__ == "__main__":
    asyncio.run(main())
```

运行服务：
```bash
python my_service.py
```

#### Go 快速示例

创建文件 `main.go`：

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    
    "github.com/neo-framework/neo-ipc-go" // 假设的客户端包
)

func main() {
    // 创建客户端
    client, err := ipc.NewClient("localhost:9999")
    if err != nil {
        log.Fatal(err)
    }
    
    // 添加处理器
    client.HandleFunc("greet", func(params map[string]interface{}) (interface{}, error) {
        name, ok := params["name"].(string)
        if !ok {
            name = "Guest"
        }
        
        return map[string]string{
            "message": fmt.Sprintf("Hello, %s! Welcome to Neo Framework.", name),
        }, nil
    })
    
    // 注册服务
    err = client.RegisterService("go-demo", map[string]string{
        "version": "1.0.0",
        "author": "demo",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    log.Println("Service 'go-demo' is running...")
    
    // 开始服务
    client.Start()
}
```

运行服务：
```bash
go run main.go
```

### 3. 测试你的服务

通过 HTTP 网关调用你的服务：

```bash
# 调用 greet 方法
curl -X POST http://localhost:8080/api/python-demo/greet \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo"}'

# 响应：
# {"message": "Hello, Neo! Welcome to Neo Framework."}

# 调用 calculate 方法
curl -X POST http://localhost:8080/api/python-demo/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 10, "b": 20, "operation": "add"}'

# 响应：
# {"result": 30}
```

## 完整项目结构示例

### Python 项目

```
my-python-service/
├── requirements.txt
├── config.py
├── service.py
├── handlers/
│   ├── __init__.py
│   ├── user.py
│   └── order.py
└── main.py
```

`requirements.txt`:
```
asyncio
json
struct
```

`main.py`:
```python
import asyncio
from neo_ipc_client import NeoIPCClient
from handlers import user, order

async def main():
    client = NeoIPCClient()
    
    # 注册所有处理器
    user.register_handlers(client)
    order.register_handlers(client)
    
    # 启动服务
    await client.connect()
    await client.register_service("business-service")
    await client.start()

if __name__ == "__main__":
    asyncio.run(main())
```

### Node.js 项目

```
my-node-service/
├── package.json
├── index.js
├── lib/
│   └── neo-client.js
└── handlers/
    ├── user.js
    └── order.js
```

`package.json`:
```json
{
  "name": "my-node-service",
  "version": "1.0.0",
  "main": "index.js",
  "scripts": {
    "start": "node index.js"
  }
}
```

`index.js`:
```javascript
const NeoIPCClient = require('./lib/neo-client');
const userHandlers = require('./handlers/user');
const orderHandlers = require('./handlers/order');

async function main() {
    const client = new NeoIPCClient();
    
    // 注册处理器
    userHandlers.register(client);
    orderHandlers.register(client);
    
    // 启动服务
    await client.connect();
    await client.registerService('node-service');
    await client.start();
    
    console.log('Service is running...');
}

main().catch(console.error);
```

## 生产环境部署

### 1. 使用环境变量配置

```python
import os

NEO_IPC_HOST = os.getenv('NEO_IPC_HOST', 'localhost')
NEO_IPC_PORT = int(os.getenv('NEO_IPC_PORT', '9999'))
SERVICE_NAME = os.getenv('SERVICE_NAME', 'my-service')

client = NeoIPCClient(host=NEO_IPC_HOST, port=NEO_IPC_PORT)
```

### 2. 添加健康检查

```python
@client.handler("health")
async def health_check(params):
    return {
        "status": "healthy",
        "version": "1.0.0",
        "uptime": get_uptime()
    }
```

### 3. 实现优雅关闭

```python
import signal

def signal_handler(signum, frame):
    print("Shutting down gracefully...")
    client.close()
    sys.exit(0)

signal.signal(signal.SIGINT, signal_handler)
signal.signal(signal.SIGTERM, signal_handler)
```

### 4. Docker 部署示例

`Dockerfile`:
```dockerfile
FROM python:3.9-slim

WORKDIR /app

COPY requirements.txt .
RUN pip install -r requirements.txt

COPY . .

ENV NEO_IPC_HOST=neo-framework
ENV NEO_IPC_PORT=9999

CMD ["python", "main.py"]
```

`docker-compose.yml`:
```yaml
version: '3.8'

services:
  neo-framework:
    image: neo-framework:latest
    ports:
      - "8080:8080"
      - "9999:9999"
  
  my-service:
    build: .
    depends_on:
      - neo-framework
    environment:
      - NEO_IPC_HOST=neo-framework
      - NEO_IPC_PORT=9999
      - SERVICE_NAME=my-service
```

## 监控和日志

### 添加日志记录

```python
import logging

logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

@client.handler("process")
async def process(params):
    logger.info(f"Processing request: {params}")
    try:
        result = await do_processing(params)
        logger.info(f"Processing completed: {result}")
        return result
    except Exception as e:
        logger.error(f"Processing failed: {e}")
        raise
```

### 添加指标收集

```python
from prometheus_client import Counter, Histogram

request_count = Counter('service_requests_total', 'Total requests')
request_duration = Histogram('service_request_duration_seconds', 'Request duration')

@client.handler("process")
@request_duration.time()
async def process(params):
    request_count.inc()
    # 处理逻辑
    return result
```

## 故障排查

### 常见问题和解决方案

1. **连接失败**
   ```bash
   # 检查 Neo Framework 是否运行
   ps aux | grep neo
   
   # 检查端口是否开放
   netstat -tunlp | grep 9999
   ```

2. **服务注册失败**
   - 检查服务名是否唯一
   - 确认消息格式正确
   - 查看服务端日志

3. **请求超时**
   - 增加处理器的超时时间
   - 检查是否有阻塞操作
   - 优化处理逻辑

4. **内存泄漏**
   - 确保正确关闭连接
   - 清理未使用的资源
   - 使用内存分析工具

## 下一步

- 阅读[完整协议文档](./IPC_PROTOCOL_GUIDE.md)了解更多细节
- 查看[示例项目](../examples/)获取更多参考
- 参与[社区讨论](https://github.com/neo-framework/discussions)

恭喜！你已经成功将服务接入 Neo Framework。