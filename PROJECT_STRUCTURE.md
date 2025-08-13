# Neo Framework 项目结构

## 目录说明

```
light/
├── bin/                    # 编译后的二进制文件
│   ├── neo.exe            # Neo核心框架
│   ├── http-gateway.exe  # HTTP网关服务
│   ├── tcp-gateway.exe   # TCP网关服务
│   └── test_client.exe   # TCP测试客户端
│
├── cmd-src/               # 命令行程序源代码
│   ├── neo/              # Neo核心主程序
│   └── gateway/          # 网关相关程序（已废弃）
│
├── services/              # 独立服务源代码
│   ├── http-gateway/     # HTTP网关服务
│   └── tcp-gateway/      # TCP网关服务
│
├── scripts/               # 所有脚本文件
│   ├── build/            # 构建脚本
│   │   └── build-all.bat # 编译所有服务
│   ├── dev/              # 开发脚本
│   │   ├── start-neo.bat      # 启动Neo核心
│   │   ├── start-gateways.bat # 启动网关服务
│   │   └── ...
│   ├── test/             # 测试脚本
│   │   ├── test-http-gateway.bat # HTTP网关测试
│   │   └── test-tcp-gateway.bat  # TCP网关测试
│   ├── deploy/           # 部署脚本（预留）
│   └── start-all.bat     # 一键启动整个系统
│
├── internal/              # 内部包
│   ├── config/           # 配置管理
│   ├── core/             # 核心服务
│   ├── ipc/              # IPC通信
│   ├── protocol/         # 协议定义
│   ├── registry/         # 服务注册
│   ├── transport/        # 传输层
│   ├── types/            # 类型定义
│   └── utils/            # 工具函数
│
├── examples-ipc/          # IPC服务示例
│   ├── go/               # Go语言示例
│   ├── python/           # Python示例
│   ├── nodejs/           # Node.js示例
│   ├── java/             # Java示例
│   └── php/              # PHP示例
│
├── test/                  # 测试代码
│   ├── integration/      # 集成测试
│   ├── stress/           # 压力测试
│   └── manual/           # 手动测试
│
├── configs/               # 配置文件
│   ├── default.yml       # 默认配置
│   ├── development.yml   # 开发环境配置
│   └── production.yml    # 生产环境配置
│
└── docs/                  # 文档
    ├── ARCHITECTURE.md    # 架构说明
    ├── IPC_PROTOCOL.md    # IPC协议文档
    └── ...

```

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

### 3. 测试服务
```bash
# 测试HTTP网关
scripts\test\test-http-gateway.bat

# 测试TCP网关  
scripts\test\test-tcp-gateway.bat
```

## 开发指南

### 添加新服务
1. 在 `services/` 目录下创建新服务目录
2. 编写服务代码
3. 更新 `scripts/build/build-all.bat` 添加编译步骤
4. 创建相应的启动和测试脚本

### 脚本组织原则
- **build/** - 编译相关脚本
- **dev/** - 开发环境启动、调试脚本
- **test/** - 各类测试脚本
- **deploy/** - 部署、打包脚本

### 二进制文件管理
- 所有编译输出统一放在 `bin/` 目录
- 不要将二进制文件提交到git
- 使用 `build-all.bat` 统一编译

## 服务架构

```
┌─────────────┐     ┌─────────────┐
│ HTTP Client │     │ TCP Client  │
└──────┬──────┘     └──────┬──────┘
       │                   │
       ▼                   ▼
┌─────────────┐     ┌─────────────┐
│HTTP Gateway │     │ TCP Gateway │
│  (Port 8081)│     │  (Port 7777)│
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
- 9999: Neo IPC Server
- 8081: HTTP Gateway
- 7777: TCP Gateway