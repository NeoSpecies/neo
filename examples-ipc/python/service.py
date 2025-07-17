#!/usr/bin/env python3
"""
Neo Framework IPC Python 示例服务
演示如何使用 Python 创建一个 IPC 服务
"""

import asyncio
import json
import struct
import logging
import time
import os
from datetime import datetime
from typing import Dict, Any, Optional
from dataclasses import dataclass
from enum import IntEnum

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
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
    """简化版 Neo IPC 客户端"""
    
    def __init__(self, host: str = "localhost", port: int = 9999):
        self.host = host
        self.port = port
        self.reader: Optional[asyncio.StreamReader] = None
        self.writer: Optional[asyncio.StreamWriter] = None
        self.handlers: Dict[str, callable] = {}
        self.service_name: Optional[str] = None
        
    async def connect(self):
        """连接到 IPC 服务器"""
        self.reader, self.writer = await asyncio.open_connection(self.host, self.port)
        logger.info(f"Connected to Neo IPC server at {self.host}:{self.port}")
        
    async def register_service(self, service_name: str, metadata: Dict[str, str] = None):
        """注册服务"""
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
            }).encode('utf-8'),
            metadata={}
        )
        
        await self._send_message(msg)
        logger.info(f"Service '{service_name}' registered")
        
    def handler(self, method: str):
        """装饰器：注册方法处理器"""
        def decorator(func):
            self.handlers[method] = func
            logger.info(f"Handler registered for method: {method}")
            return func
        return decorator
        
    async def _send_message(self, msg: Message):
        """发送消息到 Neo 框架"""
        # 序列化元数据
        metadata_json = json.dumps(msg.metadata).encode('utf-8')
        
        # 构建消息内容
        content = bytearray()
        
        # 消息类型
        content.extend(struct.pack('<B', msg.msg_type))
        
        # ID
        id_bytes = msg.id.encode('utf-8')
        content.extend(struct.pack('<I', len(id_bytes)))
        content.extend(id_bytes)
        
        # Service
        service_bytes = msg.service.encode('utf-8')
        content.extend(struct.pack('<I', len(service_bytes)))
        content.extend(service_bytes)
        
        # Method
        method_bytes = msg.method.encode('utf-8')
        content.extend(struct.pack('<I', len(method_bytes)))
        content.extend(method_bytes)
        
        # Metadata
        content.extend(struct.pack('<I', len(metadata_json)))
        content.extend(metadata_json)
        
        # Data
        content.extend(struct.pack('<I', len(msg.data)))
        content.extend(msg.data)
        
        # 发送总长度和消息
        self.writer.write(struct.pack('<I', len(content)))
        self.writer.write(content)
        await self.writer.drain()
        
    async def _read_message(self) -> Optional[Message]:
        """从 Neo 框架读取消息"""
        # 读取消息长度
        len_bytes = await self.reader.readexactly(4)
        msg_len = struct.unpack('<I', len_bytes)[0]
        
        # 读取消息内容
        msg_bytes = await self.reader.readexactly(msg_len)
        offset = 0
        
        # 解析消息类型
        msg_type = MessageType(msg_bytes[offset])
        offset += 1
        
        # 解析 ID
        id_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        msg_id = msg_bytes[offset:offset+id_len].decode('utf-8')
        offset += id_len
        
        # 解析 Service
        service_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        service = msg_bytes[offset:offset+service_len].decode('utf-8')
        offset += service_len
        
        # 解析 Method
        method_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        method = msg_bytes[offset:offset+method_len].decode('utf-8')
        offset += method_len
        
        # 解析 Metadata
        metadata_len = struct.unpack('<I', msg_bytes[offset:offset+4])[0]
        offset += 4
        metadata_json = msg_bytes[offset:offset+metadata_len].decode('utf-8')
        metadata = json.loads(metadata_json) if metadata_json else {}
        offset += metadata_len
        
        # 解析 Data
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
                }).encode('utf-8'),
                metadata={"error": "true"}
            )
            await self._send_message(error_resp)
            return
            
        try:
            # 解析请求数据
            request_data = json.loads(msg.data.decode('utf-8')) if msg.data else {}
            
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
                data=json.dumps(result).encode('utf-8'),
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
                }).encode('utf-8'),
                metadata={"error": "true"}
            )
            await self._send_message(error_resp)
            
    async def run(self):
        """运行服务"""
        # 启动心跳
        asyncio.create_task(self._heartbeat_loop())
        
        # 处理消息
        while True:
            try:
                msg = await self._read_message()
                if msg is None:
                    break
                    
                if msg.msg_type == MessageType.REQUEST:
                    await self._handle_request(msg)
                    
            except Exception as e:
                logger.error(f"Error in message loop: {e}")
                break
                
    async def _heartbeat_loop(self):
        """心跳循环"""
        while True:
            await asyncio.sleep(30)
            try:
                msg = Message(
                    msg_type=MessageType.HEARTBEAT,
                    id="",
                    service=self.service_name,
                    method="",
                    data=b"",
                    metadata={}
                )
                await self._send_message(msg)
                logger.debug("Heartbeat sent")
            except Exception as e:
                logger.error(f"Heartbeat error: {e}")
                break


async def main():
    """主函数"""
    # 从环境变量读取配置
    host = os.getenv('NEO_IPC_HOST', 'localhost')
    port = int(os.getenv('NEO_IPC_PORT', '9999'))  # 使用正确的默认端口
    
    # 创建客户端
    client = NeoIPCClient(host, port)
    
    # 注册处理器
    @client.handler("hello")
    async def hello(params):
        name = params.get("name", "World")
        return {
            "message": f"Hello, {name}!",
            "timestamp": datetime.now().isoformat(),
            "service": "Python Demo Service"
        }
    
    @client.handler("calculate")
    async def calculate(params):
        a = params.get("a", 0)
        b = params.get("b", 0)
        operation = params.get("operation", "add")
        
        operations = {
            "add": a + b,
            "subtract": a - b,
            "multiply": a * b,
            "divide": a / b if b != 0 else "Cannot divide by zero"
        }
        
        result = operations.get(operation, "Unknown operation")
        return {
            "result": result,
            "operation": operation,
            "a": a,
            "b": b
        }
    
    @client.handler("echo")
    async def echo(params):
        message = params.get("message", "")
        return {
            "echo": message,
            "length": len(message),
            "reversed": message[::-1]
        }
    
    @client.handler("getTime")
    async def get_time(params):
        format_str = params.get("format", "iso")
        now = datetime.now()
        
        formats = {
            "iso": now.isoformat(),
            "unix": int(now.timestamp()),
            "readable": now.strftime("%Y-%m-%d %H:%M:%S")
        }
        
        return {
            "time": formats.get(format_str, now.isoformat()),
            "timezone": time.tzname[0],
            "format": format_str
        }
    
    @client.handler("getInfo")
    async def get_info(params):
        return {
            "service": "demo-service-python",
            "language": "Python",
            "version": "1.0.0",
            "handlers": list(client.handlers.keys()),
            "uptime": "N/A",  # 简化示例，不计算运行时间
            "system": {
                "platform": os.name,
                "python_version": "3.x"
            }
        }
    
    try:
        # 连接并注册服务
        await client.connect()
        await client.register_service("demo-service-python", {
            "language": "python",
            "version": "1.0.0",
            "description": "Python demo service for Neo Framework"
        })
        
        logger.info("Python demo service is running...")
        logger.info(f"Listening on {host}:{port}")
        logger.info("Available methods: hello, calculate, echo, getTime, getInfo")
        
        # 运行服务
        await client.run()
        
    except KeyboardInterrupt:
        logger.info("Shutting down...")
    except Exception as e:
        logger.error(f"Service error: {e}")


if __name__ == "__main__":
    asyncio.run(main())