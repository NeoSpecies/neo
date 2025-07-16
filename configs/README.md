# Neo Framework 配置指南

## 配置文件说明

Neo Framework 支持多环境配置，通过不同的配置文件来适配不同的运行环境。

### 配置文件列表

- `default.yml` - 默认配置，适用于生产环境
- `development.yml` - 开发环境配置
- `production.yml` - 生产环境配置
- `test.yml` - 测试环境配置

### 使用方法

1. **通过命令行参数指定配置文件**：
   ```bash
   # 使用开发环境配置
   go run cmd/neo/main.go -config configs/development.yml
   
   # 使用生产环境配置
   go run cmd/neo/main.go -config configs/production.yml
   
   # 使用测试环境配置
   go run cmd/neo/main.go -config configs/test.yml
   ```

2. **通过环境变量覆盖配置**：
   所有配置项都可以通过环境变量覆盖，环境变量名格式为 `NEO_<SECTION>_<KEY>`，例如：
   ```bash
   # 覆盖HTTP网关端口
   export NEO_GATEWAY_ADDRESS=":8888"
   
   # 覆盖IPC服务器端口
   export NEO_IPC_ADDRESS=":9988"
   
   # 覆盖日志级别
   export NEO_LOG_LEVEL="debug"
   ```

3. **通过命令行参数覆盖特定配置**：
   ```bash
   # 覆盖HTTP和IPC端口
   go run cmd/neo/main.go -http :8888 -ipc :9988
   ```

## 环境配置差异

### 开发环境 (development.yml)
- **特点**：快速失败、详细日志、小资源占用
- **超时时间**：较短（10秒）
- **连接池**：较小（50个连接）
- **日志级别**：DEBUG，带颜色和代码位置
- **健康检查**：频繁（10秒）

### 生产环境 (production.yml)
- **特点**：稳定可靠、高性能、大资源占用
- **超时时间**：适中（30秒）
- **连接池**：较大（200个连接）
- **日志级别**：INFO，JSON格式
- **健康检查**：适中（30秒）

### 测试环境 (test.yml)
- **特点**：快速执行、隔离端口、最小资源
- **超时时间**：很短（5秒）
- **连接池**：很小（10个连接）
- **日志级别**：DEBUG，无颜色
- **端口**：使用不同端口避免冲突

## 配置项详解

### server - 服务器基本配置
- `name`: 服务名称
- `version`: 服务版本
- `startup_delay`: 启动延迟（毫秒）
- `shutdown_timeout`: 关闭超时（秒）

### transport - 传输层配置
- `timeout`: 请求超时时间
- `retry_count`: 重试次数
- `max_connections`: 最大连接数
- `health_check_interval`: 健康检查间隔

### registry - 服务注册中心配置
- `type`: 注册中心类型（inmemory/etcd/consul）
- `cleanup_interval`: 清理间隔
- `instance_expiry`: 实例过期时间

### gateway - HTTP网关配置
- `address`: 监听地址
- `read_timeout`: 读取超时
- `write_timeout`: 写入超时

### ipc - IPC服务器配置
- `address`: 监听地址
- `max_clients`: 最大客户端连接数
- `max_message_size`: 最大消息大小

### logging - 日志配置
- `level`: 日志级别（debug/info/warn/error）
- `format`: 日志格式（text/json）
- `with_color`: 是否使用颜色
- `with_location`: 是否显示代码位置

## 最佳实践

1. **环境隔离**：不同环境使用不同的配置文件
2. **敏感信息**：使用环境变量存储敏感信息，不要硬编码在配置文件中
3. **版本控制**：将配置文件纳入版本控制，但排除包含敏感信息的本地配置
4. **配置验证**：启动时会自动验证配置的合法性
5. **热更新**：配置管理器支持配置变更通知（需要实现配置监听）

## 自定义配置

如需创建自定义配置文件：

1. 复制 `default.yml` 作为模板
2. 修改需要的配置项
3. 保存为新文件（如 `custom.yml`）
4. 使用 `-config configs/custom.yml` 参数启动