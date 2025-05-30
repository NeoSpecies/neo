import unittest
import time
import threading
import requests
from concurrent.futures import ThreadPoolExecutor
from pool.pool import ConnectionPool
from metrics.server import MetricsServer

class TestMetrics(unittest.TestCase):
    def setUp(self):
        # 启动指标服务器
        self.metrics_server = MetricsServer(host='localhost', port=8000)
        self.metrics_server.start()
        
        # 创建连接池
        self.pool = ConnectionPool(
            host='localhost',
            port=9090,
            min_size=2,
            max_size=5,
            connection_timeout=1.0,
            idle_timeout=5.0,
            max_lifetime=10.0,
            health_check_interval=1.0
        )
        
        # 等待服务器启动
        time.sleep(1)

    def tearDown(self):
        self.pool.close()
        self.metrics_server.stop()

    def test_basic_metrics(self):
        """测试基本指标收集"""
        # 获取初始指标
        response = requests.get('http://localhost:8000/metrics')
        self.assertEqual(response.status_code, 200)
        initial_metrics = response.text
        
        # 验证连接池指标存在
        self.assertIn('ipc_pool_connections_total', initial_metrics)
        self.assertIn('ipc_pool_connections_active', initial_metrics)
        self.assertIn('ipc_pool_connections_idle', initial_metrics)
        
        # 验证初始连接数
        self.assertIn('ipc_pool_connections_total{host="localhost",port="9090"} 2.0', 
                     initial_metrics)

    def test_request_metrics(self):
        """测试请求指标收集"""
        # 模拟一些请求
        for _ in range(5):
            conn = self.pool.get_connection()
            self.assertIsNotNone(conn)
            conn.send(b"test data")
            self.pool.return_connection(conn)
        
        # 获取更新后的指标
        response = requests.get('http://localhost:8000/metrics')
        updated_metrics = response.text
        
        # 验证请求计数器
        self.assertIn('ipc_requests_total{host="localhost",port="9090",status="success"}', 
                     updated_metrics)
        
        # 验证字节计数器
        self.assertIn('ipc_request_bytes_sent_total{host="localhost",port="9090"}', 
                     updated_metrics)

    def test_error_metrics(self):
        """测试错误指标收集"""
        # 模拟一些错误
        conn = self.pool.get_connection()
        self.assertIsNotNone(conn)
        conn.state = ConnectionState.ERROR
        self.pool.return_connection(conn)
        
        # 获取更新后的指标
        response = requests.get('http://localhost:8000/metrics')
        error_metrics = response.text
        
        # 验证错误计数器
        self.assertIn('ipc_connection_errors_total{error_type="connection_error"', 
                     error_metrics)

    def test_scaling_metrics(self):
        """测试扩缩容指标收集"""
        # 触发扩容
        conns = []
        for _ in range(5):  # 超过 min_size
            conn = self.pool.get_connection()
            self.assertIsNotNone(conn)
            conns.append(conn)
        
        # 获取扩容后的指标
        response = requests.get('http://localhost:8000/metrics')
        scaling_metrics = response.text
        
        # 验证扩容操作计数器
        self.assertIn('ipc_pool_scaling_operations_total{operation="scale_up"', 
                     scaling_metrics)
        
        # 归还连接并等待缩容
        for conn in conns:
            self.pool.return_connection(conn)
        time.sleep(6)  # 等待自动缩容
        
        # 获取缩容后的指标
        response = requests.get('http://localhost:8000/metrics')
        scaling_metrics = response.text
        
        # 验证缩容操作计数器
        self.assertIn('ipc_pool_scaling_operations_total{operation="scale_down"', 
                     scaling_metrics)

    def test_health_check_metrics(self):
        """测试健康检查指标收集"""
        # 等待健康检查执行
        time.sleep(2)
        
        # 获取健康检查指标
        response = requests.get('http://localhost:8000/metrics')
        health_metrics = response.text
        
        # 验证健康检查计数器
        self.assertIn('ipc_health_check_operations_total{result="healthy"', 
                     health_metrics)

    def test_concurrent_metrics(self):
        """测试并发场景下的指标收集"""
        def worker():
            conn = self.pool.get_connection()
            self.assertIsNotNone(conn)
            conn.send(b"test")
            time.sleep(0.1)  # 模拟工作负载
            self.pool.return_connection(conn)
            return True

        # 创建多个并发请求
        with ThreadPoolExecutor(max_workers=10) as executor:
            results = list(executor.map(lambda _: worker(), range(10)))
        
        # 验证所有请求都成功
        self.assertTrue(all(results))
        
        # 获取并发后的指标
        response = requests.get('http://localhost:8000/metrics')
        concurrent_metrics = response.text
        
        # 验证负载均衡决策计数器
        self.assertIn('ipc_balancer_decisions_total', concurrent_metrics)
        
        # 验证请求延迟直方图
        self.assertIn('ipc_request_latency_seconds_bucket', concurrent_metrics)

if __name__ == '__main__':
    unittest.main() 