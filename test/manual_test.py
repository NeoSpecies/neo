#!/usr/bin/env python3
"""
Neo Framework 手动测试助手
提供交互式的手动测试工具
"""
import requests
import json
import sys
from datetime import datetime

# ANSI 颜色代码
class Colors:
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    ENDC = '\033[0m'
    BOLD = '\033[1m'

def print_colored(text: str, color: str):
    """打印彩色文本"""
    print(f"{color}{text}{Colors.ENDC}")

def print_header(title: str):
    """打印标题"""
    print("\n" + "=" * 60)
    print_colored(f"  {title}", Colors.BOLD)
    print("=" * 60)

def check_health():
    """健康检查"""
    try:
        response = requests.get("http://localhost:8080/health", timeout=2)
        if response.status_code == 200:
            data = response.json()
            print_colored(f"✓ Neo Framework 健康状态: {data['status']}", Colors.GREEN)
            print(f"  时间: {data.get('time', 'N/A')}")
            return True
    except Exception as e:
        print_colored(f"✗ 健康检查失败: {e}", Colors.RED)
    return False

def test_method(service_name: str, method: str, data: dict):
    """测试单个方法"""
    try:
        url = f"http://localhost:8080/api/{service_name}/{method}"
        print(f"\n请求 URL: {url}")
        print(f"请求数据: {json.dumps(data, indent=2)}")
        
        response = requests.post(url, json=data, timeout=10)
        
        print(f"\n响应状态码: {response.status_code}")
        
        if response.status_code == 200:
            result = response.json()
            print_colored("响应数据:", Colors.GREEN)
            print(json.dumps(result, indent=2, ensure_ascii=False))
        else:
            print_colored(f"错误响应: {response.text}", Colors.RED)
            
    except Exception as e:
        print_colored(f"请求失败: {e}", Colors.RED)

def manual_test_service(service_name: str):
    """手动测试服务的所有方法"""
    print_header(f"手动测试 {service_name}")
    
    # 测试用例
    test_cases = [
        ("1. Hello 方法", "hello", {"name": "Manual Test"}),
        ("2. Calculate 方法 (加法)", "calculate", {"a": 20, "b": 15, "operation": "add"}),
        ("3. Calculate 方法 (减法)", "calculate", {"a": 20, "b": 15, "operation": "subtract"}),
        ("4. Calculate 方法 (乘法)", "calculate", {"a": 20, "b": 15, "operation": "multiply"}),
        ("5. Calculate 方法 (除法)", "calculate", {"a": 20, "b": 5, "operation": "divide"}),
        ("6. Echo 方法", "echo", {"message": "Manual testing Neo Framework!"}),
        ("7. GetTime 方法", "getTime", {"format": "iso"}),
        ("8. GetInfo 方法", "getInfo", {})
    ]
    
    for title, method, data in test_cases:
        print_colored(f"\n{title}", Colors.BLUE)
        input("按 Enter 执行测试...")
        test_method(service_name, method, data)

def interactive_mode():
    """交互式测试模式"""
    print_header("交互式测试模式")
    
    services = {
        "1": "demo-service-python",
        "2": "demo-service-go",
        "3": "demo-service-nodejs",
        "4": "demo-service-java",
        "5": "demo-service-php"
    }
    
    while True:
        print("\n选择要测试的服务:")
        print("1. Python 服务")
        print("2. Go 服务")
        print("3. Node.js 服务")
        print("4. Java 服务")
        print("5. PHP 服务")
        print("6. 自定义测试")
        print("0. 退出")
        
        choice = input("\n请选择 (0-6): ").strip()
        
        if choice == "0":
            print_colored("退出测试", Colors.YELLOW)
            break
        elif choice in services:
            manual_test_service(services[choice])
        elif choice == "6":
            # 自定义测试
            service_name = input("输入服务名称: ").strip()
            method = input("输入方法名称: ").strip()
            
            # 构建请求数据
            data = {}
            print("输入请求参数 (输入空行结束):")
            while True:
                key = input("参数名: ").strip()
                if not key:
                    break
                value = input(f"{key} 的值: ").strip()
                
                # 尝试解析为数字
                try:
                    if '.' in value:
                        data[key] = float(value)
                    else:
                        data[key] = int(value)
                except:
                    data[key] = value
            
            test_method(service_name, method, data)
        else:
            print_colored("无效选择，请重试", Colors.RED)

def batch_test_all():
    """批量测试所有服务"""
    print_header("批量测试所有服务")
    
    services = [
        ("Python", "demo-service-python"),
        ("Go", "demo-service-go"),
        ("Node.js", "demo-service-nodejs"),
        ("Java", "demo-service-java"),
        ("PHP", "demo-service-php")
    ]
    
    for lang, service_name in services:
        print_colored(f"\n测试 {lang} 服务", Colors.BLUE)
        input(f"确保 {lang} 服务已启动，按 Enter 继续...")
        
        # 快速测试 hello 方法
        try:
            response = requests.post(
                f"http://localhost:8080/api/{service_name}/hello",
                json={"name": lang},
                timeout=5
            )
            if response.status_code == 200:
                data = response.json()
                print_colored(f"✓ {lang}: {data.get('message', 'Success')}", Colors.GREEN)
            else:
                print_colored(f"✗ {lang}: HTTP {response.status_code}", Colors.RED)
        except Exception as e:
            print_colored(f"✗ {lang}: {e}", Colors.RED)

def main():
    """主函数"""
    print_colored("Neo Framework 手动测试助手", Colors.BOLD)
    print(f"测试时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    
    # 健康检查
    if not check_health():
        print_colored("\n请先启动 Neo Framework:", Colors.YELLOW)
        print("go run cmd/neo/main.go")
        return 1
    
    while True:
        print("\n选择测试模式:")
        print("1. 交互式测试")
        print("2. 批量快速测试")
        print("3. 健康检查")
        print("0. 退出")
        
        choice = input("\n请选择 (0-3): ").strip()
        
        if choice == "0":
            break
        elif choice == "1":
            interactive_mode()
        elif choice == "2":
            batch_test_all()
        elif choice == "3":
            check_health()
        else:
            print_colored("无效选择", Colors.RED)
    
    print_colored("\n测试结束", Colors.YELLOW)
    return 0

if __name__ == "__main__":
    try:
        sys.exit(main())
    except KeyboardInterrupt:
        print_colored("\n\n测试被用户中断", Colors.YELLOW)
        sys.exit(1)