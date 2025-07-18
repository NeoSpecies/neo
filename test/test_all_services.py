#!/usr/bin/env python3
"""
Neo Framework 全语言服务自动化测试脚本
测试所有支持的语言服务：Python, Go, Node.js, Java, PHP
"""
import subprocess
import time
import requests
import json
import os
import sys
import signal
from datetime import datetime
from typing import Dict, List, Tuple

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

def check_neo_framework():
    """检查Neo Framework是否运行"""
    try:
        response = requests.get("http://localhost:8080/health", timeout=2)
        if response.status_code == 200:
            print_colored("✓ Neo Framework 正在运行", Colors.GREEN)
            return True
    except:
        pass
    
    print_colored("✗ Neo Framework 未运行", Colors.RED)
    print("请先启动: go run cmd/neo/main.go")
    return False

def test_service_methods(service_name: str) -> Tuple[int, int]:
    """测试单个服务的所有方法"""
    base_url = "http://localhost:8080"
    passed = 0
    total = 0
    
    # 测试方法列表
    tests = [
        {
            "name": "hello",
            "data": {"name": "Test"},
            "expected_field": "message"
        },
        {
            "name": "calculate", 
            "data": {"a": 10, "b": 5, "operation": "add"},
            "expected_field": "result",
            "expected_value": 15
        },
        {
            "name": "echo",
            "data": {"message": "Hello Neo!"},
            "expected_field": "echo",
            "expected_value": "Hello Neo!"
        },
        {
            "name": "getTime",
            "data": {},
            "expected_field": "time"
        },
        {
            "name": "getInfo",
            "data": {},
            "expected_field": "service",
            "expected_value": service_name
        }
    ]
    
    for test in tests:
        total += 1
        try:
            response = requests.post(
                f"{base_url}/api/{service_name}/{test['name']}",
                json=test['data'],
                timeout=5
            )
            
            if response.status_code == 200:
                data = response.json()
                
                # 检查预期字段
                if test['expected_field'] in data:
                    # 如果有预期值，检查值是否匹配
                    if 'expected_value' in test:
                        if data[test['expected_field']] == test['expected_value']:
                            print_colored(f"  ✓ {test['name']}: 成功", Colors.GREEN)
                            passed += 1
                        else:
                            print_colored(f"  ✗ {test['name']}: 值不匹配", Colors.RED)
                            print(f"    预期: {test['expected_value']}, 实际: {data[test['expected_field']]}")
                    else:
                        print_colored(f"  ✓ {test['name']}: 成功", Colors.GREEN)
                        passed += 1
                else:
                    print_colored(f"  ✗ {test['name']}: 缺少字段 {test['expected_field']}", Colors.RED)
            else:
                print_colored(f"  ✗ {test['name']}: HTTP {response.status_code}", Colors.RED)
        except Exception as e:
            print_colored(f"  ✗ {test['name']}: {str(e)}", Colors.RED)
    
    return passed, total

def start_service(language: str, service_dir: str, command: List[str]) -> subprocess.Popen:
    """启动语言服务"""
    try:
        # 切换到服务目录
        os.chdir(service_dir)
        
        # 启动服务
        if os.name == 'nt':  # Windows
            process = subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                creationflags=subprocess.CREATE_NEW_PROCESS_GROUP
            )
        else:  # Unix/Linux
            process = subprocess.Popen(
                command,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                preexec_fn=os.setsid
            )
        
        # 等待服务启动
        time.sleep(3)
        
        # 检查进程是否还在运行
        if process.poll() is None:
            print_colored(f"✓ {language} 服务已启动 (PID: {process.pid})", Colors.GREEN)
            return process
        else:
            print_colored(f"✗ {language} 服务启动失败", Colors.RED)
            return None
            
    except Exception as e:
        print_colored(f"✗ 启动 {language} 服务失败: {e}", Colors.RED)
        return None

def stop_service(process: subprocess.Popen, language: str):
    """停止服务进程"""
    if process:
        try:
            if os.name == 'nt':  # Windows
                subprocess.call(['taskkill', '/F', '/T', '/PID', str(process.pid)], 
                              stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            else:  # Unix/Linux
                os.killpg(os.getpgid(process.pid), signal.SIGTERM)
            
            process.wait(timeout=5)
            print_colored(f"✓ {language} 服务已停止", Colors.YELLOW)
        except:
            try:
                process.kill()
            except:
                pass

def test_language_service(language: str, service_name: str, service_dir: str, command: List[str]) -> Tuple[int, int]:
    """测试单个语言服务"""
    print_header(f"测试 {language} 服务")
    
    # 检查服务目录是否存在
    if not os.path.exists(service_dir):
        print_colored(f"✗ 服务目录不存在: {service_dir}", Colors.RED)
        return 0, 5
    
    # 启动服务
    process = start_service(language, service_dir, command)
    if not process:
        return 0, 5
    
    try:
        # 测试服务方法
        passed, total = test_service_methods(service_name)
        
        # 显示结果
        if passed == total:
            print_colored(f"\n{language} 服务: {passed}/{total} 测试通过 ✓", Colors.GREEN)
        else:
            print_colored(f"\n{language} 服务: {passed}/{total} 测试通过", Colors.YELLOW)
        
        return passed, total
        
    finally:
        # 停止服务
        stop_service(process, language)
        # 返回原目录
        os.chdir(os.path.dirname(os.path.abspath(__file__)))

def main():
    """主测试函数"""
    print_colored("Neo Framework 全语言服务自动化测试", Colors.BOLD)
    print(f"测试时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    
    # 检查Neo Framework
    if not check_neo_framework():
        return 1
    
    # 获取项目根目录
    test_dir = os.path.dirname(os.path.abspath(__file__))
    root_dir = os.path.dirname(test_dir)
    
    # 语言服务配置
    services = [
        {
            "language": "Python",
            "service_name": "demo-service-python",
            "service_dir": os.path.join(root_dir, "examples-ipc", "python"),
            "command": [sys.executable, "service.py"]
        },
        {
            "language": "Go",
            "service_name": "demo-service-go",
            "service_dir": os.path.join(root_dir, "examples-ipc", "go"),
            "command": ["go", "run", "service.go"]
        },
        {
            "language": "Node.js",
            "service_name": "demo-service-nodejs",
            "service_dir": os.path.join(root_dir, "examples-ipc", "nodejs"),
            "command": ["node", "service.js"]
        },
        {
            "language": "Java",
            "service_name": "demo-service-java",
            "service_dir": os.path.join(root_dir, "examples-ipc", "java"),
            "command": ["java", "-cp", ".;gson-2.10.1.jar" if os.name == 'nt' else ".:gson-2.10.1.jar", "Service"]
        },
        {
            "language": "PHP",
            "service_name": "demo-service-php",
            "service_dir": os.path.join(root_dir, "examples-ipc", "php"),
            "command": ["php", "service.php"]
        }
    ]
    
    # 测试结果统计
    total_passed = 0
    total_tests = 0
    results = []
    
    # 测试每个服务
    for service in services:
        passed, total = test_language_service(
            service["language"],
            service["service_name"],
            service["service_dir"],
            service["command"]
        )
        total_passed += passed
        total_tests += total
        results.append({
            "language": service["language"],
            "passed": passed,
            "total": total
        })
    
    # 显示总结
    print_header("测试结果总结")
    
    for result in results:
        status = "✓" if result["passed"] == result["total"] else "✗"
        color = Colors.GREEN if result["passed"] == result["total"] else Colors.YELLOW
        print_colored(
            f"{status} {result['language']}: {result['passed']}/{result['total']} 测试通过",
            color
        )
    
    print("\n" + "-" * 60)
    success_rate = (total_passed / total_tests * 100) if total_tests > 0 else 0
    
    if total_passed == total_tests:
        print_colored(f"总计: {total_passed}/{total_tests} 测试通过 (100%) ✓", Colors.GREEN)
        print_colored("\n所有测试通过！", Colors.GREEN)
        return 0
    else:
        print_colored(f"总计: {total_passed}/{total_tests} 测试通过 ({success_rate:.1f}%)", Colors.YELLOW)
        print_colored(f"\n有 {total_tests - total_passed} 个测试失败", Colors.RED)
        return 1

if __name__ == "__main__":
    try:
        exit_code = main()
        sys.exit(exit_code)
    except KeyboardInterrupt:
        print_colored("\n\n测试被用户中断", Colors.YELLOW)
        sys.exit(1)
    except Exception as e:
        print_colored(f"\n测试出错: {e}", Colors.RED)
        sys.exit(1)