# Neo Framework 测试

本目录包含Neo Framework的各种测试。

## 目录结构

```
test/
├── python/           # Python相关测试脚本
│   ├── test_complete_flow.py    # 完整流程测试
│   ├── test_direct_call.py      # 直接调用测试
│   ├── test_full_chain.py       # 完整链路测试
│   ├── test_ipc_client.py       # IPC客户端测试
│   ├── test_simple_client.py    # 简单客户端测试
│   └── test_stress.py           # 压力测试
├── integration/      # Go集成测试
│   ├── test_core_service.go     # 核心服务测试
│   ├── test_interface.go        # 接口测试
│   ├── test_list_services.go    # 服务列表测试
│   └── test_registry_debug.go   # 注册中心调试测试
└── stress/          # 压力测试相关
```

## 运行测试

### Python测试

```bash
# 运行单个测试
python test/python/test_simple_client.py

# 运行压力测试
python test/python/test_stress.py
```

### Go集成测试

```bash
# 运行特定测试
go run test/integration/test_core_service.go

# 运行所有Go测试
go test ./...
```

## 测试说明

### Python测试
- `test_ipc_client.py` - 测试基本的IPC连接和消息发送
- `test_stress.py` - 对HTTP网关进行压力测试
- `test_complete_flow.py` - 测试完整的HTTP->语言服务调用流程

### Go集成测试
- `test_core_service.go` - 测试核心服务功能
- `test_list_services.go` - 测试服务注册和发现
- `test_registry_debug.go` - 调试服务注册中心

## 注意事项

1. 运行测试前确保Neo Framework已启动
2. 语言服务测试需要先启动对应的服务（如`examples-ipc/python/service.py`）
3. 日志文件会生成在`logs/`目录中