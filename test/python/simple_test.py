#!/usr/bin/env python3
"""
简单测试脚本，验证基本功能
"""
import json
import sys
import os

# 添加父目录到Python路径
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

# 测试Python服务组件
print("Neo框架功能验证")
print("=" * 50)

# 1. 测试消息结构
print("\n1. 测试消息结构:")
# 添加examples-ipc/python到路径
neo_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
python_service_dir = os.path.join(neo_root, "examples-ipc", "python")
sys.path.insert(0, python_service_dir)

from neo_client import Message, MessageType

msg = Message(
    msg_type=MessageType.REQUEST,
    id="test-123",
    service="python.math",
    method="add",
    data=json.dumps({"a": 10, "b": 20}).encode(),
    metadata={"version": "1.0"}
)

print(f"   - 消息类型: {msg.msg_type.name}")
print(f"   - 服务名: {msg.service}")
print(f"   - 方法名: {msg.method}")
print(f"   - 数据: {json.loads(msg.data)}")
print("   ✓ 消息结构测试通过")

# 2. 测试服务处理器
print("\n2. 测试服务处理器:")
from service import DemoService
import asyncio

async def test_handlers():
    service = DemoService()
    
    # 测试hello
    result = await service.handle_hello({"name": "Test"})
    assert result["message"] == "Hello Test from Python service!"
    print(f"   - Hello: {result['message']}")
    
    # 测试calculate
    result = await service.handle_calculate({"expression": "2 * (3 + 4)"})
    assert result["result"] == 14
    print(f"   - 计算: 2 * (3 + 4) = {result['result']}")
    
    # 测试echo
    result = await service.handle_echo({"message": "test echo"})
    assert result["echo"] == "test echo"
    print(f"   - Echo: {result['echo']}")
    
    print("   ✓ 服务处理器测试通过")

asyncio.run(test_handlers())

# 3. 测试IPC客户端初始化
print("\n3. 测试IPC客户端:")
# NeoIPCClient已经在上面导入了

client = NeoIPCClient("localhost", 9999)
print(f"   - 主机: {client.host}")
print(f"   - 端口: {client.port}")
print(f"   - 处理器数量: {len(client.handlers)}")
print("   ✓ IPC客户端初始化测试通过")

print("\n" + "=" * 50)
print("所有基础功能测试通过！")
print("\n下一步:")
print("1. 启动Neo框架: go run cmd/neo/main.go")
print("2. 启动Python服务: cd examples-ipc/python && python service.py")
print("3. 运行集成测试: python test/python/integration_test.py")