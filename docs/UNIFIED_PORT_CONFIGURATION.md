# Neo Framework 统一端口配置指南

版本：1.0  
最后更新：2025-08-05

## 🎯 核心原则：统一使用默认端口

为了避免端口混乱和配置不一致的问题，Neo Framework 在所有环境中统一使用以下默认端口：

| 服务 | 默认端口 | 用途 |
|------|----------|------|
| **HTTP Gateway** | **8080** | HTTP API 网关，接收外部请求 |
| **IPC Server** | **9999** | 内部进程通信，服务注册和消息转发 |

## ⚠️ 重要提醒

### 为什么要统一端口？

1. **避免混乱**：历史上存在多个端口配置（如 28080, 29999），导致测试失败和调试困难
2. **简化部署**：所有环境使用相同端口，减少配置错误
3. **提高效率**：开发者不需要记住多套端口配置

### 已验证的配置

✅ 所有语言示例已更新为使用默认端口：
- Python: `os.getenv('NEO_IPC_PORT', '9999')`
- Go: `os.Getenv("NEO_IPC_PORT")` 默认 "9999"
- Node.js: `process.env.NEO_IPC_PORT || 9999`
- Java: `System.getenv("NEO_IPC_PORT")` 默认 "9999"
- PHP: `getenv('NEO_IPC_PORT') ?: 9999`

## 📋 配置方式（优先级从高到低）

### 1. 命令行参数（最高优先级）
```bash
./neo.exe -http :8080 -ipc :9999
```

### 2. 环境变量
```bash
# Linux/Mac
export NEO_HTTP_PORT=8080
export NEO_IPC_PORT=9999
export NEO_IPC_HOST=localhost

# Windows
set NEO_HTTP_PORT=8080
set NEO_IPC_PORT=9999
set NEO_IPC_HOST=localhost
```

### 3. 配置文件（最低优先级）
```yaml
# configs/default.yml
gateway:
  address: ":8080"
ipc:
  address: ":9999"
```

## 🚀 快速启动指南

### 标准启动流程

1. **启动 Neo Framework**
```bash
# 直接运行（使用默认端口）
go run cmd/neo/main.go

# 或者构建后运行
go build -o neo.exe cmd/neo/main.go
./neo.exe
```

2. **确认启动成功**
```
=== Neo Framework ===
HTTP网关: http://localhost:8080
IPC服务器: localhost:9999
健康检查: http://localhost:8080/health
```

3. **启动语言服务**
```bash
# Python
cd examples-ipc/python && python service.py

# Go
cd examples-ipc/go && go run service.go

# Node.js
cd examples-ipc/nodejs && node service.js

# Java
cd examples-ipc/java && java -cp .:gson-2.10.1.jar Service

# PHP
cd examples-ipc/php && php service.php
```

## 🔧 特殊场景配置

### 场景1：端口被占用

如果默认端口被占用，通过环境变量统一修改：

```bash
# 设置新端口
export NEO_HTTP_PORT=8888
export NEO_IPC_PORT=9998

# 启动 Neo
./neo.exe

# 启动服务（自动使用新端口）
python examples-ipc/python/service.py
```

### 场景2：多实例部署

运行多个 Neo 实例时：

```bash
# 实例1（默认）
NEO_HTTP_PORT=8080 NEO_IPC_PORT=9999 ./neo.exe

# 实例2
NEO_HTTP_PORT=8081 NEO_IPC_PORT=9998 ./neo.exe

# 实例3
NEO_HTTP_PORT=8082 NEO_IPC_PORT=9997 ./neo.exe
```

### 场景3：Docker 部署

```dockerfile
# Dockerfile
ENV NEO_HTTP_PORT=8080
ENV NEO_IPC_PORT=9999
EXPOSE 8080 9999
```

```yaml
# docker-compose.yml
services:
  neo:
    environment:
      - NEO_HTTP_PORT=8080
      - NEO_IPC_PORT=9999
    ports:
      - "8080:8080"
      - "9999:9999"
```

## 📊 端口使用检查

### Windows
```cmd
# 检查端口占用
netstat -ano | findstr :8080
netstat -ano | findstr :9999

# 查看占用进程
tasklist | findstr <PID>
```

### Linux/Mac
```bash
# 检查端口占用
lsof -i :8080
lsof -i :9999

# 或使用 netstat
netstat -tulpn | grep :8080
netstat -tulpn | grep :9999
```

## ❌ 避免的做法

1. **不要硬编码端口**
```python
# ❌ 错误
client = NeoIPCClient("localhost", 29999)

# ✅ 正确
port = os.getenv('NEO_IPC_PORT', '9999')
client = NeoIPCClient("localhost", int(port))
```

2. **不要使用旧端口**
```bash
# ❌ 错误（旧端口）
./neo.exe -http :28080 -ipc :29999

# ✅ 正确（统一端口）
./neo.exe -http :8080 -ipc :9999
```

3. **不要混用配置方式**
```bash
# ❌ 错误（配置文件用 9999，环境变量用 29999）
# configs/default.yml: ipc.port: 9999
export NEO_IPC_PORT=29999

# ✅ 正确（统一配置）
# configs/default.yml: ipc.port: 9999
export NEO_IPC_PORT=9999
```

## 🛠️ 故障排查

### 问题1：服务连接失败

**症状**：`connection refused` 或 `timeout`

**解决步骤**：
1. 确认 Neo Framework 正在运行
2. 检查端口配置是否一致
3. 验证防火墙设置

### 问题2：端口冲突

**症状**：`bind: address already in use`

**解决步骤**：
1. 使用端口检查命令找出占用进程
2. 终止占用进程或更换端口
3. 通过环境变量统一设置新端口

### 问题3：服务注册失败

**症状**：服务启动但无法处理请求

**解决步骤**：
1. 确认使用相同的 IPC 端口
2. 检查服务名是否正确
3. 查看 Neo Framework 日志

## 📚 相关文档

- [端口管理指南](./PORT_MANAGEMENT.md) - 详细的端口管理和自动化工具
- [IPC 协议指南](./IPC_PROTOCOL_GUIDE.md) - IPC 通信协议详解
- [测试手册](./TEST_MANUAL.md) - 测试和故障排查
- [配置文件说明](../configs/README.md) - 配置文件详解

## 🎓 最佳实践总结

1. **始终使用默认端口**：8080 (HTTP) 和 9999 (IPC)
2. **通过环境变量管理**：便于容器化和自动化部署
3. **避免硬编码**：使用配置文件或环境变量
4. **保持一致性**：所有服务使用相同的端口配置
5. **文档化特殊配置**：如果必须使用非默认端口，请记录原因

---

*文档编写：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*

记住：**统一配置，减少混乱**！