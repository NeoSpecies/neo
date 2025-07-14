import asyncio
import json
import socket
import struct
import logging
from typing import Dict, Any, Optional, Callable
from dataclasses import dataclass
from enum import IntEnum

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class MessageType(IntEnum):
    REQUEST = 1
    RESPONSE = 2
    REGISTER = 3
    HEARTBEAT = 4


@dataclass
class Message:
    msg_type: MessageType
    id: str
    service: str
    method: str
    data: bytes
    metadata: Dict[str, str]


class NeoIPCClient:
    def __init__(self, host: str = "localhost", port: int = 35999):
        self.host = host
        self.port = port
        self.reader: Optional[asyncio.StreamReader] = None
        self.writer: Optional[asyncio.StreamWriter] = None
        self.handlers: Dict[str, Callable] = {}
        self.pending_requests: Dict[str, asyncio.Future] = {}
        self.service_name: Optional[str] = None
        
    async def connect(self):
        """建立IPC连接"""
        self.reader, self.writer = await asyncio.open_connection(self.host, self.port)
        logger.info(f"Connected to Neo IPC server at {self.host}:{self.port}")
        
    async def register_service(self, service_name: str, metadata: Dict[str, str] = None):
        """注册服务到Neo框架"""
        self.service_name = service_name
        metadata = metadata or {}
        
        msg = Message(
            msg_type=MessageType.REGISTER,
            id="",
            service=service_name,
            method="",
            data=json.dumps({
                "name": service_name,
                "metadata": metadata
            }).encode(),
            metadata={}
        )
        
        await self._send_message(msg)
        logger.info(f"Service '{service_name}' registered")
        
    def register_handler(self, method: str, handler: Callable):
        """注册方法处理器"""
        self.handlers[method] = handler
        logger.info(f"Handler registered for method: {method}")
        
    async def _send_message(self, msg: Message):
        """发送消息到Neo框架"""
        # 序列化消息
        metadata_json = json.dumps(msg.metadata).encode()
        
        # 构建消息格式: [type:1][id_len:4][id][service_len:4][service][method_len:4][method][metadata_len:4][metadata][data_len:4][data]
        # 使用小端字节序与Go端保持一致
        msg_bytes = struct.pack('<B', msg.msg_type)
        
        # ID
        id_bytes = msg.id.encode()
        msg_bytes += struct.pack('<I', len(id_bytes)) + id_bytes
        
        # Service
        service_bytes = msg.service.encode()
        msg_bytes += struct.pack('<I', len(service_bytes)) + service_bytes
        
        # Method
        method_bytes = msg.method.encode()
        msg_bytes += struct.pack('<I', len(method_bytes)) + method_bytes
        
        # Metadata
        msg_bytes += struct.pack('<I', len(metadata_json)) + metadata_json
        
        # Data
        msg_bytes += struct.pack('<I', len(msg.data)) + msg.data
        
        # 发送总长度和消息
        self.writer.write(struct.pack('<I', len(msg_bytes)))
        self.writer.write(msg_bytes)
        await self.writer.drain()
        
    async def _read_message(self) -> Optional[Message]:
        """从Neo框架读取消息"""
        # 读取消息长度
        len_bytes = await self.reader.readexactly(4)
        msg_len = struct.unpack('<I', len_bytes)[0]
        
        # 读取消息内容
        msg_bytes = await self.reader.readexactly(msg_len)
        offset = 0
        
        # 解析消息类型
        msg_type = MessageType(msg_bytes[offset])
        offset += 1
        
        # 解析ID
        id_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        msg_id = msg_bytes[offset:offset+id_len].decode()
        offset += id_len
        
        # 解析Service
        service_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        service = msg_bytes[offset:offset+service_len].decode()
        offset += service_len
        
        # 解析Method
        method_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        method = msg_bytes[offset:offset+method_len].decode()
        offset += method_len
        
        # 解析Metadata
        metadata_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        metadata_json = msg_bytes[offset:offset+metadata_len].decode()
        metadata = json.loads(metadata_json) if metadata_json else {}
        offset += metadata_len
        
        # 解析Data
        data_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        data = msg_bytes[offset:offset+data_len]
        
        return Message(msg_type, msg_id, service, method, data, metadata)
        
    async def _handle_request(self, msg: Message):
        """处理收到的请求"""
        if msg.method not in self.handlers:
            error_resp = Message(
                msg_type=MessageType.RESPONSE,
                id=msg.id,
                service=msg.service,
                method=msg.method,
                data=json.dumps({
                    "error": f"Method '{msg.method}' not found"
                }).encode(),
                metadata={"error": "true"}
            )
            await self._send_message(error_resp)
            return
            
        try:
            # 解析请求数据
            request_data = json.loads(msg.data.decode()) if msg.data else {}
            
            # 调用处理器
            handler = self.handlers[msg.method]
            if asyncio.iscoroutinefunction(handler):
                result = await handler(request_data)
            else:
                result = handler(request_data)
                
            # 发送响应
            response = Message(
                msg_type=MessageType.RESPONSE,
                id=msg.id,
                service=msg.service,
                method=msg.method,
                data=json.dumps(result).encode(),
                metadata={}
            )
            await self._send_message(response)
            
        except Exception as e:
            logger.error(f"Error handling request: {e}")
            error_resp = Message(
                msg_type=MessageType.RESPONSE,
                id=msg.id,
                service=msg.service,
                method=msg.method,
                data=json.dumps({
                    "error": str(e)
                }).encode(),
                metadata={"error": "true"}
            )
            await self._send_message(error_resp)
            
    async def listen(self):
        """监听并处理消息"""
        logger.info("Starting to listen for messages...")
        
        while True:
            try:
                msg = await self._read_message()
                if msg is None:
                    break
                    
                if msg.msg_type == MessageType.REQUEST:
                    # 异步处理请求
                    asyncio.create_task(self._handle_request(msg))
                elif msg.msg_type == MessageType.RESPONSE:
                    # 处理响应
                    if msg.id in self.pending_requests:
                        self.pending_requests[msg.id].set_result(msg)
                        
            except Exception as e:
                logger.error(f"Error in message loop: {e}")
                break
                
    async def close(self):
        """关闭连接"""
        if self.writer:
            self.writer.close()
            await self.writer.wait_closed()
        logger.info("Connection closed")