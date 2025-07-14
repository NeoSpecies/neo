#!/usr/bin/env python3
"""
Neo Framework压力测试
"""
import asyncio
import aiohttp
import time
import json
from concurrent.futures import ThreadPoolExecutor

async def make_request(session, url, data, request_id):
    """发送单个HTTP请求"""
    try:
        start_time = time.time()
        async with session.post(url, json=data, timeout=aiohttp.ClientTimeout(total=30)) as response:
            end_time = time.time()
            duration = end_time - start_time
            
            if response.status == 200:
                result = await response.json()
                return {
                    "request_id": request_id,
                    "status": "success",
                    "duration": duration,
                    "result": result
                }
            else:
                text = await response.text()
                return {
                    "request_id": request_id,
                    "status": "error",
                    "duration": duration,
                    "http_status": response.status,
                    "error": text
                }
    except Exception as e:
        end_time = time.time()
        duration = end_time - start_time
        return {
            "request_id": request_id,
            "status": "exception",
            "duration": duration,
            "error": str(e)
        }

async def stress_test():
    """执行压力测试"""
    base_url = "http://localhost:28080"
    
    # 测试参数
    num_requests = 50  # 总请求数
    concurrent_requests = 10  # 并发数
    
    print(f"🧪 开始压力测试")
    print(f"   基础URL: {base_url}")
    print(f"   总请求数: {num_requests}")
    print(f"   并发数: {concurrent_requests}")
    print()
    
    # 准备测试数据
    test_cases = [
        {
            "url": f"{base_url}/api/python.math/add",
            "data": {"a": i, "b": i+1},
            "name": "add"
        }
        for i in range(num_requests)
    ]
    
    # 创建HTTP会话
    connector = aiohttp.TCPConnector(limit=concurrent_requests)
    async with aiohttp.ClientSession(connector=connector) as session:
        
        # 先测试健康检查
        print("🔍 检查服务状态...")
        try:
            async with session.get(f"{base_url}/health", timeout=aiohttp.ClientTimeout(total=5)) as response:
                if response.status == 200:
                    health_data = await response.json()
                    print(f"✅ 服务健康: {health_data}")
                else:
                    print(f"❌ 健康检查失败: {response.status}")
                    return
        except Exception as e:
            print(f"❌ 无法连接到服务: {e}")
            return
        
        print(f"\n🚀 开始发送 {num_requests} 个并发请求...")
        start_time = time.time()
        
        # 创建并发任务
        semaphore = asyncio.Semaphore(concurrent_requests)
        
        async def limited_request(test_case, request_id):
            async with semaphore:
                return await make_request(session, test_case["url"], test_case["data"], request_id)
        
        # 执行所有请求
        tasks = [
            limited_request(test_case, i) 
            for i, test_case in enumerate(test_cases)
        ]
        
        results = await asyncio.gather(*tasks)
        
        end_time = time.time()
        total_duration = end_time - start_time
        
        # 分析结果
        successful = [r for r in results if r["status"] == "success"]
        errors = [r for r in results if r["status"] == "error"]
        exceptions = [r for r in results if r["status"] == "exception"]
        
        print(f"\n📊 压力测试结果:")
        print(f"   总耗时: {total_duration:.2f}秒")
        print(f"   总请求数: {len(results)}")
        print(f"   成功: {len(successful)} ({len(successful)/len(results)*100:.1f}%)")
        print(f"   HTTP错误: {len(errors)} ({len(errors)/len(results)*100:.1f}%)")
        print(f"   异常: {len(exceptions)} ({len(exceptions)/len(results)*100:.1f}%)")
        print(f"   平均QPS: {len(results)/total_duration:.2f}")
        
        if successful:
            durations = [r["duration"] for r in successful]
            print(f"   平均响应时间: {sum(durations)/len(durations):.3f}秒")
            print(f"   最快响应: {min(durations):.3f}秒")
            print(f"   最慢响应: {max(durations):.3f}秒")
            
            # 显示一些成功的结果
            print(f"\n✅ 成功请求示例:")
            for i, result in enumerate(successful[:3]):
                print(f"   {i+1}. 请求{result['request_id']}: {result['result']} (耗时: {result['duration']:.3f}s)")
        
        if errors:
            print(f"\n❌ HTTP错误示例:")
            for i, error in enumerate(errors[:3]):
                print(f"   {i+1}. 请求{error['request_id']}: HTTP {error['http_status']} - {error['error']}")
        
        if exceptions:
            print(f"\n💥 异常示例:")
            for i, exc in enumerate(exceptions[:3]):
                print(f"   {i+1}. 请求{exc['request_id']}: {exc['error']}")

if __name__ == "__main__":
    asyncio.run(stress_test())