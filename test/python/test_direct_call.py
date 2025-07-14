#!/usr/bin/env python3
"""
直接测试HTTP请求，看看是否能看到日志输出
"""
import requests
import json

def test_direct_call():
    """测试直接HTTP调用"""
    try:
        # 测试健康检查
        print("🔍 测试健康检查...")
        response = requests.get("http://localhost:28080/health", timeout=5)
        print(f"✅ 健康检查响应: {response.json()}")
        
        # 测试API调用
        print("\n🔍 测试API调用...")
        response = requests.post(
            "http://localhost:28080/api/python.math/add",
            json={"a": 5, "b": 3},
            timeout=10
        )
        
        print(f"HTTP Status: {response.status_code}")
        print(f"Response Headers: {dict(response.headers)}")
        
        if response.content:
            print(f"Response Body: {response.text}")
        else:
            print("❌ 空响应")
            
    except requests.exceptions.Timeout:
        print("❌ 请求超时")
    except requests.exceptions.ConnectionError:
        print("❌ 连接错误")
    except Exception as e:
        print(f"❌ 请求失败: {e}")

if __name__ == "__main__":
    test_direct_call()