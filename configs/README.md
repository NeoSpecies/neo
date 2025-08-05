# Neo Framework 配置文件详细指南

## 🎯 快速开始

Neo Framework使用YAML格式的配置文件，支持多环境配置。默认情况下，框架会加载`default.yml`。

### 最简单的启动方式
```bash
# 使用默认配置启动（加载default.yml）
go run cmd/neo/main.go

# 使用开发环境配置启动
go run cmd/neo/main.go -config configs/development.yml
```

## 📁 配置文件说明

| 文件名 | 用途 | 适用场景 | 端口配置 |
|--------|------|----------|----------|
| `default.yml` | 默认配置文件 | 生产环境基准配置 | **HTTP:8080, IPC:9999** ✅ |
| `development.yml` | 开发环境配置 | 本地开发调试 | **HTTP:8080, IPC:9999** ✅ |
| `production.yml` | 生产环境配置 | 线上部署 | **HTTP:8080, IPC:9999** ✅ |
| `test.yml` | 测试环境配置 | 自动化测试 | HTTP:18080, IPC:19999 |

### 🔴 重要：统一端口配置
为了避免混乱，Neo Framework 统一使用以下默认端口：
- **HTTP Gateway**: `8080`
- **IPC Server**: `9999`

所有语言的示例代码都默认连接到这些端口。如需修改，请通过环境变量统一调整：
```bash
export NEO_IPC_PORT=9999
export NEO_HTTP_PORT=8080
```

### 为什么需要这些配置文件？

1. **default.yml** - 提供合理的默认值，即使不指定配置文件也能正常运行
2. **development.yml** - 优化开发体验：详细日志、快速失败、小资源占用
3. **production.yml** - 优化生产性能：大连接池、长超时、JSON日志
4. **test.yml** - 隔离测试环境：不同端口、极短超时、无重试

## 🔧 配置文件结构详解

### 1. 服务器基本配置 (server)
```yaml
server:
  name: "neo-gateway"          # 服务名称，用于日志和监控
  version: "0.1.0"             # 服务版本
  startup_delay: 100ms         # 启动延迟，防止资源竞争
  shutdown_timeout: 10         # 优雅关闭超时时间（秒）
```

### 2. 传输层配置 (transport)
```yaml
transport:
  timeout: 30s                 # 请求总超时时间
  retry_count: 3               # 失败重试次数
  max_connections: 100         # 连接池最大连接数
  min_connections: 10          # 连接池最小连接数
  max_idle_time: 5m            # 连接最大空闲时间
  health_check_interval: 30s   # 健康检查间隔
```

### 3. 服务注册中心配置 (registry)
```yaml
registry:
  type: "inmemory"             # 注册中心类型（目前仅支持inmemory）
  health_check_interval: 30s   # 服务健康检查间隔
  cleanup_interval: 10s        # 无效服务清理间隔
  instance_expiry: 5m          # 服务实例过期时间
  ttl: 30s                     # 服务注册TTL
```

### 4. HTTP网关配置 (gateway)
```yaml
gateway:
  address: ":8080"             # HTTP监听地址（:8080表示监听所有网卡的8080端口）
  read_timeout: 30s            # 读取请求超时
  write_timeout: 30s           # 写入响应超时
  max_header_bytes: 1048576    # 最大请求头大小（1MB）
  request_timeout: 30s         # 单个请求处理超时
```

### 5. IPC服务器配置 (ipc)
```yaml
ipc:
  address: ":9999"             # IPC监听地址
  max_clients: 200             # 最大客户端连接数
  buffer_size: 4096            # 读写缓冲区大小（字节）
  max_message_size: 10485760   # 最大消息大小（10MB）
  response_timeout: 30s        # 等待响应超时
```

### 6. 日志配置 (logging)
```yaml
logging:
  level: "info"                # 日志级别: debug, info, warn, error
  format: "json"               # 日志格式: json（结构化）, text（可读性好）
  output: "console"            # 输出目标: console, file
  with_color: true             # 是否使用颜色（仅text格式）
  with_location: false         # 是否显示代码位置（文件名:行号）
```

## 🚀 使用方法

### 方法1：命令行参数指定配置文件
```bash
# 开发环境
go run cmd/neo/main.go -config configs/development.yml

# 生产环境
go run cmd/neo/main.go -config configs/production.yml

# 测试环境（使用不同端口）
go run cmd/neo/main.go -config configs/test.yml
```

### 方法2：环境变量覆盖配置
所有配置项都可以通过环境变量覆盖，格式为`NEO_<SECTION>_<KEY>`：

```bash
# Windows
set NEO_GATEWAY_ADDRESS=:8888
set NEO_IPC_ADDRESS=:9988
set NEO_LOGGING_LEVEL=debug

# Linux/Mac
export NEO_GATEWAY_ADDRESS=":8888"
export NEO_IPC_ADDRESS=":9988"
export NEO_LOGGING_LEVEL="debug"
```

### 方法3：命令行参数直接覆盖
```bash
# 覆盖端口配置
go run cmd/neo/main.go -http :8888 -ipc :9988

# 组合使用：指定配置文件并覆盖特定值
go run cmd/neo/main.go -config configs/development.yml -http :8888
```

### 优先级说明
配置优先级从高到低：
1. 命令行参数（最高优先级）
2. 环境变量
3. 配置文件
4. 默认值（最低优先级）

## 📊 环境配置对比

| 配置项 | 开发环境 | 生产环境 | 测试环境 |
|--------|----------|----------|----------|
| **日志级别** | debug | info | debug |
| **日志格式** | text（带颜色） | json | text（无颜色） |
| **超时时间** | 10秒 | 30秒 | 5秒 |
| **重试次数** | 1次 | 3次 | 0次 |
| **连接池大小** | 50 | 200 | 10 |
| **健康检查间隔** | 10秒 | 30秒 | 5秒 |
| **HTTP端口** | 8080 | 8080 | 18080 |
| **IPC端口** | 9999 | 9999 | 19999 |

## 🔍 配置验证

框架启动时会自动验证配置：
- 检查必填项是否存在
- 验证端口号是否合法（1-65535）
- 验证超时时间是否为正数
- 检查文件路径是否可访问

如果配置有误，程序会打印错误信息并退出。

## 💡 最佳实践

### 1. 环境隔离
- 开发环境使用`development.yml`
- 测试环境使用`test.yml`（不同端口避免冲突）
- 生产环境使用`production.yml`

### 2. 敏感信息管理
```bash
# 不要在配置文件中硬编码敏感信息
# 使用环境变量存储敏感配置
export NEO_DATABASE_PASSWORD="your-secret-password"
export NEO_API_KEY="your-api-key"
```

### 3. 配置文件版本控制
```gitignore
# 将默认配置纳入版本控制
configs/default.yml
configs/development.yml
configs/production.yml
configs/test.yml

# 排除包含敏感信息的本地配置
configs/local.yml
configs/*.local.yml
```

### 4. 自定义配置
创建自定义配置文件的步骤：
```bash
# 1. 复制模板
cp configs/default.yml configs/custom.yml

# 2. 编辑配置
vim configs/custom.yml

# 3. 使用自定义配置启动
go run cmd/neo/main.go -config configs/custom.yml
```

## 🛠️ 故障排查

### 常见问题

1. **端口被占用**
   ```bash
   # 错误：bind: address already in use
   # 解决：修改配置文件中的端口或使用命令行参数
   go run cmd/neo/main.go -http :8888 -ipc :9988
   ```

2. **配置文件找不到**
   ```bash
   # 错误：failed to load config: open configs/xxx.yml: no such file
   # 解决：确保配置文件路径正确
   go run cmd/neo/main.go -config ./configs/development.yml
   ```

3. **YAML格式错误**
   ```bash
   # 错误：failed to parse YAML
   # 解决：检查YAML语法，特别是缩进和特殊字符
   ```

## 📚 扩展阅读

- [主项目README](../README.md) - 了解项目整体架构
- [测试手册](../docs/TEST_MANUAL.md) - 了解如何测试不同环境配置
- [IPC协议指南](../docs/IPC_PROTOCOL_GUIDE.md) - 了解IPC通信配置

---

*文档编写：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*

如有任何配置相关问题，请查看[故障排查指南](../docs/TEST_MANUAL.md#故障排查指南)或联系 neospecies@outlook.com。