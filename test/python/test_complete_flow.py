#!/usr/bin/env python3
"""
测试完整的调用流程，包含详细的错误处理和日志
"""
import requests
import time
import json

def test_complete_flow():
    """测试完整的HTTP->Go->IPC->Python调用流程"""
    base_url = "http://localhost:32080"
    
    print("🔍 开始测试完整调用流程...")
    
    # 1. 首先测试健康检查
    print("\n1️⃣ 测试健康检查...")
    try:
        response = requests.get(f"{base_url}/health", timeout=5)
        print(f"✅ 健康检查成功: {response.json()}")
    except Exception as e:
        print(f"❌ 健康检查失败: {e}")
        return False
    
    # 2. 等待一下确保服务完全启动
    print("\n⏳ 等待服务完全启动...")
    time.sleep(2)
    
    # 3. 测试API调用 - 加法
    print("\n2️⃣ 测试加法API调用...")
    try:
        payload = {"a": 5, "b": 3}
        print(f"请求数据: {payload}")
        
        response = requests.post(
            f"{base_url}/api/python.math/add",
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=15
        )
        
        print(f"HTTP状态码: {response.status_code}")
        print(f"响应头: {dict(response.headers)}")
        
        if response.status_code == 200:
            try:
                result = response.json()
                print(f"✅ 加法结果: {result}")
                if isinstance(result, dict) and "result" in result:
                    expected = payload["a"] + payload["b"]
                    if result["result"] == expected:
                        print(f"✅ 计算结果正确: {expected}")
                    else:
                        print(f"❌ 计算结果错误，期望 {expected}，得到 {result['result']}")
                else:
                    print(f"⚠️ 响应格式异常: {result}")
            except json.JSONDecodeError:
                print(f"❌ 无法解析JSON响应: {response.text}")
        else:
            print(f"❌ HTTP错误 {response.status_code}: {response.text}")
            
    except requests.exceptions.Timeout:
        print("❌ 请求超时 - 可能是服务间通信问题")
    except requests.exceptions.ConnectionError:
        print("❌ 连接错误 - 服务可能未启动")
    except Exception as e:
        print(f"❌ 未知错误: {e}")
    
    # 4. 测试乘法
    print("\n3️⃣ 测试乘法API调用...")
    try:
        payload = {"a": 4, "b": 7}
        print(f"请求数据: {payload}")
        
        response = requests.post(
            f"{base_url}/api/python.math/multiply",
            json=payload,
            headers={"Content-Type": "application/json"},
            timeout=15
        )
        
        print(f"HTTP状态码: {response.status_code}")
        
        if response.status_code == 200:
            try:
                result = response.json()
                print(f"✅ 乘法结果: {result}")
                if isinstance(result, dict) and "result" in result:
                    expected = payload["a"] * payload["b"]
                    if result["result"] == expected:
                        print(f"✅ 计算结果正确: {expected}")
                    else:
                        print(f"❌ 计算结果错误，期望 {expected}，得到 {result['result']}")
            except json.JSONDecodeError:
                print(f"❌ 无法解析JSON响应: {response.text}")
        else:
            print(f"❌ HTTP错误 {response.status_code}: {response.text}")
            
    except Exception as e:
        print(f"❌ 乘法测试失败: {e}")

if __name__ == "__main__":
    test_complete_flow()