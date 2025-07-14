# Neo Framework Python服务测试指南（更新版）

由于默认端口可能被占用，本指南提供了使用自定义端口的测试方案。

## 快速测试步骤

### 第一步：使用自定义端口启动Neo Framework

打开第一个PowerShell或命令提示符窗口：

```bash
# 进入项目目录
cd C:\Dev\neo

# 使用自定义端口启动（选择未被占用的端口）
go run cmd/neo/main.go -http :48080 -ipc :49999
```

如果这些端口仍被占用，可以尝试其他端口组合：
- `-http :58080 -ipc :59999`
- `-http :38080 -ipc :39999`
- `-http :18080 -ipc :19999`

**预期输出**：
```
=== Neo Framework ===
A high-performance microservice communication framework

HTTP网关: http://localhost:48080
IPC服务器: localhost:49999
健康检查: http://localhost:48080/health
API接口: http://localhost:48080/api/{service}/{method}

按 Ctrl+C 停止服务
```

### 第二步：验证框架健康状态

打开第二个终端窗口：

```bash
# 测试健康检查端点（注意端口号）
curl http://localhost:48080/health
```

**预期响应**：
```json
{"status":"healthy","time":"2025-01-14T12:00:00+08:00"}
```

### 第三步：更新Python服务端口并启动

首先需要修改Python服务的端口配置。在第二个终端窗口中：

```bash
# 进入Python服务目录
cd C:\Dev\neo\python_service

# 修改example_service.py的端口（使用文本编辑器）
# 找到这一行：
#     self.client = NeoIPCClient(port=29999)
# 改为：
#     self.client = NeoIPCClient(port=49999)

# 或者使用PowerShell命令直接修改
(Get-Content example_service.py) -replace 'port=\d+', 'port=49999' | Set-Content example_service.py

# 启动Python服务
python example_service.py
```

**预期输出**：
```
INFO:__main__:Starting Python Math Service...
INFO:neo_client:Connected to Neo IPC server at localhost:49999
INFO:neo_client:Service 'python.math' registered
INFO:neo_client:Handler registered for method: add
INFO:neo_client:Handler registered for method: multiply
INFO:neo_client:Handler registered for method: calculate
INFO:neo_client:Starting to listen for messages...
```

### 第四步：执行功能测试

打开第三个终端窗口进行测试：

#### 测试加法功能

```bash
curl -X POST http://localhost:48080/api/python.math/add -H "Content-Type: application/json" -d "{\"a\": 10, \"b\": 20}"
```

**预期响应**：
```json
{"result": 30}
```

#### 测试乘法功能

```bash
curl -X POST http://localhost:48080/api/python.math/multiply -H "Content-Type: application/json" -d "{\"a\": 7, \"b\": 8}"
```

**预期响应**：
```json
{"result": 56}
```

#### 测试表达式计算

```bash
curl -X POST http://localhost:48080/api/python.math/calculate -H "Content-Type: application/json" -d "{\"expression\": \"2 * (3 + 4)\"}"
```

**预期响应**：
```json
{"result": 14}
```

## 端口占用问题解决方案

### 方法1：查找并终止占用端口的进程

```bash
# Windows PowerShell
# 查找占用端口的进程
netstat -ano | findstr :29999

# 假设输出显示PID为112716，终止该进程
Stop-Process -Id 112716 -Force

# 或使用任务管理器手动终止
```

### 方法2：使用动态端口分配

创建一个批处理文件 `start_with_free_ports.bat`：

```batch
@echo off
echo Finding available ports...

:: 查找可用的HTTP端口（从30000开始）
set HTTP_PORT=30000
:FIND_HTTP
netstat -an | findstr :%HTTP_PORT% >nul
if %errorlevel%==0 (
    set /a HTTP_PORT+=1
    goto FIND_HTTP
)

:: 查找可用的IPC端口（从31000开始）
set IPC_PORT=31000
:FIND_IPC
netstat -an | findstr :%IPC_PORT% >nul
if %errorlevel%==0 (
    set /a IPC_PORT+=1
    goto FIND_IPC
)

echo Found available ports:
echo   HTTP: %HTTP_PORT%
echo   IPC: %IPC_PORT%
echo.
echo Starting Neo Framework...
go run cmd/neo/main.go -http :%HTTP_PORT% -ipc :%IPC_PORT%
```

### 方法3：使用Docker避免端口冲突

创建 `docker-compose.yml`：

```yaml
version: '3.8'
services:
  neo-framework:
    build: .
    ports:
      - "28080:28080"
      - "29999:29999"
    networks:
      - neo-network

  python-service:
    build: ./python_service
    depends_on:
      - neo-framework
    environment:
      - NEO_IPC_HOST=neo-framework
      - NEO_IPC_PORT=29999
    networks:
      - neo-network

networks:
  neo-network:
    driver: bridge
```

## 简化测试脚本

创建 `test_with_custom_ports.py`：

```python
import subprocess
import time
import requests
import sys

def find_free_port(start_port):
    """查找可用端口"""
    import socket
    port = start_port
    while port < start_port + 100:
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            s.bind(('', port))
            s.close()
            return port
        except:
            port += 1
    return None

def main():
    # 查找可用端口
    http_port = find_free_port(30000)
    ipc_port = find_free_port(31000)
    
    if not http_port or not ipc_port:
        print("无法找到可用端口")
        sys.exit(1)
    
    print(f"使用端口: HTTP={http_port}, IPC={ipc_port}")
    
    # 启动Neo Framework
    neo_proc = subprocess.Popen([
        "go", "run", "cmd/neo/main.go",
        "-http", f":{http_port}",
        "-ipc", f":{ipc_port}"
    ])
    
    time.sleep(3)  # 等待启动
    
    # 测试健康检查
    try:
        resp = requests.get(f"http://localhost:{http_port}/health")
        print(f"健康检查: {resp.json()}")
    except Exception as e:
        print(f"健康检查失败: {e}")
    
    # 这里可以继续添加Python服务启动和测试代码
    
    print("\n按Ctrl+C停止...")
    try:
        neo_proc.wait()
    except KeyboardInterrupt:
        neo_proc.terminate()

if __name__ == "__main__":
    main()
```

## 测试检查清单（更新版）

- [ ] 找到可用的端口组合
- [ ] Neo Framework在自定义端口成功启动
- [ ] 健康检查返回正确响应
- [ ] Python服务配置了正确的IPC端口
- [ ] Python服务成功连接并注册
- [ ] HTTP API调用返回正确结果
- [ ] 错误处理正常工作
- [ ] 服务在多次请求下保持稳定

## 故障排查提示

1. **始终检查端口可用性**
   ```bash
   netstat -an | findstr :端口号
   ```

2. **确保Python服务使用相同的IPC端口**
   - Neo Framework启动时显示的IPC端口
   - Python服务配置的端口必须一致

3. **查看详细日志**
   ```bash
   # 启用调试日志
   go run cmd/neo/main.go -log debug -http :48080 -ipc :49999
   ```

现在请按照更新后的指南进行测试，使用自定义端口来避免冲突。