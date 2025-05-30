import unittest
import time
import threading
from concurrent.futures import ThreadPoolExecutor
from pool.pool import ConnectionPool
from pool.connection import ConnectionState

class TestConnectionPool(unittest.TestCase):
    def setUp(self):
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

    def tearDown(self):
        self.pool.close()

    def test_pool_initialization(self):
        """测试连接池初始化"""
        stats = self.pool.get_stats()
        self.assertEqual(stats['total_connections'], 2)  # min_size
        self.assertEqual(stats['idle_connections'], 2)
        self.assertEqual(stats['active_connections'], 0)

    def test_get_connection(self):
        """测试获取连接"""
        conn = self.pool.get_connection()
        self.assertIsNotNone(conn)
        self.assertEqual(conn.state, ConnectionState.IDLE)
        
        stats = self.pool.get_stats()
        self.assertEqual(stats['total_connections'], 2)

    def test_connection_lifecycle(self):
        """测试连接生命周期"""
        # 获取连接
        conn = self.pool.get_connection()
        self.assertIsNotNone(conn)
        
        # 模拟使用连接
        test_data = b"test data"
        success = conn.send(test_data)
        self.assertTrue(success)
        self.assertEqual(conn.state, ConnectionState.BUSY)
        
        # 归还连接
        self.pool.return_connection(conn)
        self.assertEqual(conn.state, ConnectionState.IDLE)
        
        # 验证统计信息
        stats = conn.get_stats()
        self.assertEqual(stats.total_bytes_sent, len(test_data))
        self.assertEqual(stats.total_requests, 1)

    def test_auto_scaling(self):
        """测试自动扩缩容"""
        # 获取所有初始连接
        conns = []
        for _ in range(2):  # min_size = 2
            conn = self.pool.get_connection()
            self.assertIsNotNone(conn)
            conns.append(conn)
        
        # 再请求一个连接，应该会创建新连接
        extra_conn = self.pool.get_connection()
        self.assertIsNotNone(extra_conn)
        conns.append(extra_conn)
        
        stats = self.pool.get_stats()
        self.assertEqual(stats['total_connections'], 3)  # 自动扩容
        
        # 归还所有连接
        for conn in conns:
            self.pool.return_connection(conn)
        
        # 等待自动缩容
        time.sleep(6)  # 大于 idle_timeout (5.0)
        
        stats = self.pool.get_stats()
        self.assertEqual(stats['total_connections'], 2)  # 缩容到 min_size

    def test_concurrent_access(self):
        """测试并发访问"""
        def worker():
            conn = self.pool.get_connection()
            self.assertIsNotNone(conn)
            time.sleep(0.1)  # 模拟工作负载
            self.pool.return_connection(conn)
            return True

        # 创建10个并发任务
        with ThreadPoolExecutor(max_workers=10) as executor:
            results = list(executor.map(lambda _: worker(), range(10)))
        
        # 验证所有任务都成功完成
        self.assertTrue(all(results))
        
        # 验证连接池状态
        stats = self.pool.get_stats()
        self.assertLessEqual(stats['total_connections'], 5)  # 不超过 max_size
        self.assertEqual(stats['active_connections'], 0)  # 所有连接都已归还

    def test_health_check(self):
        """测试健康检查"""
        # 获取一个连接并标记为错误状态
        conn = self.pool.get_connection()
        self.assertIsNotNone(conn)
        conn.state = ConnectionState.ERROR
        self.pool.return_connection(conn)
        
        # 等待健康检查移除错误连接
        time.sleep(2)  # 大于 health_check_interval (1.0)
        
        stats = self.pool.get_stats()
        self.assertEqual(stats['error_connections'], 0)  # 错误连接应被移除

    def test_load_balancing(self):
        """测试负载均衡"""
        # 获取并使用多个连接
        conns = {}
        for _ in range(10):
            conn = self.pool.get_connection()
            self.assertIsNotNone(conn)
            
            # 记录每个连接被选择的次数
            conn_id = id(conn)
            conns[conn_id] = conns.get(conn_id, 0) + 1
            
            # 模拟使用连接
            conn.send(b"test")
            self.pool.return_connection(conn)
        
        # 验证负载是否分散
        usage_counts = list(conns.values())
        self.assertTrue(max(usage_counts) - min(usage_counts) <= 2)  # 负载应该相对均衡

if __name__ == '__main__':
    unittest.main() 