# Neo框架架构演进记录

## 版本历史

### v1.0 - 初始架构
- 基础的微服务通信框架
- 支持HTTP和IPC协议
- 模块化设计：transport、protocol、registry、config、types
- 高内聚低耦合的包设计

### v2.0 - 网关和异步支持
- 新增HTTP网关功能
- 实现完整的IPC服务器
- 支持异步请求处理
- 集成Python服务支持

### v2.1 - 应用集成和端口标准化
- 新增完整的应用程序启动器（cmd/neo/main.go）
- 端口标准化和多应用支持
- 完整的配置文件支持
- Application级别的生命周期管理

### v3.0 - 微服务架构重构（当前版本）
- **服务解耦**：HTTP和TCP网关从核心分离为独立服务
- **项目重组**：优化目录结构，分离源码、脚本和二进制
- **IPC增强**：修复请求转发，支持服务间直接通信
- **网关独立**：每个网关自包含IPC客户端，无共享依赖
- **构建优化**：统一构建和部署流程

## 架构演进详情

### 1. 新增组件

#### 1.1 独立网关服务 (`services/http-gateway`, `services/tcp-gateway`)
**背景**：需要灵活的外部接入点，支持不同协议和独立部署。

**设计决策**：
- 网关作为独立微服务运行
- 每个网关自包含IPC客户端
- 支持独立部署和扩展
- HTTP网关：RESTful API，路径 `/api/{service}/{method}`
- TCP网关：原生Socket，JSON消息协议

**实现要点**：
- 完全独立的服务，无共享代码
- 通过IPC协议注册和通信
- 支持水平扩展

#### 1.2 IPC服务器 (`internal/ipc`)
**背景**：需要高性能的进程间通信机制，支持跨语言服务集成。

**设计决策**：
- 基于TCP的二进制协议
- 支持多种消息类型：REQUEST、RESPONSE、REGISTER、HEARTBEAT
- 异步消息处理机制
- 服务自动注册和发现

**协议格式**：
```
[消息长度:4字节][消息类型:1字节][ID长度:4字节][ID:N字节]
[服务名长度:4字节][服务名:N字节][方法名长度:4字节][方法名:N字节]
[元数据长度:4字节][元数据JSON:N字节][数据长度:4字节][数据:N字节]
```

#### 1.3 多语言服务集成 (`examples-ipc`)
**背景**：支持多种编程语言编写的服务接入Neo框架。

**设计决策**：
- 为每种语言提供IPC客户端实现
- 支持Python、Go、Node.js、Java、PHP等语言
- 统一的二进制协议通信
- 自动服务注册和方法处理器机制

### 2. 核心改进

#### 2.1 异步处理支持
**改进内容**：
- 新增`AsyncService`和`AsyncTransport`
- 基于请求ID的异步响应匹配
- 支持并发请求处理

**关键代码**：
```go
// AsyncIPCServer.ForwardRequest
func (s *AsyncIPCServer) ForwardRequest(ctx context.Context, serviceName string, method string, data []byte) ([]byte, error) {
    // 异步发送请求并等待响应
}
```

#### 2.2 服务注册增强
**改进内容**：
- 支持元数据注册
- 动态服务发现
- 连接管理和服务映射

### 3. 调用流程变化

#### 3.1 原始调用流程
```
Client → Transport → Protocol → Service
```

#### 3.2 新调用流程
```
HTTP Client → Gateway → Core.Service → Transport → IPC Server → 语言服务
                ↑                                                        ↓
                ← ← ← ← ← ← ← 异步响应 ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ←
```

### 4. 命令行工具演进

#### 4.1 cmd/main.go
- 原始的单一入口点

#### 4.2 cmd/gateway/main.go
- 生产环境网关
- 集成所有组件
- 优雅关闭支持

#### 4.3 cmd/simple_gateway/main.go
- 测试用简化网关
- 模拟响应
- 快速开发测试

#### 4.4 cmd/test_gateway/main.go
- 集成测试网关
- 详细日志输出
- 调试支持

### 5. 性能优化

#### 5.1 连接池管理
- 复用TCP连接
- 减少连接建立开销

#### 5.2 异步处理
- 非阻塞IO
- 并发请求处理
- 请求ID映射机制

#### 5.3 二进制协议
- 高效的消息编码
- 低延迟传输

### 6. 测试策略演进

#### 6.1 单元测试
- 每个包独立测试
- Mock依赖

#### 6.2 集成测试
- 完整调用链测试
- 多语言服务集成测试

#### 6.3 性能测试
- 并发请求测试
- 延迟测试

### 7. 未来规划

#### 7.1 短期目标
- [ ] 添加认证和授权
- [ ] 实现服务限流
- [ ] 添加监控和日志

#### 7.2 中期目标
- [ ] 支持更多语言（Java、Node.js）
- [ ] 实现服务网格功能
- [ ] 添加配置中心集成

#### 7.3 长期目标
- [ ] 云原生支持（Kubernetes）
- [ ] 分布式追踪
- [ ] 自动化运维工具

### 8. 破坏性变更

本次架构演进保持了向后兼容性，没有破坏性变更。所有新功能都是增量添加，原有接口保持不变。

### 9. 迁移指南

对于使用v1.0的项目，迁移到v2.0：

1. **无需修改现有代码**：所有v1.0的API保持兼容
2. **可选使用新功能**：
   - 如需HTTP入口，使用gateway包
   - 如需跨语言支持，集成IPC服务器
   - 如需异步处理，使用AsyncService

### 10. v2.1版本详细改进

#### 10.1 新增cmd/neo/main.go主应用程序
**背景**：需要一个统一的应用程序入口点，集成所有组件。

**设计决策**：
- Application结构体管理完整生命周期
- 命令行参数支持（-config, -log, -http, -ipc）
- 优雅启动和关闭流程
- 统一的错误处理和日志记录

**实现要点**：
```go
type Application struct {
    config         config.Config
    logger         utils.Logger
    registry       registry.ServiceRegistry
    transport      transport.Transport
    ipcServer      *ipc.IPCServer
    asyncIPC       *ipc.AsyncIPCServer
    coreService    core.Service
    httpGateway    *gateway.HTTPGateway
    shutdownCtx    context.Context
    shutdownCancel context.CancelFunc
}
```

#### 10.2 端口标准化
**改进内容**：
- **cmd/neo**: HTTP 8080, IPC 9999（主应用，推荐）
- **cmd/gateway**: HTTP 8080, IPC 9999（生产网关）
- **cmd/test_gateway**: HTTP 8080, IPC 9999（测试网关）
- **cmd/simple_gateway**: HTTP 8080（简化测试）

#### 10.3 配置文件完善
**新增内容**：
- 完整的configs/default.yml配置文件
- 传输层、注册中心、网关、IPC服务器配置
- 支持超时、重试、连接池等详细配置

### 11. v3.0 架构重构详情

#### 11.1 服务独立性改进
**改进内容**：
- HTTP和TCP网关从`internal/gateway`移至`services/`
- 每个服务完全自包含，包括IPC客户端代码
- 删除了共享的common包，避免服务间依赖
- 支持独立编译、部署和升级

#### 11.2 项目结构优化
**新结构**：
```
bin/              # 所有二进制文件
cmd-src/          # 源代码（原cmd）
services/         # 独立服务
scripts/          # 分类脚本
  ├── build/      # 构建脚本
  ├── dev/        # 开发脚本
  └── test/       # 测试脚本
```

#### 11.3 IPC服务器增强
**修复和改进**：
- 实现了handleRequest方法处理请求转发
- 添加了pendingRequests映射跟踪请求源
- 支持服务间直接通信
- 修复了请求-响应路由问题

### 12. 总结

Neo框架v3.0的架构演进主要聚焦于：
- **微服务化**：所有组件作为独立服务运行
- **解耦合**：服务间无代码依赖，仅通过IPC通信
- **灵活部署**：支持分布式部署和独立扩展
- **清晰结构**：源码、脚本、二进制分离管理
- **生产就绪**：完善的构建、测试和部署流程

这些改进使Neo框架成为真正的微服务通信框架，提供了高度灵活和可扩展的架构。

---

*文档编写：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*