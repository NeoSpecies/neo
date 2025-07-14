# Neo Framework Python服务测试指南

本文档提供了完整的Python服务测试方案，包括环境准备、服务启动、功能测试和问题排查。

## 测试环境准备

### 1. 确认依赖安装

```bash
# 检查Python版本（需要3.8+）
python --version

# 安装必要的Python包
pip install aiohttp asyncio
```

### 2. 确认端口可用

```bash
# Windows
netstat -an | findstr :28080
netstat -an | findstr :29999

# 如果端口被占用，找到并结束相关进程
```

## 测试步骤

### 第一步：启动Neo Framework

打开第一个终端窗口：

```bash
# 进入项目目录
cd C:\Dev\neo

# 使用启动脚本（Windows）
start.bat

# 或直接运行
go run cmd/neo/main.go
```

**预期输出**：
```
=== Neo Framework ===
A high-performance microservice communication framework

HTTP网关: http://localhost:28080
IPC服务器: localhost:29999
健康检查: http://localhost:28080/health
API接口: http://localhost:28080/api/{service}/{method}

按 Ctrl+C 停止服务
```

### 第二步：验证框架健康状态

打开第二个终端窗口：

```bash
# 测试健康检查端点
curl http://localhost:28080/health
```

**预期响应**：
```json
{"status":"healthy","time":"2025-01-14T12:00:00+08:00"}
```

### 第三步：启动Python服务

在第二个终端窗口中：

```bash
# 进入Python服务目录
cd C:\Dev\neo\python_service

# 启动示例服务
python example_service.py
```

**预期输出**：
```
INFO:__main__:Starting Python Math Service...
INFO:neo_client:Connected to Neo IPC server at localhost:29999
INFO:neo_client:Service 'python.math' registered
INFO:neo_client:Handler registered for method: add
INFO:neo_client:Handler registered for method: multiply
INFO:neo_client:Handler registered for method: calculate
INFO:neo_client:Starting to listen for messages...
```

### 第四步：功能测试

打开第三个终端窗口进行测试：

#### 4.1 测试加法功能

```bash
curl -X POST http://localhost:28080/api/python.math/add \
  -H "Content-Type: application/json" \
  -d "{\"a\": 10, \"b\": 20}"
```

**预期响应**：
```json
{"result": 30}
```

#### 4.2 测试乘法功能

```bash
curl -X POST http://localhost:28080/api/python.math/multiply \
  -H "Content-Type: application/json" \
  -d "{\"a\": 7, \"b\": 8}"
```

**预期响应**：
```json
{"result": 56}
```

#### 4.3 测试表达式计算

```bash
curl -X POST http://localhost:28080/api/python.math/calculate \
  -H "Content-Type: application/json" \
  -d "{\"expression\": \"2 * (3 + 4)\"}"
```

**预期响应**：
```json
{"result": 14}
```

#### 4.4 测试错误处理

测试不存在的方法：
```bash
curl -X POST http://localhost:28080/api/python.math/divide \
  -H "Content-Type: application/json" \
  -d "{\"a\": 10, \"b\": 2}"
```

**预期响应**：
```json
{"error": "Method 'divide' not found"}
```

测试不存在的服务：
```bash
curl -X POST http://localhost:28080/api/python.unknown/test \
  -H "Content-Type: application/json" \
  -d "{}"
```

**预期响应**：
HTTP 404 错误

### 第五步：并发测试

使用Python测试脚本进行并发测试：

```bash
cd C:\Dev\neo
python test/python/test_simple_client.py
```

或使用压力测试：

```bash
python test/stress/test_stress.py
```

### 第六步：监控日志

在测试过程中，可以查看日志了解详细信息：

```bash
# 查看Neo框架日志
tail -f logs/neo_*.log

# 查看Python服务日志
tail -f logs/python_*.log
```

## 测试检查清单

请按顺序验证以下项目：

- [ ] Neo Framework成功启动
- [ ] 健康检查返回正确响应
- [ ] Python服务成功连接并注册
- [ ] 加法功能正常工作
- [ ] 乘法功能正常工作
- [ ] 表达式计算功能正常工作
- [ ] 错误处理返回合适的错误信息
- [ ] 并发请求能够正确处理
- [ ] 服务在压力测试下保持稳定

## 常见问题及解决方案

### 1. 端口被占用

**错误信息**：
```
failed to listen: listen tcp :28080: bind: Only one usage of each socket address
```

**解决方案**：
- 查找占用端口的进程并结束
- 或使用不同端口：`go run cmd/neo/main.go -http :30080 -ipc :30999`

### 2. Python服务无法连接

**错误信息**：
```
ConnectionRefusedError: [WinError 10061] No connection could be made
```

**解决方案**：
- 确保Neo Framework已启动
- 检查防火墙设置
- 确认端口号正确（29999）

### 3. 服务发现失败

**错误信息**：
```
no service instances found
```

**解决方案**：
- 确保Python服务已启动并注册成功
- 检查服务名称是否正确（python.math）
- 查看Neo Framework日志确认注册信息

### 4. 请求超时

**可能原因**：
- Python服务处理时间过长
- 网络延迟
- 服务健康检查失败

**解决方案**：
- 检查Python服务是否正常运行
- 查看Python服务控制台是否有错误输出
- 确认健康检查超时时间设置（当前为5分钟）

## 高级测试

### 1. 自定义服务测试

创建新的Python服务进行测试：

```python
# my_service.py
import asyncio
from neo_client import NeoIPCClient

class MyService:
    def __init__(self):
        self.client = NeoIPCClient(port=29999)
    
    async def handle_echo(self, data):
        return {"echo": data.get("message", "")}
    
    async def start(self):
        await self.client.connect()
        await self.client.register_service("my.echo", {"version": "1.0"})
        self.client.register_handler("echo", self.handle_echo)
        await self.client.listen()

if __name__ == "__main__":
    service = MyService()
    asyncio.run(service.start())
```

测试：
```bash
curl -X POST http://localhost:28080/api/my.echo/echo \
  -H "Content-Type: application/json" \
  -d "{\"message\": \"Hello Neo!\"}"
```

### 2. 性能基准测试

使用Apache Bench进行性能测试：

```bash
# 1000个请求，10个并发
ab -n 1000 -c 10 -p data.json -T application/json http://localhost:28080/api/python.math/add
```

data.json内容：
```json
{"a": 5, "b": 3}
```

### 3. 长时间运行测试

让服务运行较长时间，观察：
- 内存使用情况
- CPU使用率
- 响应时间变化
- 是否有内存泄漏

## 测试报告模板

测试完成后，请记录以下信息：

```
测试日期：2025-01-14
测试环境：Windows 11 / Go 1.19 / Python 3.11

功能测试结果：
- [ ] 健康检查：通过/失败
- [ ] 加法功能：通过/失败
- [ ] 乘法功能：通过/失败
- [ ] 表达式计算：通过/失败
- [ ] 错误处理：通过/失败

性能测试结果：
- 平均响应时间：___ ms
- 最大响应时间：___ ms
- QPS：___
- 错误率：___%

问题记录：
1. 问题描述：
   解决方案：

2. 问题描述：
   解决方案：

改进建议：
1. 
2. 
```

## 下一步

测试完成后，你可以：

1. **实现心跳机制**：在Python客户端添加定期心跳，保持服务健康状态
2. **添加更多服务**：创建其他类型的Python服务
3. **优化性能**：调整连接池、并发数等参数
4. **部署到生产**：使用Docker或systemd进行部署

祝测试顺利！如有任何问题，请查看日志文件或参考故障排查部分。