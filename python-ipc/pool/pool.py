import threading
import time
import logging
from typing import List, Optional, Dict, Any
from concurrent.futures import ThreadPoolExecutor
from .connection import Connection, ConnectionState, ConnectionStats
from .balancer import LoadBalancer, create_balancer
from ..metrics.prometheus import PrometheusMetrics

logger = logging.getLogger(__name__)

class ConnectionPool:
    """连接池实现"""
    def __init__(self, 
                 host: str,
                 port: int,
                 min_size: int = 5,
                 max_size: int = 20,
                 connection_timeout: float = 5.0,
                 idle_timeout: float = 60.0,
                 max_lifetime: float = 3600.0,
                 health_check_interval: float = 30.0,
                 balancer_strategy: str = 'weighted_response_time'):
        
        self.host = host
        self.port = port
        self.min_size = min_size
        self.max_size = max_size
        self.connection_timeout = connection_timeout
        self.idle_timeout = idle_timeout
        self.max_lifetime = max_lifetime
        self.health_check_interval = health_check_interval
        
        self._connections: List[Connection] = []
        self._lock = threading.RLock()
        self._balancer = create_balancer(balancer_strategy)
        self._stopped = False
        self._metrics = PrometheusMetrics()
        self._start_time = time.time()
        
        # 设置系统信息
        self._metrics.set_system_info(
            version="1.0.0",  # 可以从配置或环境变量获取
            start_time=self._start_time
        )
        
        # 初始化连接池
        self._initialize_pool()
        
        # 启动管理线程
        self._start_management_threads()

    def _initialize_pool(self):
        """初始化连接池"""
        with self._lock:
            for _ in range(self.min_size):
                self._create_connection()
            # 更新初始指标
            self._update_metrics()

    def _create_connection(self) -> Optional[Connection]:
        """创建新连接"""
        conn = Connection(self.host, self.port, self.connection_timeout)
        if conn.connect():
            self._connections.append(conn)
            self._metrics.record_scaling_operation(
                self.host, self.port, "create"
            )
            return conn
        else:
            self._metrics.record_connection_error(
                self.host, self.port, "connection_failed"
            )
            return None

    def _remove_connection(self, conn: Connection):
        """移除连接"""
        with self._lock:
            if conn in self._connections:
                # 记录连接生命周期
                lifetime = time.time() - conn.stats.created_at
                self._metrics.record_connection_lifetime(
                    self.host, self.port, lifetime
                )
                
                conn.close()
                self._connections.remove(conn)
                self._metrics.record_scaling_operation(
                    self.host, self.port, "remove"
                )

    def _start_management_threads(self):
        """启动管理线程"""
        self._executor = ThreadPoolExecutor(max_workers=2)
        self._executor.submit(self._auto_scale_worker)
        self._executor.submit(self._health_check_worker)

    def _auto_scale_worker(self):
        """自动扩缩容工作线程"""
        while not self._stopped:
            try:
                self._auto_scale()
            except Exception as e:
                logger.error(f"自动扩缩容错误: {e}")
            time.sleep(5)  # 每5秒检查一次

    def _health_check_worker(self):
        """健康检查工作线程"""
        while not self._stopped:
            try:
                self._check_connections_health()
            except Exception as e:
                logger.error(f"健康检查错误: {e}")
            time.sleep(self.health_check_interval)

    def _auto_scale(self):
        """执行自动扩缩容（优化后：使用配置参数）"""
        with self._lock:
            total_conns = len(self._connections)
            active_conns = len([c for c in self._connections if c.state == ConnectionState.BUSY])
            idle_conns = total_conns - active_conns

            # 从配置获取动态阈值（需确保Config类包含对应字段）
            usage_ratio = active_conns / total_conns if total_conns > 0 else 1
            scale_up_threshold = self.config.scale_up_threshold  # 例如默认0.7
            scale_step = self.config.scale_step                  # 例如默认2
            scale_down_idle_threshold = self.config.scale_down_idle_threshold  # 例如默认2

            # 扩容逻辑：使用配置阈值
            if usage_ratio > scale_up_threshold and total_conns < self.max_size:
                new_conns = min(
                    scale_step,
                    self.max_size - total_conns
                )
                for _ in range(new_conns):
                    self._create_connection()
                logger.info(f"扩容 {new_conns} 个连接（阈值：{scale_up_threshold}）")
                self._metrics.record_scaling_operation(
                    self.host, self.port, "scale_up"
                )

            # 缩容逻辑：使用配置阈值
            elif idle_conns > scale_down_idle_threshold and total_conns > self.min_size:
                remove_count = min(
                    idle_conns - 1,  # 保留至少1个空闲连接
                    total_conns - self.min_size
                )
                for conn in list(self._connections):
                    if remove_count <= 0:
                        break
                    if conn.state == ConnectionState.IDLE:
                        self._remove_connection(conn)
                        remove_count -= 1
                logger.info(f"缩容 {remove_count} 个连接（阈值：{scale_down_idle_threshold}）")
                self._metrics.record_scaling_operation(
                    self.host, self.port, "scale_down"
                )

            self._update_metrics()

    def _check_connections_health(self):
        """检查连接健康状态"""
        current_time = time.time()
        with self._lock:
            for conn in list(self._connections):
                health_result = "healthy"
                
                # 检查连接是否超时
                if conn.state == ConnectionState.IDLE:
                    idle_time = current_time - conn.stats.last_used_at
                    if idle_time > self.idle_timeout:
                        logger.info(f"移除空闲连接: {idle_time:.1f}秒未使用")
                        self._remove_connection(conn)
                        health_result = "idle_timeout"
                        continue

                # 检查连接生命周期
                conn_lifetime = current_time - conn.stats.created_at
                if conn_lifetime > self.max_lifetime:
                    logger.info(f"移除过期连接: 已运行{conn_lifetime:.1f}秒")
                    self._remove_connection(conn)
                    health_result = "max_lifetime"
                    continue

                # 检查错误状态连接
                if conn.state == ConnectionState.ERROR:
                    logger.info(f"移除错误连接: {conn.get_last_error()}")
                    self._remove_connection(conn)
                    health_result = "error"
                    continue
                
                self._metrics.record_health_check(
                    self.host, self.port, health_result
                )

            self._update_metrics()

    def get_connection(self) -> Optional[Connection]:
        """获取一个可用连接"""
        with self._lock:
            # 使用负载均衡器选择连接
            conn = self._balancer.select(self._connections)
            if conn:
                self._metrics.record_balancer_decision(
                    self.host, self.port, 
                    self._balancer.__class__.__name__
                )
                return conn

            # 如果没有可用连接且未达到最大值，创建新连接
            if len(self._connections) < self.max_size:
                conn = self._create_connection()
                if conn:
                    return conn

            self._metrics.record_connection_error(
                self.host, self.port, "no_available_connection"
            )
            return None

    def return_connection(self, conn: Connection):
        """归还连接到连接池"""
        if conn.state == ConnectionState.ERROR:
            self._metrics.record_connection_error(
                self.host, self.port, "connection_error"
            )
            self._remove_connection(conn)
        else:
            conn.state = ConnectionState.IDLE
            # 记录请求指标
            stats = conn.get_stats()
            self._metrics.record_request(
                self.host, self.port,
                "success" if conn.state != ConnectionState.ERROR else "error",
                stats.total_bytes_sent,
                stats.total_bytes_received,
                stats.avg_response_time
            )
        self._update_metrics()

    def _update_metrics(self):
        """更新连接池指标"""
        stats = self.get_stats()
        self._metrics.update_pool_metrics(self.host, self.port, stats)

    def get_stats(self) -> Dict[str, Any]:
        """获取连接池统计信息"""
        with self._lock:
            total_conns = len(self._connections)
            active_conns = len([c for c in self._connections if c.state == ConnectionState.BUSY])
            idle_conns = len([c for c in self._connections if c.state == ConnectionState.IDLE])
            error_conns = len([c for c in self._connections if c.state == ConnectionState.ERROR])

            # 汇总所有连接的统计信息
            total_requests = 0
            total_errors = 0
            total_bytes_sent = 0
            total_bytes_received = 0
            total_response_time = 0
            
            for conn in self._connections:
                stats = conn.get_stats()
                total_requests += stats.total_requests
                total_errors += stats.total_errors
                total_bytes_sent += stats.total_bytes_sent
                total_bytes_received += stats.total_bytes_received
                total_response_time += stats.total_response_time

            return {
                'total_connections': total_conns,
                'active_connections': active_conns,
                'idle_connections': idle_conns,
                'error_connections': error_conns,
                'total_requests': total_requests,
                'total_errors': total_errors,
                'total_bytes_sent': total_bytes_sent,
                'total_bytes_received': total_bytes_received,
                'average_response_time': (
                    total_response_time / total_requests if total_requests > 0 else 0
                ),
                'connection_usage_ratio': (
                    active_conns / total_conns if total_conns > 0 else 0
                )
            }

    def close(self):
        """关闭连接池"""
        self._stopped = True
        with self._lock:
            for conn in self._connections:
                conn.close()
            self._connections.clear()
        self._executor.shutdown(wait=True)


class ConnectionPool:
    def __init__(self, min_size, max_size):
        self.min_size = min_size
        self.max_size = max_size
        self.idle_conns = []  # 空闲连接列表
        self.active_conns = 0  # 活跃连接数
        self.waiting_queue = 0  # 等待队列长度
        self.avg_rtt = 100  # 平均响应时间（ms，滑动平均）
        self.max_util = 0.0  # 最大连接利用率

    def start_auto_manager(self, interval=30):
        """启动自动管理循环（需在初始化时调用）"""
        def manager_loop():
            while True:
                time.sleep(interval)
                self._maybe_resize()
    
        threading.Thread(target=manager_loop, daemon=True).start()
    
    def _maybe_resize(self):
        """动态扩缩容核心逻辑"""
        total_conns = len(self.idle_conns) + self.active_conns
    
        # 扩容逻辑（连接池未满且负载高）
        if total_conns < self.max_size and self.waiting_queue > 0 and self.max_util > 0.8:
            need_add = min(int(total_conns * 0.2), self.max_size - total_conns)
            for _ in range(need_add):
                try:
                    conn = Connection()  # 假设Connection是连接对象
                    self.idle_conns.append(conn)
                except ConnectionError:
                    continue
    
        # 缩容逻辑（连接池非最小且负载低）
        if total_conns > self.min_size and self.active_conns < self.min_size and self.avg_rtt < 50:
            need_remove = min(int(total_conns * 0.1), total_conns - self.min_size)
            for _ in range(need_remove):
                if self.idle_conns:
                    conn = self.idle_conns.pop(0)
                    conn.close()

    def health_check(self):
        """执行健康检查（需定时调用）"""
        to_remove = []
        for idx, conn in enumerate(self.idle_conns):
            # 发送心跳包（使用协议定义的HEARTBEAT类型）
            heartbeat = protocol.Message(
                header=protocol.MessageHeader(
                    msg_type=protocol.MessageType.HEARTBEAT,
                    timestamp=int(time.time() * 1000)
                ),
                payload=b"ping"
            )
            send_time = time.time()
    
            # 发送并等待响应（超时500ms）
            try:
                conn.send(heartbeat)
                resp = conn.receive(timeout=0.5)
                if resp.header.msg_type != protocol.MessageType.HEARTBEAT:
                    raise ProtocolError("心跳响应类型错误")
            except (ConnectionError, ProtocolError):
                to_remove.append(idx)
                continue
    
            # 计算响应时间（滑动平均）
            rtt = (time.time() - send_time) * 1000  # 转换为毫秒
            self.avg_rtt = (self.avg_rtt * 9 + rtt) / 10
    
        # 移除异常连接（从后往前删除避免索引错位）
        for idx in reversed(to_remove):
            conn = self.idle_conns.pop(idx)
            conn.close()