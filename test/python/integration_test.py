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

# 添加examples-ipc/python到路径
neo_root = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
python_service_dir = os.path.join(neo_root, "examples-ipc", "python")
sys.path.insert(0, python_service_dir)

from service import DemoService


def test_http_api():
    """测试HTTP API调用"""
    print("=== 测试HTTP API ===")
    
    # 等待服务启动
    time.sleep(2)
    
    # 测试hello
    print("\n1. 测试hello方法:")
    try:
        response = requests.post(
            "http://localhost:8080/api/demo-service-python/hello",
            json={"name": "Integration Test"},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        print("   ✓ hello测试通过")
    except Exception as e:
        print(f"   ✗ hello测试失败: {e}")
    
    # 测试calculate
    print("\n2. 测试calculate操作:")
    try:
        response = requests.post(
            "http://localhost:8080/api/demo-service-python/calculate",
            json={"expression": "5 * 6"},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert response.json().get("result") == 30
        print("   ✓ calculate测试通过")
    except Exception as e:
        print(f"   ✗ calculate测试失败: {e}")
    
    # 测试echo
    print("\n3. 测试echo:")
    try:
        response = requests.post(
            "http://localhost:8080/api/demo-service-python/echo",
            json={"message": "Hello Neo Framework!"},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert response.json().get("echo") == "Hello Neo Framework!"
        print("   ✓ echo测试通过")
    except Exception as e:
        print(f"   ✗ echo测试失败: {e}")
    
    # 测试getTime
    print("\n4. 测试getTime:")
    try:
        response = requests.post(
            "http://localhost:8080/api/demo-service-python/getTime",
            json={},
            timeout=5
        )
        print(f"   状态码: {response.status_code}")
        print(f"   响应: {response.json()}")
        assert response.status_code == 200
        assert "time" in response.json()
        print("   ✓ getTime测试通过")
    except Exception as e:
        print(f"   ✗ getTime测试失败: {e}")
    
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
    print("启动Python Demo服务...")
    os.chdir(python_service_dir)
    service = DemoService()
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