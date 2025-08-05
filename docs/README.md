# Neo Framework 文档中心

欢迎来到 Neo Framework 文档中心！这里包含了所有你需要了解的技术文档和指南。

## 🚀 快速开始

如果你是第一次使用 Neo Framework，建议按以下顺序阅读：

1. **[主项目 README](../README.md)** - 了解项目概览和快速启动
2. **[测试手册](TEST_MANUAL.md)** - 学习如何运行和测试各语言服务
3. **[配置指南](../configs/README.md)** - 理解配置系统

### 最简单的三步启动

```bash
# 1. 启动 Neo Framework
go run cmd/neo/main.go

# 2. 启动任意语言服务（以Python为例）
cd examples-ipc/python && python service.py

# 3. 测试服务
curl -X POST http://localhost:8080/api/demo-service-python/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo"}'
```

## 📚 文档分类

### 架构文档
- **[Neo架构和代码说明](neo-architecture-and-code.md)** - 深入理解系统架构设计
- **[架构更新说明](ARCHITECTURE_UPDATE.md)** - 追踪最新架构演进
- **[Neo包详细设计](NeoPackageDetailedDesign.md)** - 核心包的设计规范

### 开发指南
- **[IPC快速开始](IPC_QUICK_START.md)** - 5分钟上手IPC服务开发
- **[IPC协议指南](IPC_PROTOCOL_GUIDE.md)** - IPC二进制协议详细规范
- **[统一端口配置](UNIFIED_PORT_CONFIGURATION.md)** 🔴 - 重要：默认端口说明和最佳实践
- **[端口管理](PORT_MANAGEMENT.md)** - 端口配置和冲突解决工具
- **[Git忽略文件指南](GITIGNORE_GUIDE.md)** - 保护敏感信息

### 测试文档
- **[测试手册](TEST_MANUAL.md)** ⭐ - 全面的测试步骤和故障排查
- **[完整测试报告](Neo_Framework_Complete_Test_Report.md)** - 五种语言服务的测试结果
- **[IPC示例测试报告](IPC_EXAMPLES_TEST_REPORT.md)** 🆕 - 最新的IPC示例深度测试和问题分析
- **[测试计划](NeoTestingPlan.md)** - 测试策略和质量保证

### 配置管理
- **[配置文件说明](../configs/README.md)** - 详细的配置使用指南
  - 开发环境配置
  - 生产环境配置
  - 测试环境配置

## 🌐 支持的语言

Neo Framework 支持以下编程语言的服务集成：

| 语言 | 示例位置 | 服务名称 | 快速启动 |
|------|----------|----------|----------|
| Python | [examples-ipc/python](../examples-ipc/python) | demo-service-python | `python service.py` |
| Go | [examples-ipc/go](../examples-ipc/go) | demo-service-go | `go run service.go` |
| Node.js | [examples-ipc/nodejs](../examples-ipc/nodejs) | demo-service-nodejs | `node service.js` |
| Java | [examples-ipc/java](../examples-ipc/java) | demo-service-java | `java -cp .:gson-2.10.1.jar Service` |
| PHP | [examples-ipc/php](../examples-ipc/php) | demo-service-php | `php service.php` |

## 🔍 常见问题快速定位

- **端口被占用？** → 查看 [端口管理](PORT_MANAGEMENT.md)
- **服务启动失败？** → 查看 [测试手册-故障排查](TEST_MANUAL.md#故障排查指南)
- **如何添加新语言？** → 查看 [IPC协议指南](IPC_PROTOCOL_GUIDE.md)
- **配置不生效？** → 查看 [配置文件说明](../configs/README.md)

## 📊 测试覆盖情况

根据最新的[测试报告](Neo_Framework_Complete_Test_Report.md)：

- ✅ Python 服务：6/6 测试通过
- ✅ Go 服务：6/6 测试通过
- ✅ Node.js 服务：6/6 测试通过
- ✅ Java 服务：6/6 测试通过
- ✅ PHP 服务：6/6 测试通过

总测试覆盖率：**100%** (30/30)

## 🛠️ 工具和脚本

- **启动脚本**：[scripts目录](../scripts/)
  - `start.bat` - Windows快速启动
  - `start.sh` - Linux/Mac快速启动
  - `Start-Neo.ps1` - PowerShell高级启动

- **测试脚本**：[test目录](../test/)
  - `run_tests.bat` - Windows测试套件
  - `run_tests.sh` - Linux/Mac测试套件

## 📝 贡献文档

如果你想为文档做贡献：

1. 遵循现有的文档格式和风格
2. 使用清晰的标题和子标题
3. 提供具体的代码示例
4. 包含常见问题和解决方案
5. 更新相关的索引和链接

## 🔗 外部资源

- [示例代码库](../examples-ipc/) - 多语言服务实现示例
- [主项目页面](../README.md) - 项目概览和快速开始
- [问题追踪](https://github.com/NeoSpecies/neo/issues) - 报告问题和建议

---

**文档版本**：1.0.1  
**最后更新**：2025-08-05

如有任何问题，请查阅具体文档或在项目Issue中提问。