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
from python_service.neo_client import Message, MessageType

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
from python_service.example_service import PythonMathService
import asyncio

async def test_handlers():
    service = PythonMathService()
    
    # 测试加法
    result = await service.handle_add({"a": 15, "b": 25})
    assert result["result"] == 40
    print(f"   - 加法: 15 + 25 = {result['result']}")
    
    # 测试乘法
    result = await service.handle_multiply({"a": 7, "b": 8})
    assert result["result"] == 56
    print(f"   - 乘法: 7 * 8 = {result['result']}")
    
    # 测试计算
    result = await service.handle_calculate({"expression": "2 * (3 + 4)"})
    assert result["result"] == 14
    print(f"   - 表达式: 2 * (3 + 4) = {result['result']}")
    
    print("   ✓ 服务处理器测试通过")

asyncio.run(test_handlers())

# 3. 测试IPC客户端初始化
print("\n3. 测试IPC客户端:")
from python_service.neo_client import NeoIPCClient

client = NeoIPCClient("localhost", 9999)
print(f"   - 主机: {client.host}")
print(f"   - 端口: {client.port}")
print(f"   - 处理器数量: {len(client.handlers)}")
print("   ✓ IPC客户端初始化测试通过")

print("\n" + "=" * 50)
print("所有基础功能测试通过！")
print("\n下一步:")
print("1. 启动Go网关服务: go run cmd/gateway/main.go")
print("2. 启动Python服务: python python_service/example_service.py")
print("3. 运行集成测试: python test/integration_test.py")