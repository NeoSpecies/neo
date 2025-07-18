# Neo Framework 测试脚本索引

本文档说明了test目录中各个测试脚本的用途和使用方法。

## 主要测试脚本

### 1. 自动化测试

#### test_all_services.py
**用途**: 自动化测试所有语言服务的完整功能
- 自动启动和停止各语言服务
- 测试所有服务方法（hello, calculate, echo, getTime, getInfo）
- 生成测试报告和统计信息
- 支持并发测试

**使用方法**:
```bash
cd test
python test_all_services.py
```

### 2. 手动测试

#### manual_test.py
**用途**: 交互式手动测试工具
- 提供菜单式界面
- 支持单个服务的详细测试
- 自定义测试参数
- 批量快速测试

**使用方法**:
```bash
cd test
python manual_test.py
```

### 3. 压力测试

#### stress/test_stress.py
**用途**: 基础压力测试脚本
- 测试单个服务的并发处理能力
- 默认测试Python服务的calculate方法

**使用方法**:
```bash
cd test/stress
python test_stress.py
```

#### stress/stress_test_all.py
**用途**: 全语言服务压力测试
- 支持测试所有语言服务
- 可配置请求数和并发数
- 提供详细的性能统计（延迟、吞吐量、P95/P99）
- 生成性能对比报告

**使用方法**:
```bash
cd test/stress
python stress_test_all.py
```

### 4. 批处理脚本

#### run_tests.bat (Windows)
**用途**: Windows批处理测试脚本
- 支持运行不同类型的测试
- 自动检查Python环境

**使用方法**:
```bash
# 运行所有测试
test\run_tests.bat all

# 运行Python测试
test\run_tests.bat python

# 运行压力测试
test\run_tests.bat stress
```

#### run_tests.sh (Unix/Linux)
**用途**: Unix/Linux Shell测试脚本
- 功能同Windows版本

**使用方法**:
```bash
# 添加执行权限
chmod +x test/run_tests.sh

# 运行测试
./test/run_tests.sh all
```

## 语言特定测试

### Python测试目录 (test/python/)

#### integration_test.py
**用途**: Python服务集成测试
- 测试HTTP到Python服务的完整调用链
- 包含服务启动和测试逻辑

#### simple_test.py
**用途**: Python服务基础功能测试
- 测试消息结构
- 测试服务处理器
- 测试IPC客户端初始化

#### test_full_chain.py
**用途**: 完整调用链测试
- 测试自定义服务注册
- 测试异步请求处理

### Go集成测试 (test/integration/)

#### test_core_service.go
**用途**: 核心服务功能测试

#### test_list_services.go
**用途**: 服务列表功能测试

#### test_registry_debug.go
**用途**: 服务注册调试测试

#### test_interface.go
**用途**: 接口测试

## 测试前准备

1. **启动Neo Framework**:
   ```bash
   go run cmd/neo/main.go
   ```

2. **检查端口**:
   - HTTP网关: 8080
   - IPC服务器: 9999

3. **安装依赖**:
   ```bash
   # Python依赖
   pip install requests aiohttp
   
   # Java需要Gson库
   # 下载: https://repo1.maven.org/maven2/com/google/code/gson/gson/2.10.1/gson-2.10.1.jar
   
   # PHP需要启用sockets扩展
   # 编辑php.ini，启用: extension=sockets
   ```

## 测试顺序建议

1. 先运行自动化测试验证基本功能: `python test_all_services.py`
2. 使用手动测试工具进行详细测试: `python manual_test.py`
3. 最后运行压力测试评估性能: `python stress/stress_test_all.py`

## 注意事项

- 确保Neo Framework已启动
- 某些测试需要先启动对应的语言服务
- PHP服务可能需要特殊配置（sockets扩展）
- Java服务需要Gson库支持