from abc import ABC, abstractmethod
from typing import List, Dict, Any
import random
import time
from .connection import Connection, ConnectionState

class LoadBalancer(ABC):
    """负载均衡器基类"""
    @abstractmethod
    def select(self, connections: List[Connection]) -> Connection:
        """选择一个连接"""
        pass

    def filter_available(self, connections: List[Connection]) -> List[Connection]:
        """过滤出可用的连接"""
        return [conn for conn in connections 
                if conn.state == ConnectionState.IDLE and conn.is_connected()]

class RandomBalancer(LoadBalancer):
    """随机负载均衡"""
    def select(self, connections: List[Connection]) -> Connection:
        available = self.filter_available(connections)
        if not available:
            return None
        return random.choice(available)

class RoundRobinBalancer(LoadBalancer):
    """轮询负载均衡"""
    def __init__(self):
        self._current_index = 0

    def select(self, connections: List[Connection]) -> Connection:
        available = self.filter_available(connections)
        if not available:
            return None
        
        self._current_index = (self._current_index + 1) % len(available)
        return available[self._current_index]

class LeastConnectionsBalancer(LoadBalancer):
    """最少连接数负载均衡"""
    def select(self, connections: List[Connection]) -> Connection:
        available = self.filter_available(connections)
        if not available:
            return None
        
        return min(available, 
                  key=lambda c: c.stats.total_requests)

class ResponseTimeBalancer(LoadBalancer):
    """响应时间负载均衡"""
    def select(self, connections: List[Connection]) -> Connection:
        available = self.filter_available(connections)
        if not available:
            return None
        
        return min(available, 
                  key=lambda c: c.stats.avg_response_time)

class WeightedResponseTimeBalancer(LoadBalancer):
    """加权响应时间负载均衡"""
    def __init__(self, recent_weight: float = 0.7):
        self.recent_weight = recent_weight
        self._last_response_times: Dict[Connection, float] = {}

    def select(self, connections: List[Connection]) -> Connection:
        available = self.filter_available(connections)
        if not available:
            return None

        current_time = time.time()
        
        # 计算加权分数
        scores = {}
        for conn in available:
            last_time = self._last_response_times.get(conn, 0)
            time_since_last_use = current_time - conn.stats.last_used_at
            
            # 结合历史响应时间和最近响应时间
            recent_score = conn.stats.avg_response_time
            historical_score = last_time
            
            score = (self.recent_weight * recent_score + 
                    (1 - self.recent_weight) * historical_score)
            
            # 考虑空闲时间，避免连接饥饿
            score *= (1 + time_since_last_use * 0.1)
            
            scores[conn] = score

        # 选择得分最低的连接
        selected = min(scores.items(), key=lambda x: x[1])[0]
        self._last_response_times[selected] = current_time
        return selected

def create_balancer(strategy: str) -> LoadBalancer:
    """创建负载均衡器工厂方法"""
    balancers = {
        'random': RandomBalancer(),
        'round_robin': RoundRobinBalancer(),
        'least_connections': LeastConnectionsBalancer(),
        'response_time': ResponseTimeBalancer(),
        'weighted_response_time': WeightedResponseTimeBalancer(),
    }
    return balancers.get(strategy, RandomBalancer()) 