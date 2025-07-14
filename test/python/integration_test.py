#!/usr/bin/env python3
"""
集成测试：验证HTTP->Go->IPC->Python的完整流程
"""
import asyncio
import json
import time
import requests
import sys
import os

# 添加父目录到Python路径
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from python_service.example_service import PythonMathService


def test_http_api():
    """测试HTTP API调用"""
    print("=== 测试HTTP API ===")
    
    # 等待服务启动
    time.sleep(2)
    
    # 测试加法
    print("\n1. 测试加法操作:")
    try:
        response = requests.post(
            "http://localhost:8080/api/python.math/add",
            json={"a": 10, "b": 20},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert response.json().get("result") == 30
        print("   ✓ 加法测试通过")
    except Exception as e:
        print(f"   ✗ 加法测试失败: {e}")
    
    # 测试乘法
    print("\n2. 测试乘法操作:")
    try:
        response = requests.post(
            "http://localhost:8080/api/python.math/multiply",
            json={"a": 5, "b": 6},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert response.json().get("result") == 30
        print("   ✓ 乘法测试通过")
    except Exception as e:
        print(f"   ✗ 乘法测试失败: {e}")
    
    # 测试复杂计算
    print("\n3. 测试复杂计算:")
    try:
        response = requests.post(
            "http://localhost:8080/api/python.math/calculate",
            json={"expression": "2 * (3 + 4)"},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert response.json().get("result") == 14
        print("   ✓ 复杂计算测试通过")
    except Exception as e:
        print(f"   ✗ 复杂计算测试失败: {e}")
    
    # 测试错误处理
    print("\n4. 测试错误处理:")
    try:
        response = requests.post(
            "http://localhost:8080/api/python.math/unknown",
            json={},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.text}")
        assert response.status_code in [404, 503]
        print("   ✓ 错误处理测试通过")
    except Exception as e:
        print(f"   ✗ 错误处理测试失败: {e}")
    
    # 测试健康检查
    print("\n5. 测试健康检查:")
    try:
        response = requests.get("http://localhost:8080/health", timeout=5)
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert response.json().get("status") == "healthy"
        print("   ✓ 健康检查测试通过")
    except Exception as e:
        print(f"   ✗ 健康检查测试失败: {e}")


async def run_python_service():
    """运行Python服务"""
    print("启动Python Math服务...")
    service = PythonMathService()
    try:
        await service.start()
    except Exception as e:
        print(f"Python服务错误: {e}")


def main():
    """主测试函数"""
    print("Neo框架集成测试")
    print("=" * 50)
    print("\n请确保以下服务正在运行:")
    print("1. Go网关服务 (端口8080)")
    print("2. IPC服务器 (端口9999)")
    print("\n启动测试...")
    
    # 在独立进程中运行Python服务
    import multiprocessing
    from multiprocessing import Process
    
    def run_service():
        asyncio.run(run_python_service())
    
    # 启动Python服务进程
    service_process = Process(target=run_service)
    service_process.start()
    
    try:
        # 等待服务启动
        print("\n等待服务启动...")
        time.sleep(3)
        
        # 运行测试
        test_http_api()
        
        print("\n" + "=" * 50)
        print("测试完成!")
        
    finally:
        # 终止服务进程
        print("\n停止Python服务...")
        service_process.terminate()
        service_process.join()


if __name__ == "__main__":
    main()