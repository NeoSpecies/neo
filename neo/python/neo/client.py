from collections import OrderedDict
from threading import Lock
import socket
import struct
import json
import uuid
import time

class IpcClient:
    def __init__(self, address):
        self.address = self._parse_address(address)
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.connect(self.address)
        self.async_bridge = AsyncBridge()

    def _parse_address(self, addr_str):
        # 解析地址字符串（示例格式："tcp://127.0.0.1:9090"）
        protocol, rest = addr_str.split('://')
        host, port = rest.split(':')
        return (host, int(port))

    def call_async(self, method: str, params: dict, callback: callable):
        msg_id = str(uuid.uuid4())
        try:
            # 构造请求协议头
            request = bytes()
            # 魔数（0xAEBD）
            request += struct.pack(">H", 0xAEBD)
            # 协议版本（0x01）
            request += bytes([0x01])
            # 回调标识位（0x01表示需要回调）
            request += bytes([0x01])
            # 回调ID长度（2字节大端序）
            request += struct.pack(">H", len(msg_id))
            # 回调ID内容
            request += msg_id.encode()
            
            # 注册回调（原有逻辑保留）
            self.async_bridge.register_callback(msg_id, callback)
            
            # 发送请求（需要补充完整协议构造）
            self.sock.sendall(request)
            
        except Exception as e:
            print(f"发送异步请求失败: {str(e)}")

class AsyncBridge:
    def __init__(self):
        self.pending_callbacks = OrderedDict()
        self.callback_lock = Lock()
        self.cleanup_interval = 300  # 5分钟清理一次过期回调
        
    def register_callback(self, msg_id: str, callback: callable):
        with self.callback_lock:
            self.pending_callbacks[msg_id] = {
                'callback': callback,
                'timestamp': time.time(),
                'retries': 0
            }
    
    def _cleanup_expired(self):
        current_time = time.time()
        with self.callback_lock:
            for msg_id in list(self.pending_callbacks.keys()):
                entry = self.pending_callbacks[msg_id]
                if current_time - entry['timestamp'] > 3600:  # 1小时过期
                    del self.pending_callbacks[msg_id]