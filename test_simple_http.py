"""简单测试HTTP调用"""
import time
import requests

print("等待5秒确保服务稳定...")
time.sleep(5)

print("\n测试HTTP调用到demo-service...")

# 尝试直接调用
try:
    print("\n1. 测试 hello 方法:")
    response = requests.post(
        "http://localhost:8080/api/demo-service/hello",
        json={"name": "Direct Test"},
        headers={"Content-Type": "application/json"},
        timeout=30  # 增加超时时间
    )
    print(f"Status: {response.status_code}")
    print(f"Response: {response.text}")
except Exception as e:
    print(f"Error: {e}")

# 测试健康检查
try:
    print("\n2. 测试健康检查:")
    response = requests.get("http://localhost:8080/health", timeout=5)
    print(f"Health check: {response.status_code} - {response.text}")
except Exception as e:
    print(f"Health check error: {e}")

# 列出注册的服务（如果有这个端点）
try:
    print("\n3. 尝试获取服务列表:")
    response = requests.get("http://localhost:8080/services", timeout=5)
    print(f"Services: {response.status_code} - {response.text}")
except Exception as e:
    print(f"Services error: {e}")