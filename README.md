# Neo Framework

Neo 是一个基于 Go 语言的高性能微服务通信框架，旨在为分布式系统提供可靠、高效的服务间通信能力，并支持多语言服务集成。

## 最新更新 (2025-08-13)

### 🎉 重大架构改进
- **服务解耦**：HTTP和TCP网关现在作为独立的微服务运行，不再内嵌于Neo核心
- **项目重组**：优化了项目结构，更清晰的源码、脚本和二进制文件管理
- **IPC增强**：修复并增强了IPC服务器的请求转发功能

### 新增功能
- **HTTP网关服务** (`services/http-gateway/`)：独立的HTTP接入服务
- **TCP网关服务** (`services/tcp-gateway/`)：独立的TCP接入服务，支持JSON协议
- **统一构建脚本**：一键编译所有服务 (`scripts/build/build-all.bat`)
- **快速启动**：一键启动整个系统 (`scripts/start-all.bat`)

## 项目概述

### 核心特性

- **微服务架构**：所有组件作为独立服务运行，通过IPC通信
- **多协议支持**：HTTP RESTful API 和高性能 TCP/IPC 协议
- **多语言集成**：通过 IPC 协议支持 Go、Python、Java、Node.js、PHP 等多语言服务
- **异步处理**：基于请求 ID 的异步请求-响应模式，支持高并发场景
- **服务发现**：内置服务注册与发现机制
- **模块化设计**：高内聚低耦合的包结构，易于扩展和维护
- **优雅关闭**：支持服务的优雅启动和关闭

### 架构设计原则

- **服务独立**：每个服务完全独立，无共享代码依赖
- **通信统一**：所有服务间通信通过IPC协议
- **可扩展性**：易于添加新的网关服务和业务服务
- **高可用性**：服务可独立部署、升级和扩容

## 快速开始

### 1. 编译所有服务
```bash
cd scripts/build
build-all.bat
```

### 2. 启动整个系统
```bash
cd scripts
start-all.bat
```
这将启动：
- Neo核心 (IPC服务器)
- Python示例服务
- HTTP网关服务
- TCP网关服务

### 3. 测试服务

#### HTTP网关测试
```bash
# 使用自动化测试脚本
scripts\test\test-http-gateway.bat

# 或手动测试
curl -X POST http://localhost:8081/api/demo-service-python/calculate \
  -H "Content-Type: application/json" \
  -d "{\"operation\":\"add\",\"a\":10,\"b\":20}"
```

#### TCP网关测试
```bash
# 使用测试客户端
scripts\test\test-tcp-gateway.bat
```

## 项目结构

```
neo/
├── bin/                        # 编译后的二进制文件（git-ignored）
│   ├── neo.exe                # Neo核心框架
│   ├── http-gateway.exe       # HTTP网关服务
│   ├── tcp-gateway.exe        # TCP网关服务
│   └── test_client.exe        # TCP测试客户端
├── cmd-src/                   # 命令行程序源代码
│   └── neo/                   # Neo核心主程序
├── services/                  # 独立服务
│   ├── http-gateway/         # HTTP网关服务
│   └── tcp-gateway/          # TCP网关服务
├── scripts/                   # 所有脚本文件
│   ├── build/                # 构建脚本
│   │   └── build-all.bat    # 编译所有服务
│   ├── dev/                  # 开发脚本
│   ├── test/                 # 测试脚本
│   └── start-all.bat        # 一键启动脚本
├── internal/                  # 内部包（Neo核心使用）
│   ├── config/               # 配置管理
│   ├── core/                 # 核心服务
│   ├── ipc/                  # IPC通信实现
│   ├── protocol/             # 协议定义
│   ├── registry/             # 服务注册
│   ├── transport/            # 传输层
│   ├── types/                # 类型定义
│   └── utils/                # 工具函数
├── examples-ipc/             # 多语言IPC服务示例
│   ├── go/                   # Go示例
│   ├── python/               # Python示例
│   ├── java/                 # Java示例
│   ├── nodejs/               # Node.js示例
│   └── php/                  # PHP示例
├── test/                      # 测试代码
│   ├── integration/          # 集成测试
│   └── stress/               # 压力测试
├── configs/                   # 配置文件
└── docs/                      # 文档
    ├── GATEWAY_SERVICES.md   # 网关服务详细说明
    └── ...
```

## 系统架构

```
┌─────────────┐     ┌─────────────┐
│ HTTP Client │     │ TCP Client  │
└──────┬──────┘     └──────┬──────┘
       │                   │
       ▼                   ▼
┌─────────────┐     ┌─────────────┐
│HTTP Gateway │     │ TCP Gateway │
│  (服务)      │     │   (服务)     │
└──────┬──────┘     └──────┬──────┘
       │                   │
       └─────────┬─────────┘
                 ▼
         ┌───────────────┐
         │   Neo Core    │
         │  IPC Server   │
         │  (Port 9999)  │
         └───────┬───────┘
                 │
    ┌────────────┴────────────┐
    ▼                          ▼
┌─────────┐              ┌─────────┐
│ Service │              │ Service │
│   (Go)  │              │ (Python)│
└─────────┘              └─────────┘
```

## 端口分配

- **9999**: Neo IPC Server (核心通信)
- **8081**: HTTP Gateway (HTTP接入)
- **7777**: TCP Gateway (TCP接入)

## 开发指南

### 添加新的网关服务

1. 在 `services/` 创建新服务目录
2. 实现IPC客户端连接和服务注册
3. 实现协议转换逻辑
4. 更新构建脚本

### 添加新的业务服务

1. 参考 `examples-ipc/` 中的示例
2. 实现IPC协议通信
3. 注册服务到Neo IPC Server
4. 处理请求并返回响应

## 测试

### 运行单元测试
```bash
go test ./...
```

### 运行集成测试
```bash
go test ./test/integration -v
```

### 运行压力测试
```bash
python test/stress/stress_test_all.py
```

## 文档

- [网关服务详细说明](docs/GATEWAY_SERVICES.md)
- [IPC协议指南](docs/IPC_PROTOCOL_GUIDE.md)
- [架构设计](docs/ARCHITECTURE_UPDATE.md)
- [项目结构说明](PROJECT_STRUCTURE.md)
- [测试手册](docs/TEST_MANUAL.md)

## 贡献

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License