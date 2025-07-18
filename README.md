# Neo Framework

Neo 是一个基于 Go 语言的高性能微服务通信框架，旨在为分布式系统提供可靠、高效的服务间通信能力，并支持多语言服务集成。

## 项目概述

### 核心特性

- **多协议支持**：HTTP RESTful API 和高性能 IPC（进程间通信）协议
- **多语言集成**：通过 IPC 协议支持 Python、Java 等多语言服务接入
- **异步处理**：基于请求 ID 的异步请求-响应模式，支持高并发场景
- **服务发现**：内置服务注册与发现机制
- **模块化设计**：高内聚低耦合的包结构，易于扩展和维护
- **优雅关闭**：支持服务的优雅启动和关闭

### 架构设计原则

- **模块化**：功能划分为独立的包，每个包职责单一
- **高内聚低耦合**：包内功能相关，包间通过接口交互
- **可测试性**：每个包可独立测试，支持单元测试和集成测试
- **可扩展性**：易于添加新协议和服务机制

## 项目结构

```
neo/
├── cmd/                        # 命令行应用
│   ├── neo/                   # 主要应用程序（推荐使用）
│   ├── gateway/               # 生产环境 HTTP 网关
│   ├── simple_gateway/        # 简化测试网关
│   ├── test_gateway/          # 集成测试网关
│   └── main.go               # 基础服务应用
├── configs/                   # 配置文件
│   └── default.yml           # 默认配置
├── internal/                  # 内部包（不对外暴露）
│   ├── config/               # 配置管理
│   │   ├── config.go         # 配置结构和方法
│   │   ├── provider.go       # 配置提供者接口
│   │   └── errors.go         # 错误定义
│   ├── core/                 # 核心业务逻辑
│   │   ├── service.go        # 同步服务实现
│   │   └── async_service.go  # 异步服务实现
│   ├── gateway/              # HTTP 网关实现
│   │   └── http_gateway.go   # HTTP 请求处理和路由
│   ├── ipc/                  # 进程间通信
│   │   ├── server.go         # IPC 服务器实现
│   │   └── async_handler.go  # 异步请求处理器
│   ├── protocol/             # 协议层
│   │   ├── codec.go          # 编解码接口
│   │   ├── http.go           # HTTP 协议实现
│   │   └── ipc.go            # IPC 协议实现
│   ├── registry/             # 服务注册与发现
│   │   ├── registry.go       # 注册中心接口和实现
│   │   └── instance.go       # 服务实例定义
│   ├── transport/            # 传输层实现
│   │   ├── transport.go      # 传输层接口
│   │   ├── ipc_transport.go  # IPC 传输实现
│   │   ├── async_ipc_transport.go # 异步 IPC 传输
│   │   ├── codec/            # 消息编解码
│   │   ├── conn/             # 连接管理
│   │   └── retry/            # 重试策略
│   ├── types/                # 类型定义
│   │   ├── message.go        # 消息结构
│   │   ├── request.go        # 请求结构
│   │   └── response.go       # 响应结构
│   └── utils/                # 工具函数
│       ├── log.go            # 日志工具
│       └── string.go         # 字符串处理
├── pkg/                       # 公共 API（可对外暴露）
│   └── api.go                # 客户端 API
├── examples-ipc/              # 多语言服务示例
│   ├── python/               # Python 服务示例
│   ├── go/                   # Go 服务示例
│   ├── nodejs/               # Node.js 服务示例
│   ├── java/                 # Java 服务示例
│   └── php/                  # PHP 服务示例
├── test/                      # 测试文件
│   ├── python/               # Python 测试脚本
│   ├── integration/          # Go 集成测试
│   ├── stress/               # 压力测试
│   ├── run_tests.sh         # Unix/Linux 测试脚本
│   ├── run_tests.bat        # Windows 测试脚本
│   └── README.md            # 测试说明文档
├── scripts/                   # 启动和管理脚本
│   ├── start_auto.bat       # 自动端口管理启动脚本
│   ├── Start-Neo.ps1        # PowerShell 启动脚本
│   └── stop_ports.bat       # 端口清理工具
├── docs/                      # 项目文档
│   ├── ARCHITECTURE_UPDATE.md    # 架构更新说明
│   ├── NeoPackageDetailedDesign.md # 包详细设计
│   ├── NeoTestingPlan.md        # 测试计划
│   └── PORT_MANAGEMENT.md       # 端口管理指南
├── logs/                      # 日志文件目录（git忽略）
├── go.mod                     # Go 模块定义
├── go.sum                     # Go 依赖锁定
├── start.bat                  # 快速启动脚本（Windows）
├── start.sh                   # 快速启动脚本（Unix）
└── test_manual.md            # 手动测试指南
```

## 通信协议

### HTTP 协议
- RESTful API 风格
- 路径格式：`/api/{service}/{method}`
- 支持 JSON 序列化
- 适用于 Web 服务和外部 API

### IPC 协议
- 自定义二进制协议，高性能低延迟
- 消息格式：
  ```
  [消息类型:1字节][ID长度:4字节][ID:N字节]
  [服务名长度:4字节][服务名:N字节][方法名长度:4字节][方法名:N字节]
  [元数据长度:4字节][元数据JSON:N字节][数据长度:4字节][数据:N字节]
  ```
- 消息类型：
  - REQUEST (1): 请求消息
  - RESPONSE (2): 响应消息
  - REGISTER (3): 服务注册
  - HEARTBEAT (4): 心跳检测

## 调用流程

```
HTTP客户端 → HTTPGateway → AsyncService → AsyncTransport → AsyncIPCServer → 语言服务
     ↑                                                                           ↓
     ← ← ← ← ← ← ← ← ← ← ← 异步响应 ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ← ←
```

### 详细流程说明

1. **HTTP 请求接收**：HTTPGateway 在 8080 端口接收 HTTP 请求
2. **路由解析**：解析 URL 路径提取服务名和方法名
3. **请求转发**：通过 AsyncService 处理请求
4. **服务发现**：通过 Registry 查找目标服务
5. **IPC 通信**：AsyncIPCServer 转发请求到注册的服务
6. **服务处理**：目标语言服务（Python/Go/Node.js/Java/PHP）处理请求
7. **响应返回**：响应按原路返回给客户端

## 快速开始

### 环境要求

- Go 1.16 或更高版本
- Python 3.7+ （测试 Python 服务）
- Node.js 14+ （测试 Node.js 服务）
- Java 8+ （测试 Java 服务）
- PHP 7.4+ （测试 PHP 服务，需要启用 sockets 扩展）

### 启动服务

#### 1. 启动 Neo Framework
```bash
# 使用默认配置启动
go run cmd/neo/main.go

# 或使用特定环境配置
go run cmd/neo/main.go -config configs/development.yml
```

您应该看到输出：
```
=== Neo Framework ===
HTTP网关: http://localhost:8080
IPC服务器: localhost:9999
健康检查: http://localhost:8080/health
```

#### 2. 启动语言服务示例

在另一个终端窗口，选择要测试的语言服务：

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
# 需要先下载 Gson 库：https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar
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
php service.php
# 服务将注册为 demo-service-php
```

#### 3. 通过 HTTP 网关测试

服务启动后，可以通过HTTP网关调用任意语言的服务：

```bash
# 健康检查
curl http://localhost:8080/health

# 调用 Python 服务
curl -X POST http://localhost:8080/api/demo-service-python/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo Framework"}'

# 调用 Go 服务
curl -X POST http://localhost:8080/api/demo-service-go/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo Framework"}'

# 调用 Node.js 服务
curl -X POST http://localhost:8080/api/demo-service-nodejs/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo Framework"}'

# 调用 Java 服务
curl -X POST http://localhost:8080/api/demo-service-java/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo Framework"}'

# 调用 PHP 服务
curl -X POST http://localhost:8080/api/demo-service-php/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo Framework"}'
```

#### 统一的 API 格式

所有服务都遵循相同的API格式：
- URL路径：`http://localhost:8080/api/{service-name}/{method}`
- 请求方法：POST
- Content-Type：application/json
- 响应格式：JSON

每个服务都实现了以下方法：
- `hello` - 问候方法
- `calculate` - 数学计算
- `echo` - 回显消息
- `getTime` - 获取当前时间
- `getInfo` - 获取服务信息

更多详细测试步骤请参考 [测试手册](docs/TEST_MANUAL.md)。

## 开发指南

### 创建新服务

#### Go 服务
```go
// 实现 Service 接口
type MyService struct {
    transport transport.Transport
    registry  registry.ServiceRegistry
}

func (s *MyService) HandleRequest(ctx context.Context, req types.Request) (types.Response, error) {
    // 处理请求逻辑
}
```

#### Python 服务
```python
from neo_client import NeoIPCClient

class MyPythonService:
    def __init__(self):
        self.client = NeoIPCClient("localhost", 29999)
    
    async def my_method(self, request):
        # 处理请求
        return {"result": "success"}
    
    async def start(self):
        await self.client.connect()
        await self.client.register_service("my.service", {
            "version": "1.0.0"
        })
        self.client.register_handler("my_method", self.my_method)
        await self.client.listen()
```

### 配置说明

配置文件位于 `configs/default.yml`：

```yaml
transport:
  timeout: 30
  retry_count: 3
  
registry:
  type: inmemory
  
gateway:
  address: ":8080"    # HTTP网关端口
  
ipc:
  address: ":9999"     # IPC服务器端口
```

## 构建和打包

### 构建可执行文件

构建所有应用程序：
```bash
# 构建主应用程序
go build -o bin/neo cmd/neo/main.go

# 构建独立网关
go build -o bin/gateway cmd/gateway/main.go

# 构建测试网关
go build -o bin/test_gateway cmd/test_gateway/main.go
go build -o bin/simple_gateway cmd/simple_gateway/main.go

# 构建基础服务
go build -o bin/service cmd/main.go
```

### 跨平台构建

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o bin/neo-linux cmd/neo/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o bin/neo.exe cmd/neo/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o bin/neo-macos cmd/neo/main.go
```

### Docker 部署

创建 Dockerfile：
```dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o neo cmd/neo/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/neo .
COPY --from=builder /app/configs ./configs
EXPOSE 8080 9999
CMD ["./neo"]
```

构建和运行：
```bash
docker build -t neo-framework .
docker run -p 8080:8080 -p 9999:9999 neo-framework
```

### 发布打包

```bash
# 创建发布目录
mkdir -p release/neo-v1.0.0

# 复制必要文件
cp bin/neo release/neo-v1.0.0/
cp -r configs release/neo-v1.0.0/
cp -r examples-ipc release/neo-v1.0.0/
cp README.md release/neo-v1.0.0/

# 创建压缩包
tar -czf neo-framework-v1.0.0.tar.gz -C release neo-v1.0.0
```

## 测试

### 测试脚本

使用提供的测试脚本运行不同类型的测试：

**Windows**:
```bash
# 运行所有测试
test\run_tests.bat all

# 运行 Python 测试
test\run_tests.bat python

# 运行压力测试
test\run_tests.bat stress

# 运行集成测试
test\run_tests.bat integration
```

**Unix/Linux**:
```bash
# 运行所有测试
./test/run_tests.sh all

# 运行 Python 测试
./test/run_tests.sh python

# 运行压力测试
./test/run_tests.sh stress

# 运行集成测试
./test/run_tests.sh integration
```

### 单元测试

```bash
# 运行所有 Go 测试
go test ./...

# 运行特定包的测试
go test ./internal/core/...

# 查看测试覆盖率
go test -cover ./...
```

### 手动测试

详细的手动测试步骤请参考 `test_manual.md`。

### 压力测试

```bash
# 运行压力测试脚本
python test/stress/test_stress.py
```

压力测试会发送大量并发请求来测试框架的性能和稳定性。

## 包设计说明

### 核心包

- **`internal/config`**：配置管理，支持文件和环境变量
- **`internal/types`**：通用数据类型定义
- **`internal/protocol`**：协议实现，支持 HTTP 和 IPC
- **`internal/transport`**：传输层，包括连接管理、编解码和重试
- **`internal/registry`**：服务注册与发现
- **`internal/core`**：核心业务逻辑，包括同步和异步服务
- **`internal/gateway`**：HTTP 网关实现
- **`internal/ipc`**：IPC 服务器和异步处理器

### 依赖关系

```
cmd → core, config, gateway, ipc
core → transport, registry
transport → protocol, config
protocol → types
registry → config
gateway → core, registry
ipc → registry
```

## 支持的语言服务

Neo Framework 通过 IPC 协议支持多种编程语言：

| 语言 | 服务名称 | 示例位置 | 特点 |
|------|----------|----------|------|
| Python | demo-service-python | examples-ipc/python | 异步支持，简洁实现 |
| Go | demo-service-go | examples-ipc/go | 原生性能，类型安全 |
| Node.js | demo-service-nodejs | examples-ipc/nodejs | 事件驱动，无依赖 |
| Java | demo-service-java | examples-ipc/java | 企业级，需要Gson |
| PHP | demo-service-php | examples-ipc/php | Web友好，需要sockets扩展 |

每种语言都实现了相同的服务接口，可以无缝切换使用。

## 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 添加适当的注释和文档
- 编写单元测试覆盖新功能

### 安全注意事项

- **绝不提交 AI 助手文件**（.claude, .anthropic, .cursor 等）
- **检查 .gitignore** 确保敏感文件被忽略
- **运行 `scripts/setup-git-hooks.bat`** 设置 pre-commit 钩子
- 详情请查看 `docs/GITIGNORE_GUIDE.md`

## 文档

项目文档位于 `docs/` 目录：

- `docs/ARCHITECTURE_UPDATE.md` - 架构演进历史、迁移指南和未来规划
- `docs/NeoPackageDetailedDesign.md` - 包的详细设计规范和实现要求
- `docs/NeoTestingPlan.md` - 完整的测试策略和质量保证计划
- `docs/neo-architecture-and-code.md` - 架构分析、代码示例和重构指导
- `docs/PORT_MANAGEMENT.md` - 端口管理和自动化脚本使用指南
- `docs/TEST_MANUAL.md` - 完整的测试手册，包含所有语言服务的测试步骤
- `docs/Neo_Framework_Complete_Test_Report.md` - 全语言服务测试报告
- `examples-ipc/README.md` - 多语言IPC服务示例说明
- `configs/README.md` - 配置文件详细使用指南

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情

## 联系方式

- 项目主页：[https://github.com/NeoSpecies/neo/](https://github.com/NeoSpecies/neo/)
- 问题追踪：[https://github.com/NeoSpecies/neo//issues](https://github.com/NeoSpecies/neo//issues)

---

**当前版本**：0.1.0-alpha  
**最后更新**：2025年7月14日