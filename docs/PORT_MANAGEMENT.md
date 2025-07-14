# Neo Framework 端口管理指南

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
# 清理默认端口（28080, 29999）
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

## Python服务端口配置

Python服务现在会自动读取启动脚本保存的端口配置：

1. 启动脚本会在 `%TEMP%\neo_ports.env` 保存端口信息
2. Python服务启动时自动读取该配置
3. 如果没有配置文件，使用默认端口29999

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

- **生产环境**：28080, 29999（默认）
- **开发环境**：38080, 39999
- **测试环境**：48080, 49999
- **备用端口**：58080, 59999

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

1. **开发环境**：使用 `Start-Neo.ps1 -Force` 快速迭代
2. **测试环境**：使用 `start_auto.bat` 交互式管理
3. **生产环境**：固定端口，使用系统服务管理

现在你可以使用这些脚本轻松管理端口问题，不用每次手动更换端口了！