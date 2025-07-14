#!/usr/bin/env python3
"""
简单的HTTP客户端，用于测试Neo框架的API
"""
import requests
import json
import sys


def call_api(service, method, data=None):
    """调用API并显示结果"""
    url = f"http://localhost:8080/api/{service}/{method}"
    
    try:
        if data:
            response = requests.post(url, json=data, timeout=5)
        else:
            response = requests.get(url, timeout=5)
        
        print(f"\n请求: {method}")
        print(f"URL: {url}")
        if data:
            print(f"数据: {json.dumps(data, indent=2)}")
        print(f"\n状态码: {response.status_code}")
        
        try:
            print(f"响应: {json.dumps(response.json(), indent=2)}")
        except:
            print(f"响应: {response.text}")
            
    except requests.exceptions.ConnectionError:
        print("错误: 无法连接到服务器，请确保网关服务正在运行")
    except requests.exceptions.Timeout:
        print("错误: 请求超时")
    except Exception as e:
        print(f"错误: {e}")


def main():
    """主函数"""
    print("Neo框架测试客户端")
    print("=" * 50)
    
    if len(sys.argv) < 2:
        print("\n使用方法:")
        print("  python test_client.py <command> [args...]")
        print("\n示例:")
        print("  python test_client.py add 10 20")
        print("  python test_client.py multiply 5 6")
        print("  python test_client.py calculate \"2 * (3 + 4)\"")
        print("  python test_client.py health")
        return
    
    command = sys.argv[1]
    
    if command == "add" and len(sys.argv) >= 4:
        a = float(sys.argv[2])
        b = float(sys.argv[3])
        call_api("python.math", "add", {"a": a, "b": b})
        
    elif command == "multiply" and len(sys.argv) >= 4:
        a = float(sys.argv[2])
        b = float(sys.argv[3])
        call_api("python.math", "multiply", {"a": a, "b": b})
        
    elif command == "calculate" and len(sys.argv) >= 3:
        expression = sys.argv[2]
        call_api("python.math", "calculate", {"expression": expression})
        
    elif command == "health":
        response = requests.get("http://localhost:8080/health", timeout=5)
        print(f"健康检查: {response.json()}")
        
    else:
        print(f"未知命令: {command}")


if __name__ == "__main__":
    main()