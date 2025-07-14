# Neo Framework 测试指南

## 自动化测试

运行完整的测试套件：
```bash
python test_framework.py
```

这个测试脚本会：
1. 启动 Neo Framework 主应用程序（推荐）
2. 启动 Python 测试服务并注册到 IPC 服务器
3. 执行多个 HTTP 请求测试
4. 自动清理进程和文件

## 手动测试步骤

### 1. 启动服务器

**方法1：启动主应用程序（推荐）**
```bash
# 在项目根目录下
go run cmd/neo/main.go
```

**方法2：启动独立网关**
```bash
go run cmd/gateway/main.go
```

你应该看到输出：
```
=== Neo Framework ===
HTTP网关: http://localhost:28080
IPC服务器: localhost:29999
健康检查: http://localhost:28080/health
```

### 2. 启动 Python 服务

在另一个终端：
```bash
# 创建一个简单的 Python 服务
python python_service/example_service.py
```

或使用提供的测试服务：
```bash
python test_math_service.py
```

### 3. 发送测试请求

使用 curl 测试：

```bash
# 健康检查
curl http://localhost:28080/health

# 数学运算 - 加法
curl -X POST http://localhost:28080/api/test.math/add \
  -H "Content-Type: application/json" \
  -d '{"a": 5, "b": 3}'

# 数学运算 - 乘法
curl -X POST http://localhost:28080/api/test.math/multiply \
  -H "Content-Type: application/json" \
  -d '{"a": 4, "b": 7}'

# 数学运算 - 除法
curl -X POST http://localhost:28080/api/test.math/divide \
  -H "Content-Type: application/json" \
  -d '{"a": 10, "b": 2}'
```

### 4. 使用 Python 脚本测试

```python
import requests

# 测试加法
response = requests.post(
    "http://localhost:28080/api/test.math/add",
    json={"a": 10, "b": 20}
)
print(f"加法结果: {response.json()}")

# 测试乘法
response = requests.post(
    "http://localhost:28080/api/test.math/multiply",
    json={"a": 5, "b": 6}
)
print(f"乘法结果: {response.json()}")
```

## 测试流程说明

1. **HTTP 请求**：客户端发送 POST 请求到 `/api/{service}/{method}`
2. **Gateway 处理**：Go Gateway 接收请求，解析 service 和 method
3. **IPC 转发**：通过 AsyncService 将请求转发到 IPC Server
4. **Python 处理**：IPC Server 找到对应的 Python 服务连接，发送请求
5. **响应返回**：Python 服务处理完成后，响应原路返回

## 调试技巧

1. **查看 Gateway 日志**：观察 Go 端的请求处理过程
2. **查看 Python 服务日志**：观察 Python 端的消息接收和处理
3. **使用 Wireshark**：监控 29999 端口的 TCP 通信
4. **检查服务注册**：确保 Python 服务成功注册到 IPC Server

## 常见问题

1. **端口占用**：确保 28080 和 29999 端口未被占用（主应用）或 18080/19999（独立网关）
2. **Python 路径**：确保 python_service 目录在 Python 路径中
3. **服务名称**：请求的服务名必须与注册的服务名完全匹配