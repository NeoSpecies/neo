#!/usr/bin/env python3
"""
Neo Framework 全语言服务压力测试
测试所有语言服务的性能和并发处理能力
"""
import asyncio
import aiohttp
import time
import statistics
from typing import List, Dict, Any
import json

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

async def make_request(session: aiohttp.ClientSession, url: str, data: dict, request_id: int) -> Dict[str, Any]:
    """发送单个HTTP请求"""
    try:
        start_time = time.time()
        async with session.post(url, json=data, timeout=aiohttp.ClientTimeout(total=10)) as response:
            end_time = time.time()
            duration = (end_time - start_time) * 1000  # 转换为毫秒
            
            if response.status == 200:
                result = await response.json()
                return {
                    "request_id": request_id,
                    "status": "success",
                    "duration_ms": duration,
                    "result": result
                }
            else:
                return {
                    "request_id": request_id,
                    "status": "error",
                    "duration_ms": duration,
                    "http_status": response.status
                }
    except asyncio.TimeoutError:
        return {
            "request_id": request_id,
            "status": "timeout",
            "duration_ms": 10000
        }
    except Exception as e:
        return {
            "request_id": request_id,
            "status": "exception",
            "error": str(e)
        }

async def stress_test_service(service_name: str, num_requests: int = 100, concurrent_requests: int = 10):
    """对单个服务进行压力测试"""
    base_url = "http://localhost:8080"
    
    print_colored(f"\n开始测试 {service_name}", Colors.BLUE)
    print(f"  请求总数: {num_requests}")
    print(f"  并发数: {concurrent_requests}")
    
    # 准备测试数据 - 混合不同的方法
    test_cases = []
    methods = [
        ("hello", {"name": f"User{i}"}),
        ("calculate", {"a": i, "b": i+1, "operation": "add"}),
        ("echo", {"message": f"Message {i}"}),
        ("getTime", {}),
        ("getInfo", {})
    ]
    
    for i in range(num_requests):
        method, data = methods[i % len(methods)]
        test_cases.append({
            "url": f"{base_url}/api/{service_name}/{method}",
            "data": data,
            "method": method
        })
    
    # 创建HTTP会话
    connector = aiohttp.TCPConnector(limit=concurrent_requests)
    async with aiohttp.ClientSession(connector=connector) as session:
        
        # 预热请求
        print("  预热中...")
        for _ in range(5):
            await make_request(session, f"{base_url}/api/{service_name}/hello", {"name": "warmup"}, -1)
        
        # 开始压力测试
        print("  开始压力测试...")
        start_time = time.time()
        
        # 创建并发限制
        semaphore = asyncio.Semaphore(concurrent_requests)
        
        async def limited_request(test_case, request_id):
            async with semaphore:
                return await make_request(session, test_case["url"], test_case["data"], request_id)
        
        # 执行所有请求
        tasks = [limited_request(test_case, i) for i, test_case in enumerate(test_cases)]
        results = await asyncio.gather(*tasks)
        
        end_time = time.time()
        total_duration = end_time - start_time
        
        # 分析结果
        success_results = [r for r in results if r["status"] == "success"]
        error_results = [r for r in results if r["status"] == "error"]
        timeout_results = [r for r in results if r["status"] == "timeout"]
        exception_results = [r for r in results if r["status"] == "exception"]
        
        # 计算统计数据
        if success_results:
            durations = [r["duration_ms"] for r in success_results]
            avg_duration = statistics.mean(durations)
            median_duration = statistics.median(durations)
            min_duration = min(durations)
            max_duration = max(durations)
            p95_duration = sorted(durations)[int(len(durations) * 0.95)] if len(durations) > 20 else max_duration
            p99_duration = sorted(durations)[int(len(durations) * 0.99)] if len(durations) > 100 else max_duration
        else:
            avg_duration = median_duration = min_duration = max_duration = p95_duration = p99_duration = 0
        
        # 显示结果
        print(f"\n  测试结果:")
        print(f"  总耗时: {total_duration:.2f} 秒")
        print(f"  吞吐量: {num_requests / total_duration:.2f} 请求/秒")
        print(f"  成功: {len(success_results)} ({len(success_results)/num_requests*100:.1f}%)")
        
        if error_results:
            print_colored(f"  错误: {len(error_results)} ({len(error_results)/num_requests*100:.1f}%)", Colors.RED)
        if timeout_results:
            print_colored(f"  超时: {len(timeout_results)} ({len(timeout_results)/num_requests*100:.1f}%)", Colors.RED)
        if exception_results:
            print_colored(f"  异常: {len(exception_results)} ({len(exception_results)/num_requests*100:.1f}%)", Colors.RED)
        
        if success_results:
            print(f"\n  响应时间统计 (毫秒):")
            print(f"  平均值: {avg_duration:.2f}")
            print(f"  中位数: {median_duration:.2f}")
            print(f"  最小值: {min_duration:.2f}")
            print(f"  最大值: {max_duration:.2f}")
            print(f"  P95: {p95_duration:.2f}")
            print(f"  P99: {p99_duration:.2f}")
        
        return {
            "service": service_name,
            "total_requests": num_requests,
            "success_rate": len(success_results) / num_requests * 100,
            "throughput": num_requests / total_duration,
            "avg_latency": avg_duration,
            "p95_latency": p95_duration,
            "p99_latency": p99_duration
        }

async def main():
    """主函数"""
    print_colored("Neo Framework 全语言服务压力测试", Colors.BOLD)
    print("=" * 60)
    
    # 检查服务健康状态
    try:
        async with aiohttp.ClientSession() as session:
            async with session.get("http://localhost:8080/health", timeout=aiohttp.ClientTimeout(total=2)) as response:
                if response.status == 200:
                    print_colored("✓ Neo Framework 正在运行", Colors.GREEN)
                else:
                    print_colored("✗ Neo Framework 健康检查失败", Colors.RED)
                    return
    except:
        print_colored("✗ 无法连接到 Neo Framework", Colors.RED)
        print("请先启动: go run cmd/neo/main.go")
        return
    
    # 测试配置
    services = [
        "demo-service-python",
        "demo-service-go",
        "demo-service-nodejs",
        "demo-service-java",
        "demo-service-php"
    ]
    
    # 询问测试参数
    print("\n测试参数设置:")
    try:
        num_requests = int(input("每个服务的请求总数 (默认100): ") or "100")
        concurrent_requests = int(input("并发请求数 (默认10): ") or "10")
    except:
        num_requests = 100
        concurrent_requests = 10
    
    # 测试结果汇总
    all_results = []
    
    # 对每个服务进行压力测试
    for service in services:
        print(f"\n{'='*60}")
        
        # 询问是否测试该服务
        test_service = input(f"测试 {service}? (y/n, 默认y): ").lower() != 'n'
        if not test_service:
            continue
        
        # 确保服务已启动
        print(f"请确保 {service} 已启动")
        input("按 Enter 继续...")
        
        try:
            result = await stress_test_service(service, num_requests, concurrent_requests)
            all_results.append(result)
        except Exception as e:
            print_colored(f"测试 {service} 时出错: {e}", Colors.RED)
    
    # 显示汇总结果
    if all_results:
        print_colored(f"\n{'='*60}", Colors.BOLD)
        print_colored("压力测试汇总", Colors.BOLD)
        print_colored(f"{'='*60}", Colors.BOLD)
        
        print(f"\n{'服务':<25} {'成功率':<10} {'吞吐量':<15} {'平均延迟':<12} {'P95延迟':<12} {'P99延迟':<12}")
        print("-" * 95)
        
        for result in all_results:
            success_color = Colors.GREEN if result['success_rate'] >= 95 else Colors.YELLOW if result['success_rate'] >= 80 else Colors.RED
            
            print(f"{result['service']:<25} "
                  f"{success_color}{result['success_rate']:>6.1f}%{Colors.ENDC}    "
                  f"{result['throughput']:>10.1f} req/s  "
                  f"{result['avg_latency']:>8.2f} ms  "
                  f"{result['p95_latency']:>8.2f} ms  "
                  f"{result['p99_latency']:>8.2f} ms")
        
        # 找出最佳和最差性能
        best_throughput = max(all_results, key=lambda x: x['throughput'])
        worst_throughput = min(all_results, key=lambda x: x['throughput'])
        best_latency = min(all_results, key=lambda x: x['avg_latency'])
        worst_latency = max(all_results, key=lambda x: x['avg_latency'])
        
        print(f"\n最高吞吐量: {best_throughput['service']} ({best_throughput['throughput']:.1f} req/s)")
        print(f"最低延迟: {best_latency['service']} ({best_latency['avg_latency']:.2f} ms)")
        
        if len(all_results) > 1:
            print(f"\n最低吞吐量: {worst_throughput['service']} ({worst_throughput['throughput']:.1f} req/s)")
            print(f"最高延迟: {worst_latency['service']} ({worst_latency['avg_latency']:.2f} ms)")

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print_colored("\n\n压力测试被用户中断", Colors.YELLOW)
    except Exception as e:
        print_colored(f"\n压力测试出错: {e}", Colors.RED)