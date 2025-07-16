"""测试 IPC 服务通过 HTTP 网关的调用"""
import subprocess
import time
import requests
import json
import sys

def test_service(lang, service_path):
    """测试指定语言的服务"""
    print(f"\n{'='*50}")
    print(f"Testing {lang} Service")
    print('='*50)
    
    # 启动服务
    print(f"Starting {lang} service...")
    if lang == "Python":
        cmd = ["python", "service.py"]
    elif lang == "Go":
        cmd = ["go", "run", "service.go"]
    elif lang == "Node.js":
        cmd = ["node", "service.js"]
    else:
        return False
    
    proc = subprocess.Popen(cmd, cwd=service_path)
    time.sleep(3)  # 等待服务启动
    
    # 测试各个方法
    test_methods = [
        {
            "method": "hello",
            "data": {"name": f"{lang} User"},
            "expected_keys": ["message", "timestamp"]
        },
        {
            "method": "calculate",
            "data": {"a": 10, "b": 5, "operation": "multiply"},
            "expected_keys": ["result", "operation", "a", "b"]
        },
        {
            "method": "echo",
            "data": {"message": f"Hello from {lang}!"},
            "expected_keys": ["echo", "length", "reversed"]
        },
        {
            "method": "getTime",
            "data": {"format": "readable"},
            "expected_keys": ["time", "format"]
        },
        {
            "method": "getInfo",
            "data": {},
            "expected_keys": ["service", "language", "version", "handlers"]
        }
    ]
    
    success_count = 0
    
    for test in test_methods:
        print(f"\nTesting {test['method']}...")
        try:
            response = requests.post(
                f"http://localhost:8080/api/demo-service/{test['method']}",
                json=test['data'],
                headers={"Content-Type": "application/json"},
                timeout=5
            )
            
            if response.status_code == 200:
                result = response.json()
                print(f"✓ {test['method']}: Success")
                print(f"  Response: {json.dumps(result, indent=2)}")
                
                # 验证响应包含预期的键
                missing_keys = [key for key in test['expected_keys'] if key not in result]
                if missing_keys:
                    print(f"  ⚠ Missing expected keys: {missing_keys}")
                else:
                    success_count += 1
            else:
                print(f"✗ {test['method']}: HTTP {response.status_code}")
                print(f"  Response: {response.text}")
                
        except requests.exceptions.Timeout:
            print(f"✗ {test['method']}: Timeout")
        except requests.exceptions.ConnectionError:
            print(f"✗ {test['method']}: Connection error")
        except Exception as e:
            print(f"✗ {test['method']}: {type(e).__name__}: {e}")
    
    # 停止服务
    print(f"\nStopping {lang} service...")
    proc.terminate()
    time.sleep(1)
    
    print(f"\nResults: {success_count}/{len(test_methods)} tests passed")
    return success_count == len(test_methods)

def main():
    """主测试函数"""
    print("HTTP Service Integration Test")
    print("=============================")
    
    # 检查Neo Framework是否运行
    print("Checking Neo Framework...")
    try:
        response = requests.get("http://localhost:8080/health", timeout=2)
        if response.status_code == 200:
            print("✓ Neo Framework is running")
        else:
            print("✗ Neo Framework health check failed")
            return
    except:
        print("✗ Neo Framework is not running on port 8080")
        return
    
    # 测试各语言服务
    test_cases = [
        ("Python", "examples-ipc/python"),
        ("Go", "examples-ipc/go"),
        ("Node.js", "examples-ipc/nodejs")
    ]
    
    results = []
    for lang, path in test_cases:
        result = test_service(lang, path)
        results.append((lang, result))
        time.sleep(2)  # 服务间间隔
    
    # 总结
    print(f"\n{'='*50}")
    print("Test Summary")
    print('='*50)
    for lang, passed in results:
        status = "✓ PASSED" if passed else "✗ FAILED"
        print(f"{lang}: {status}")

if __name__ == "__main__":
    main()