import struct
import time
import uuid
import zlib
from typing import Optional, Tuple, Dict, Any
from dataclasses import dataclass
from enum import IntEnum
import logging
from .compression import CompressionManager, CompressionError

logger = logging.getLogger(__name__)

class MessageType(IntEnum):
    """消息类型"""
    REQUEST = 1    # 请求消息
    RESPONSE = 2   # 响应消息
    ERROR = 3      # 错误消息
    HEARTBEAT = 4  # 心跳消息

class Priority(IntEnum):
    """消息优先级"""
    LOW = 0
    NORMAL = 1
    HIGH = 2
    URGENT = 3

class ProtocolError(Exception):
    """协议错误"""
    pass

@dataclass
class MessageHeader:
    """消息头"""
    version: int = 1                      # 协议版本
    msg_type: MessageType = MessageType.REQUEST  # 消息类型
    compression: str = 'none'             # 压缩算法
    priority: Priority = Priority.NORMAL   # 优先级
    trace_id: str = ''                    # 追踪ID
    timestamp: float = 0.0                # 时间戳
    payload_size: int = 0                 # 负载大小
    checksum: int = 0                     # 校验和
    
    # 消息头格式：
    # | 版本(1B) | 类型(1B) | 压缩(1B) | 优先级(1B) | 追踪ID(36B) |
    # | 时间戳(8B) | 负载大小(4B) | 校验和(4B) |
    HEADER_FORMAT = '!BBBB36sLIL'
    HEADER_SIZE = struct.calcsize(HEADER_FORMAT)
    
    def pack(self) -> bytes:
        """打包消息头"""
        return struct.pack(
            self.HEADER_FORMAT,
            self.version,
            self.msg_type,
            ord(self.compression[0]),  # 使用算法名称的首字母
            self.priority,
            self.trace_id.encode(),
            int(self.timestamp),
            self.payload_size,
            self.checksum
        )
    
    @classmethod
    def unpack(cls, data: bytes) -> 'MessageHeader':
        """解包消息头"""
        if len(data) < cls.HEADER_SIZE:
            raise ProtocolError("Invalid header size")
        
        try:
            (version, msg_type, compression, priority,
             trace_id, timestamp, payload_size, checksum) = struct.unpack(
                cls.HEADER_FORMAT, data[:cls.HEADER_SIZE]
            )
            
            # 解析压缩算法
            comp_map = {'n': 'none', 'g': 'gzip', 'z': 'zstd', 'l': 'lz4'}
            comp_type = comp_map.get(chr(compression).lower(), 'none')
            
            return cls(
                version=version,
                msg_type=MessageType(msg_type),
                compression=comp_type,
                priority=Priority(priority),
                trace_id=trace_id.decode().strip('\x00'),
                timestamp=timestamp,
                payload_size=payload_size,
                checksum=checksum
            )
        except Exception as e:
            raise ProtocolError(f"Failed to unpack header: {e}")

@dataclass
class Message:
    """IPC消息"""
    header: MessageHeader
    payload: bytes
    
    @property
    def trace_id(self) -> str:
        return self.header.trace_id
    
    @property
    def msg_type(self) -> MessageType:
        return self.header.msg_type
    
    @property
    def priority(self) -> Priority:
        return self.header.priority
# 定义协议头类（与Go结构体字段顺序/类型严格一致）
class ProtocolHeader:
    def __init__(self):
        self.magic = 0x12345678  # 固定魔数（与Go一致）
        self.version = 1
        self.msg_id_len = 0
        self.method_name_len = 0
        self.param_len = 0
        self.file_count = 0
        self.compression_alg = 0  # 压缩算法标识
        self.trace_id_len = 0     # 追踪ID长度

    # 编码协议头（使用struct模块与Go对齐）
    def encode(self):
        # 格式字符串与Go的binary.BigEndian严格对应（I:uint32, B:uint8, H:uint16, I:uint32）
        fmt = "!IBH H I B B H"  # 注意字段顺序与Go结构体一致
        return struct.pack(fmt,
            self.magic,
            self.version,
            self.msg_id_len,
            self.method_name_len,
            self.param_len,
            self.file_count,
            self.compression_alg,
            self.trace_id_len
        )

    # 解码协议头（与Go反序列化逻辑一致）
    @classmethod
    def decode(cls, data):
        fmt = "!IBH H I B B H"
        unpacked = struct.unpack(fmt, data[:struct.calcsize(fmt)])
        header = cls()
        header.magic, header.version, header.msg_id_len, \
        header.method_name_len, header.param_len, header.file_count, \
        header.compression_alg, header.trace_id_len = unpacked
        return header
        
class Protocol:
    """IPC协议实现"""
    
    def __init__(self, compression: str = 'none'):
        """
        初始化协议
        
        Args:
            compression: 压缩算法，支持 'none', 'gzip', 'zstd', 'lz4'
        """
        self.compression = CompressionManager(compression)
        self._trace_ids = set()  # 用于检测重复消息
    
    def _generate_trace_id(self) -> str:
        """生成唯一的追踪ID"""
        while True:
            trace_id = str(uuid.uuid4())
            if trace_id not in self._trace_ids:
                self._trace_ids.add(trace_id)
                # 限制集合大小，防止无限增长
                if len(self._trace_ids) > 10000:
                    self._trace_ids.pop()
                return trace_id
    
    def _calculate_checksum(self, data: bytes) -> int:
        """计算CRC32校验和"""
        return zlib.crc32(data) & 0xFFFFFFFF
    
    def _verify_checksum(self, data: bytes, expected: int) -> bool:
        """验证校验和"""
        actual = self._calculate_checksum(data)
        return actual == expected
    
    def pack(self, payload: bytes, msg_type: MessageType = MessageType.REQUEST,
             priority: Priority = Priority.NORMAL) -> bytes:
        """
        打包消息
        
        Args:
            payload: 消息负载
            msg_type: 消息类型
            priority: 消息优先级
            
        Returns:
            打包后的消息数据
            
        Raises:
            ProtocolError: 打包失败
        """
        try:
            # 压缩负载
            compressed_payload = self.compression.compress(payload)
            
            # 创建消息头
            header = MessageHeader(
                msg_type=msg_type,
                compression=self.compression.get_algorithm(),
                priority=priority,
                trace_id=self._generate_trace_id(),
                timestamp=time.time(),
                payload_size=len(compressed_payload),
                checksum=self._calculate_checksum(compressed_payload)
            )
            
            # 打包消息
            packed = header.pack() + compressed_payload
            logger.debug(
                f"Packed message: type={msg_type.name}, "
                f"size={len(packed)}, trace_id={header.trace_id}"
            )
            return packed
            
        except Exception as e:
            raise ProtocolError(f"Failed to pack message: {e}")
    
    def unpack(self, data: bytes) -> Message:
        """
        解包消息
        
        Args:
            data: 要解包的数据
            
        Returns:
            解包后的消息对象
            
        Raises:
            ProtocolError: 解包失败
        """
        try:
            # 解析消息头
            header = MessageHeader.unpack(data)
            
            # 提取负载
            payload_data = data[MessageHeader.HEADER_SIZE:]
            if len(payload_data) != header.payload_size:
                raise ProtocolError(
                    f"Payload size mismatch: expected {header.payload_size}, "
                    f"got {len(payload_data)}"
                )
            
            # 验证校验和
            if not self._verify_checksum(payload_data, header.checksum):
                raise ProtocolError("Checksum verification failed")
            
            # 创建压缩管理器
            decompressor = CompressionManager(header.compression)
            
            # 解压负载
            payload = decompressor.decompress(payload_data)
            
            logger.debug(
                f"Unpacked message: type={header.msg_type.name}, "
                f"size={len(payload)}, trace_id={header.trace_id}"
            )
            
            return Message(header=header, payload=payload)
            
        except Exception as e:
            raise ProtocolError(f"Failed to unpack message: {e}")
    
    def create_response(self, request: Message, payload: bytes) -> bytes:
        """
        创建响应消息
        
        Args:
            request: 请求消息
            payload: 响应负载
            
        Returns:
            打包后的响应消息
        """
        return self.pack(
            payload,
            msg_type=MessageType.RESPONSE,
            priority=request.priority
        )
    
    def create_error(self, request: Message, error: str) -> bytes:
        """
        创建错误消息
        
        Args:
            request: 请求消息
            error: 错误信息
            
        Returns:
            打包后的错误消息
        """
        return self.pack(
            error.encode(),
            msg_type=MessageType.ERROR,
            priority=request.priority
        )
    
    def create_heartbeat(self) -> bytes:
        """
        创建心跳消息
        
        Returns:
            打包后的心跳消息
        """
        return self.pack(
            b'',
            msg_type=MessageType.HEARTBEAT,
            priority=Priority.LOW
        ) 