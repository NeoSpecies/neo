import socket
import time
from typing import Optional, Tuple
from dataclasses import dataclass
from enum import Enum

class ConnectionState(Enum):
    """连接状态枚举"""
    IDLE = "idle"           # 空闲状态
    BUSY = "busy"           # 正在使用
    CLOSED = "closed"       # 已关闭
    ERROR = "error"         # 错误状态

@dataclass
class ConnectionStats:
    """连接统计信息"""
    created_at: float = 0.0          # 创建时间
    last_used_at: float = 0.0        # 最后使用时间
    total_requests: int = 0          # 总请求数
    total_errors: int = 0            # 总错误数
    total_bytes_sent: int = 0        # 总发送字节数
    total_bytes_received: int = 0    # 总接收字节数
    avg_response_time: float = 0.0   # 平均响应时间
    total_response_time: float = 0.0 # 总响应时间

class Connection:
    """表示连接池中的一个连接"""
    def __init__(self, host: str, port: int, timeout: float = 5.0):
        self.host = host
        self.port = port
        self.timeout = timeout
        self.socket: Optional[socket.socket] = None
        self.state = ConnectionState.CLOSED
        self.stats = ConnectionStats(created_at=time.time())
        self._last_error: Optional[Exception] = None
        self._current_request_start: float = 0.0

    def connect(self) -> bool:
        """建立连接"""
        try:
            if self.socket is not None:
                self.close()
            
            self.socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
            self.socket.settimeout(self.timeout)
            self.socket.connect((self.host, self.port))
            self.state = ConnectionState.IDLE
            return True
        except Exception as e:
            self._last_error = e
            self.state = ConnectionState.ERROR
            self.stats.total_errors += 1
            return False

    def close(self):
        """关闭连接"""
        if self.socket is not None:
            try:
                self.socket.close()
            except:
                pass
            finally:
                self.socket = None
        self.state = ConnectionState.CLOSED

    def is_connected(self) -> bool:
        """检查连接是否有效"""
        if self.socket is None or self.state in (ConnectionState.CLOSED, ConnectionState.ERROR):
            return False
        # 可以添加更多的连接检查逻辑
        return True

    def send(self, data: bytes) -> bool:
        """发送数据"""
        if not self.is_connected():
            return False
        
        try:
            self._current_request_start = time.time()
            self.state = ConnectionState.BUSY
            self.stats.last_used_at = self._current_request_start
            self.socket.sendall(data)
            self.stats.total_bytes_sent += len(data)
            return True
        except Exception as e:
            self._last_error = e
            self.state = ConnectionState.ERROR
            self.stats.total_errors += 1
            return False

    def receive(self, buffer_size: int = 4096) -> Tuple[Optional[bytes], bool]:
        """接收数据"""
        if not self.is_connected():
            return None, False

        try:
            data = self.socket.recv(buffer_size)
            if data:
                self.stats.total_bytes_received += len(data)
                return data, True
            return None, False
        except Exception as e:
            self._last_error = e
            self.state = ConnectionState.ERROR
            self.stats.total_errors += 1
            return None, False
        finally:
            if self._current_request_start > 0:
                response_time = time.time() - self._current_request_start
                self.stats.total_response_time += response_time
                self.stats.total_requests += 1
                self.stats.avg_response_time = (
                    self.stats.total_response_time / self.stats.total_requests
                )
            self.state = ConnectionState.IDLE

    def get_stats(self) -> ConnectionStats:
        """获取连接统计信息"""
        return self.stats

    def get_last_error(self) -> Optional[Exception]:
        """获取最后一次错误"""
        return self._last_error

    def reset_stats(self):
        """重置统计信息"""
        self.stats = ConnectionStats(created_at=self.stats.created_at)
        self._last_error = None 