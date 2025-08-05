# Neo Framework IPC 多语言客户端示例

本目录包含了 Neo Framework IPC 协议的多语言客户端实现示例，可以直接运行测试。

## 📁 目录结构

```
examples-ipc/
├── python/      # Python 客户端示例
├── go/          # Go 客户端示例
├── java/        # Java 客户端示例
├── nodejs/      # Node.js 客户端示例
├── php/         # PHP 客户端示例
└── README.md    # 本文档
```

## 🚀 快速开始

### 1. 启动 Neo Framework

```bash
# 方式1：使用默认配置
go run cmd/neo/main.go

# 方式2：使用开发环境配置
go run cmd/neo/main.go -config configs/development.yml

# 方式3：使用启动脚本
./scripts/start_auto.bat  # Windows
./start.sh               # Linux/Mac
```

### 2. 运行示例服务

每个语言目录都包含了可直接运行的示例，选择你熟悉的语言：

#### Python
```bash
cd examples-ipc/python
python service.py
```

#### Go
```bash
cd examples-ipc/go
go run service.go
```

#### Java
```bash
cd examples-ipc/java
javac -cp . *.java
java Service
```

#### Node.js
```bash
cd examples-ipc/nodejs
npm install
node service.js
```

#### PHP
```bash
cd examples-ipc/php
php service.php
```

### 3. 测试服务

所有示例服务都实现了相同的接口，可以使用以下命令测试：

```bash
# 测试 hello 方法
curl -X POST http://localhost:8080/api/demo-service/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "Neo"}'

# 测试 calculate 方法
curl -X POST http://localhost:8080/api/demo-service/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 10, "b": 20, "operation": "add"}'

# 测试 echo 方法
curl -X POST http://localhost:8080/api/demo-service/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello Neo Framework!"}'
```

## 📊 语言特性对比

| 特性 | Python | Go | Java | Node.js | PHP |
|------|--------|-----|------|---------|-----|
| 异步支持 | ✅ asyncio | ✅ goroutine | ✅ Thread | ✅ 原生 | ❌ |
| 性能等级 | ★★★☆☆ | ★★★★★ | ★★★★☆ | ★★★☆☆ | ★★☆☆☆ |
| 开发效率 | ★★★★★ | ★★★☆☆ | ★★★☆☆ | ★★★★☆ | ★★★★★ |
| 依赖管理 | pip | go mod | maven/gradle | npm | composer |
| 适用场景 | 数据处理、AI | 高性能服务 | 企业应用 | Web服务 | Web应用 |

## 🔧 环境要求

- **Python**: >= 3.7
- **Go**: >= 1.16
- **Java**: >= 8
- **Node.js**: >= 12
- **PHP**: >= 7.0

## 📝 示例功能

所有示例都实现了以下功能：

1. **hello** - 简单的问候功能
2. **calculate** - 基础数学运算（加减乘除）
3. **echo** - 回显消息
4. **getTime** - 获取服务器时间
5. **getInfo** - 获取服务信息

## 🛠️ 自定义开发

基于这些示例，你可以：

1. 修改服务名称和方法
2. 添加自己的业务逻辑
3. 集成到现有项目
4. 扩展协议功能

## 📚 相关文档

- [IPC 协议详细说明](../docs/IPC_PROTOCOL_GUIDE.md)
- [快速入门指南](../docs/IPC_QUICK_START.md)
- [Neo Framework 文档](../README.md)

## ❓ 常见问题

**Q: 连接失败怎么办？**
- 确保 Neo Framework 正在运行
- 检查端口 9999 是否被占用
- 查看防火墙设置

**Q: 如何修改连接地址？**
- 查看各语言示例中的配置部分
- 通常可以通过环境变量或参数设置

**Q: 如何添加新的处理方法？**
- 参考示例中的 handler 注册方式
- 确保方法名在服务中唯一

## 🤝 贡献

欢迎提交更多语言的示例实现！

---

*作者：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*