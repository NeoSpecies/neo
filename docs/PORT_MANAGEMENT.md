# Neo Framework 端口管理指南

## 默认端口配置

Neo Framework 使用以下默认端口：

| 服务 | 默认端口 | 用途 | 配置位置 |
|------|----------|------|----------|
| **HTTP Gateway** | **8080** | HTTP API 网关 | `configs/default.yml` → `http.port` |
| **IPC Server** | **9999** | 内部进程通信 | `configs/default.yml` → `ipc.port` |

### ⚠️ 重要提示
- **统一默认端口**：HTTP 8080, IPC 9999
- **所有脚本和配置都应使用这些端口**
- **避免使用其他端口以防混淆**

## 快速开始

### 1. 自动启动（推荐）

使用智能启动脚本，自动处理端口占用问题：

```bash
# Windows 批处理（自动清理端口）
start_auto.bat

# PowerShell（更多选项）
.\Start-Neo.ps1 -AutoCleanup

# PowerShell（强制模式，自动终止占用进程）
.\Start-Neo.ps1 -Force
```

### 2. 手动清理端口

如果需要手动清理端口：

```bash
# 清理默认端口（8080, 9999）
stop_ports.bat
```

### 3. 使用自定义端口

```bash
# PowerShell
.\Start-Neo.ps1 -HttpPort 38080 -IpcPort 39999

# 批处理
go run cmd/neo/main.go -http :38080 -ipc :39999
```

## 启动脚本说明

### `start_auto.bat`
- 自动检测端口占用
- 提供三种选择：自动清理、手动选择端口、取消
- 保存端口配置供Python服务使用

### `Start-Neo.ps1`
PowerShell脚本，功能最完善：

```powershell
# 基本使用
.\Start-Neo.ps1

# 自动清理占用的端口
.\Start-Neo.ps1 -AutoCleanup

# 强制模式（不询问，直接清理）
.\Start-Neo.ps1 -Force

# 使用自定义端口
.\Start-Neo.ps1 -HttpPort 38080 -IpcPort 39999
```

### `stop_ports.bat`
- 检查并终止占用默认端口的进程
- 需要管理员权限运行
- 显示占用端口的进程信息

### `start_smart.bat`
- 交互式端口管理
- 显示占用端口的进程详情
- 允许选择终止进程或使用替代端口

## 各语言服务端口配置

### 环境变量配置（推荐）

所有语言的示例都支持通过环境变量配置端口：

```bash
# 设置 IPC 端口
export NEO_IPC_PORT=9999
export NEO_IPC_HOST=localhost

# Windows
set NEO_IPC_PORT=9999
set NEO_IPC_HOST=localhost
```

### 各语言默认配置

| 语言 | 环境变量 | 默认值 | 配置示例 |
|------|----------|--------|----------|
| **Python** | `NEO_IPC_PORT` | 9999 | `os.getenv('NEO_IPC_PORT', '9999')` |
| **Go** | `NEO_IPC_PORT` | 9999 | `os.Getenv("NEO_IPC_PORT")` |
| **Node.js** | `NEO_IPC_PORT` | 9999 | `process.env.NEO_IPC_PORT || 9999` |
| **Java** | `NEO_IPC_PORT` | 9999 | `System.getenv("NEO_IPC_PORT")` |
| **PHP** | `NEO_IPC_PORT` | 9999 | `getenv('NEO_IPC_PORT') ?: 9999` |

### Python服务自动配置

Python服务额外支持自动读取启动脚本保存的端口配置：

1. 启动脚本会在 `%TEMP%\neo_ports.env` 保存端口信息
2. Python服务启动时自动读取该配置
3. 如果没有配置文件，使用默认端口9999

## 常见场景

### 场景1：端口被其他Neo实例占用

```bash
# 使用自动清理脚本
start_auto.bat
# 选择选项1：自动终止进程
```

### 场景2：端口被其他应用占用

```bash
# 查看占用端口的进程
stop_ports.bat

# 如果是重要进程，使用替代端口
.\Start-Neo.ps1 -HttpPort 38080 -IpcPort 39999
```

### 场景3：开发时频繁重启

```bash
# 使用PowerShell强制模式
.\Start-Neo.ps1 -Force
```

## 端口分配建议

### 标准配置（推荐）
- **生产环境**：8080, 9999（官方默认）
- **开发环境**：8080, 9999（统一配置）
- **测试环境**：8080, 9999（统一配置）

### 多实例部署时
- **实例1**：8080, 9999
- **实例2**：8081, 9998
- **实例3**：8082, 9997
- **备用端口**：38080, 39999

## 故障排查

1. **权限不足**
   - 以管理员身份运行脚本
   - 或手动在任务管理器中结束进程

2. **端口仍然被占用**
   - 等待几秒让系统释放端口
   - 使用 `netstat -ano | findstr :端口号` 确认

3. **Python服务连接错误**
   - 确保使用相同的端口配置
   - 检查 `%TEMP%\neo_ports.env` 文件内容

## 最佳实践

### 1. 统一端口配置
- **始终使用默认端口**：HTTP 8080, IPC 9999
- **通过环境变量配置**：避免硬编码端口
- **配置文件管理**：使用 `configs/default.yml` 集中管理

### 2. 环境隔离
```bash
# 开发环境
export NEO_ENV=dev
export NEO_IPC_PORT=9999

# 测试环境  
export NEO_ENV=test
export NEO_IPC_PORT=9999

# 生产环境
export NEO_ENV=prod
export NEO_IPC_PORT=9999
```

### 3. 启动顺序
1. 先启动 Neo Framework
2. 等待服务完全启动（看到 "Neo Framework started successfully"）
3. 再启动各语言的 IPC 服务

### 4. 端口冲突处理
1. **开发环境**：使用 `Start-Neo.ps1 -Force` 快速清理
2. **测试环境**：使用 `start_auto.bat` 交互式管理
3. **生产环境**：使用进程管理器避免端口冲突

### 5. 监控建议
- 监控端口可用性
- 记录端口使用情况
- 设置端口占用告警

现在你可以使用统一的默认端口配置，避免端口混乱问题！

---

*文档编写：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*