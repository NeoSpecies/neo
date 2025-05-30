from prometheus_client import Counter, Gauge, Histogram, Info
from typing import Dict, Any
import threading

class PrometheusMetrics:
    """Prometheus 指标收集器"""
    
    def __init__(self):
        # 连接池指标
        self.pool_total_connections = Gauge(
            'ipc_pool_connections_total',
            '连接池中的总连接数',
            ['host', 'port']
        )
        self.pool_active_connections = Gauge(
            'ipc_pool_connections_active',
            '活动连接数',
            ['host', 'port']
        )
        self.pool_idle_connections = Gauge(
            'ipc_pool_connections_idle',
            '空闲连接数',
            ['host', 'port']
        )
        self.pool_error_connections = Gauge(
            'ipc_pool_connections_error',
            '错误连接数',
            ['host', 'port']
        )
        
        # 请求指标
        self.request_total = Counter(
            'ipc_requests_total',
            '总请求数',
            ['host', 'port', 'status']
        )
        self.request_bytes_sent = Counter(
            'ipc_request_bytes_sent_total',
            '发送的总字节数',
            ['host', 'port']
        )
        self.request_bytes_received = Counter(
            'ipc_request_bytes_received_total',
            '接收的总字节数',
            ['host', 'port']
        )
        
        # 延迟指标
        self.request_latency = Histogram(
            'ipc_request_latency_seconds',
            '请求延迟分布',
            ['host', 'port'],
            buckets=(0.001, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1.0, 2.5, 5.0, 7.5, 10.0)
        )
        
        # 连接池使用率
        self.pool_usage_ratio = Gauge(
            'ipc_pool_usage_ratio',
            '连接池使用率',
            ['host', 'port']
        )
        
        # 连接生命周期指标
        self.connection_lifetime = Histogram(
            'ipc_connection_lifetime_seconds',
            '连接生命周期分布',
            ['host', 'port'],
            buckets=(60, 300, 600, 1800, 3600, 7200, 14400, 28800, 43200, 86400)
        )
        
        # 连接错误指标
        self.connection_errors = Counter(
            'ipc_connection_errors_total',
            '连接错误总数',
            ['host', 'port', 'error_type']
        )
        
        # 负载均衡指标
        self.balancer_decisions = Counter(
            'ipc_balancer_decisions_total',
            '负载均衡器决策次数',
            ['host', 'port', 'strategy']
        )
        
        # 自动扩缩容指标
        self.pool_scaling_operations = Counter(
            'ipc_pool_scaling_operations_total',
            '自动扩缩容操作次数',
            ['host', 'port', 'operation']
        )
        
        # 健康检查指标
        self.health_check_operations = Counter(
            'ipc_health_check_operations_total',
            '健康检查操作次数',
            ['host', 'port', 'result']
        )
        
        # 系统信息
        self.ipc_info = Info('ipc_system', 'IPC系统信息')
        
        self._lock = threading.Lock()

    def update_pool_metrics(self, host: str, port: int, stats: Dict[str, Any]):
        """更新连接池指标"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.pool_total_connections.labels(**labels).set(
                stats['total_connections']
            )
            self.pool_active_connections.labels(**labels).set(
                stats['active_connections']
            )
            self.pool_idle_connections.labels(**labels).set(
                stats['idle_connections']
            )
            self.pool_error_connections.labels(**labels).set(
                stats['error_connections']
            )
            self.pool_usage_ratio.labels(**labels).set(
                stats['connection_usage_ratio']
            )

    def record_request(self, host: str, port: int, status: str, 
                      bytes_sent: int, bytes_received: int, 
                      latency: float):
        """记录请求指标"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.request_total.labels(status=status, **labels).inc()
            self.request_bytes_sent.labels(**labels).inc(bytes_sent)
            self.request_bytes_received.labels(**labels).inc(bytes_received)
            self.request_latency.labels(**labels).observe(latency)

    def record_connection_lifetime(self, host: str, port: int, lifetime: float):
        """记录连接生命周期"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.connection_lifetime.labels(**labels).observe(lifetime)

    def record_connection_error(self, host: str, port: int, error_type: str):
        """记录连接错误"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.connection_errors.labels(error_type=error_type, **labels).inc()

    def record_balancer_decision(self, host: str, port: int, strategy: str):
        """记录负载均衡决策"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.balancer_decisions.labels(strategy=strategy, **labels).inc()

    def record_scaling_operation(self, host: str, port: int, operation: str):
        """记录扩缩容操作"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.pool_scaling_operations.labels(operation=operation, **labels).inc()

    def record_health_check(self, host: str, port: int, result: str):
        """记录健康检查结果"""
        labels = {'host': host, 'port': str(port)}
        
        with self._lock:
            self.health_check_operations.labels(result=result, **labels).inc()

    def set_system_info(self, version: str, start_time: float):
        """设置系统信息"""
        self.ipc_info.info({
            'version': version,
            'start_time': str(start_time)
        }) 