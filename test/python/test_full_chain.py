#!/usr/bin/env python3
"""
完整的HTTP->Python调用链测试
"""
import asyncio
import time
import requests
import subprocess
import sys
import os
from pathlib import Path

async def run_python_service():
    """运行Python服务"""
    print("🔧 启动Python数学服务...")
    
    # 切换到python_service目录
    service_dir = Path(__file__).parent / "python_service"
    os.chdir(service_dir)
    
    # 导入并运行服务
    from neo_client import NeoIPCClient
    
    class QuickMathService:
        def __init__(self):
            self.client = NeoIPCClient(port=45999)
            
        async def start(self):
            await self.client.connect()
            await self.client.register_service("quick.math", {"test": "true"})
            self.client.register_handler("add", self.handle_add)
            
            print("✅ Python服务已注册并监听...")
            await self.client.listen()
            
        async def handle_add(self, data):
            a = data.get("a", 0)
            b = data.get("b", 0)
            result = a + b
            print(f"Python处理: {a} + {b} = {result}")
            return {"result": result}
    
    service = QuickMathService()
    await service.start()

def test_http_call():
    """测试HTTP调用"""
    print("🧪 测试HTTP调用...")
    
    url = "http://localhost:34081/api/quick.math/add"
    data = {"a": 10, "b": 20}
    
    try:
        response = requests.post(url, json=data, timeout=10)
        if response.status_code == 200:
            result = response.json()
            print(f"✅ HTTP调用成功: {result}")
            return True
        else:
            print(f"❌ HTTP调用失败: {response.status_code} - {response.text}")
            return False
    except Exception as e:
        print(f"❌ HTTP调用异常: {e}")
        return False

async def main():
    """主测试函数"""
    print("🚀 开始完整链路测试...")
    
    # 先手动启动Neo框架
    print("⚠️  请先手动启动Neo框架:")
    print("   go run cmd/neo/main.go -http :34081 -ipc :45999")
    print("   然后按回车继续...")
    input()
    
    # 启动Python服务
    try:
        await asyncio.wait_for(run_python_service(), timeout=60)
    except asyncio.TimeoutError:
        print("❌ Python服务启动超时")
    except Exception as e:
        print(f"❌ Python服务启动失败: {e}")

if __name__ == "__main__":
    asyncio.run(main())