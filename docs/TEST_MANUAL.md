# Neo Framework 测试指南

**最后更新：2025-08-05**

## 目录
1. [快速开始](#快速开始)
2. [自动化测试](#自动化测试)
3. [手动测试步骤](#手动测试步骤)
4. [故障排查指南](#故障排查指南)
5. [常见问题](#常见问题)
6. [测试流程说明](#测试流程说明)
7. [调试技巧](#调试技巧)

## 快速开始

### 前置条件
- Go 1.16+ 已安装
- Python 3.7+ 已安装（测试 Python 服务）
- Node.js 14+ 已安装（测试 Node.js 服务）
- Java 8+ 已安装（测试 Java 服务）
- PHP 7.4+ 已安装（测试 PHP 服务，需要启用 sockets 扩展）
- 确保端口 8080（HTTP网关）和 9999（IPC服务器）未被占用

### 默认端口配置
| 服务 | 默认端口 | 说明 |
|------|----------|------|
| **HTTP Gateway** | **8080** | Neo Framework HTTP API 网关 |
| **IPC Server** | **9999** | Neo Framework IPC 通信端口 |

**注意**：所有示例代码都已配置为使用这些默认端口。如需修改，请通过环境变量 `NEO_IPC_PORT` 和配置文件统一调整。

## 自动化测试

### 单元测试
运行所有单元测试：
```bash
go test ./internal/...
```

运行特定包的测试：
```bash
go test ./internal/config -v
go test ./internal/transport -v
```

### 集成测试
运行集成测试：
```bash
go test ./test/integration -v
```

### Python 测试脚本
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
HTTP网关: http://localhost:8080
IPC服务器: localhost:9999
健康检查: http://localhost:8080/health
```

### 2. 启动服务示例

在另一个终端，选择要测试的语言：

**Python 服务：**
```bash
cd examples-ipc/python
python service.py
# 服务将注册为 demo-service-python
```

**Go 服务：**
```bash
cd examples-ipc/go
go run service.go
# 服务将注册为 demo-service-go
```

**Node.js 服务：**
```bash
cd examples-ipc/nodejs
node service.js
# 服务将注册为 demo-service-nodejs
```

**Java 服务：**
```bash
cd examples-ipc/java
# 需要先下载 Gson 库
# 下载地址：https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar
javac -cp gson-2.10.1.jar Service.java
java -cp .;gson-2.10.1.jar Service  # Windows
# java -cp .:gson-2.10.1.jar Service  # Linux/Mac
# 服务将注册为 demo-service-java
```

**PHP 服务：**
```bash
cd examples-ipc/php
# 首先检查环境
php check_env.php
# 如果提示sockets扩展未启用，需要在php.ini中启用：extension=sockets
# 如果环境检查通过，运行服务
php service.php
# 服务将注册为 demo-service-php
```

### 3. 发送测试请求

使用 curl 测试：

```bash
# 健康检查
curl http://localhost:8080/health

# Hello 方法
# 根据运行的服务选择对应的服务名
# Python: demo-service-python
# Go: demo-service-go
# Node.js: demo-service-nodejs
# Java: demo-service-java
curl -X POST http://localhost:8080/api/demo-service-python/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo Framework"}'

# 计算方法
curl -X POST http://localhost:8080/api/demo-service-python/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 10, "b": 5, "operation": "multiply"}'

# Echo 方法
curl -X POST http://localhost:8080/api/demo-service-python/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Testing Neo Framework"}'

# 获取时间
curl -X POST http://localhost:8080/api/demo-service-python/getTime \
  -H "Content-Type: application/json" \
  -d '{"format": "readable"}'

# 获取服务信息
curl -X POST http://localhost:8080/api/demo-service-python/getInfo \
  -H "Content-Type: application/json" \
  -d '{}'
```

### 4. 使用 Python 脚本测试

```python
import requests

# 测试 Hello
response = requests.post(
    "http://localhost:8080/api/demo-service-python/hello",
    json={"name": "Python Test"}
)
print(f"Hello结果: {response.json()}")

# 测试计算
response = requests.post(
    "http://localhost:8080/api/demo-service-python/calculate",
    json={"a": 10, "b": 5, "operation": "add"}
)
print(f"计算结果: {response.json()}")

# 测试 Echo
response = requests.post(
    "http://localhost:8080/api/demo-service-python/echo",
    json={"message": "Hello World"}
)
print(f"Echo结果: {response.json()}")
```

## 测试流程说明

1. **HTTP 请求**：客户端发送 POST 请求到 `/api/{service}/{method}`
2. **Gateway 处理**：Go Gateway 接收请求，解析 service 和 method
3. **IPC 转发**：通过 AsyncService 将请求转发到 IPC Server
4. **Python 处理**：IPC Server 找到对应的 Python 服务连接，发送请求
5. **响应返回**：Python 服务处理完成后，响应原路返回

## 调试技巧

1. **查看 Gateway 日志**：观察 Go 端的请求处理过程
2. **查看服务日志**：观察各语言服务的消息接收和处理
3. **使用 Wireshark**：监控 9999 端口的 TCP 通信
4. **检查服务注册**：确保服务成功注册到 IPC Server

## 并发测试

现在各服务使用不同的名称，可以同时运行多个语言的服务：

### 启动所有服务
```bash
# 终端 1 - 启动 Gateway
go run cmd/neo/main.go

# 终端 2 - Python 服务
cd examples-ipc/python && python service.py

# 终端 3 - Go 服务  
cd examples-ipc/go && go run service.go

# 终端 4 - Node.js 服务
cd examples-ipc/nodejs && node service.js

# 终端 5 - Java 服务（需要先下载 Gson）
cd examples-ipc/java
javac -cp gson-2.10.1.jar Service.java
java -cp .;gson-2.10.1.jar Service  # Windows
# java -cp .:gson-2.10.1.jar Service  # Linux/Mac

# 终端 6 - PHP 服务
cd examples-ipc/php && php service.php
```

### 测试所有服务
```bash
# 测试 Python 服务
curl -X POST http://localhost:8080/api/demo-service-python/hello -H "Content-Type: application/json" -d '{"name": "Python"}'

# 测试 Go 服务
curl -X POST http://localhost:8080/api/demo-service-go/hello -H "Content-Type: application/json" -d '{"name": "Go"}'

# 测试 Node.js 服务
curl -X POST http://localhost:8080/api/demo-service-nodejs/hello -H "Content-Type: application/json" -d '{"name": "Node.js"}'

# 测试 Java 服务
curl -X POST http://localhost:8080/api/demo-service-java/hello -H "Content-Type: application/json" -d '{"name": "Java"}'

# 测试 PHP 服务
curl -X POST http://localhost:8080/api/demo-service-php/hello -H "Content-Type: application/json" -d '{"name": "PHP"}'
```

## 故障排查指南

### 1. 配置文件错误
**错误信息：**
```
Failed to load config {error=failed to parse YAML: time: missing unit in duration "100"}
```
**解决方法：**
检查 `configs/default.yml` 中的时间配置，确保所有时间值都带有单位（如 `100ms`、`30s`）

### 2. 端口被占用
**错误信息：**
```
failed to listen: listen tcp :9999: bind: Only one usage of each socket address
```
**解决方法：**
```bash
# Windows - 查看端口占用
netstat -ano | findstr :9999
# 停止占用进程
taskkill /PID <进程ID> /F

# Linux/Mac
lsof -i :9999
kill -9 <进程ID>
```

### 3. 服务连接失败
**错误信息：**
```
dial tcp [::1]:9999: connectex: No connection could be made
```
**可能原因：**
- Gateway 未启动
- 服务使用了错误的端口（确保使用 9999）
- 防火墙阻止连接

### 4. 404 错误
**错误信息：**
```
HTTP/1.1 404 Not Found
```
**可能原因：**
- URL 路径错误（应为 `/api/服务名/方法名`）
- 服务未注册或服务名不匹配
- Gateway 运行在错误的端口（应为 8080）

### 5. Java 编译错误
**错误信息：**
```
error: package com.google.gson does not exist
```
**解决方法：**
下载 Gson 库并添加到 classpath（见 Java 服务启动说明）

### 6. PHP sockets 扩展错误
**错误信息：**
```
PHP Fatal error: Uncaught Error: Call to undefined function socket_create()
```
**解决方法：**
1. 编辑 php.ini 文件
2. 找到 `;extension=sockets`，去掉前面的分号
3. 重启 PHP 或命令行
4. 运行 `php -m | grep sockets` 确认扩展已加载

## 常见问题

1. **端口占用**：确保 8080（HTTP网关）和 9999（IPC服务器）端口未被占用
2. **服务命名**：各语言服务已使用不同名称（demo-service-python、demo-service-go、demo-service-nodejs、demo-service-java、demo-service-php），可同时运行
3. **服务名称**：请求的服务名必须与注册的服务名完全匹配
4. **环境变量**：可通过 `NEO_IPC_HOST` 和 `NEO_IPC_PORT` 环境变量自定义连接配置
5. **PHP配置**：PHP服务需要启用sockets扩展，Windows下extension_dir需要正确设置

## 性能测试

可以使用 Apache Bench (ab) 或 wrk 进行压力测试：

```bash
# 使用 ab 测试
ab -n 1000 -c 10 -p data.json -T application/json http://localhost:8080/api/demo-service-go/hello

# 使用 wrk 测试
wrk -t4 -c100 -d30s -s post.lua http://localhost:8080/api/demo-service-go/hello
```

## 完整测试报告

运行完整的测试套件生成详细报告：

```bash
python test_all_languages.py
```

这将测试所有语言的服务并生成 `test_report.md` 文件。

---

*文档编写：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*