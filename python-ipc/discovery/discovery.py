import asyncio
import logging
from typing import Dict, List, Optional, Any
from .protocol import pack_message, unpack_response, IPCProtocolError

logger = logging.getLogger(__name__)

class ServiceInfo:
    """服务信息数据类"""
    def __init__(self, **kwargs):
        self.id: str = kwargs.get('id', '')
        self.name: str = kwargs.get('name', '')
        self.address: str = kwargs.get('address', '')
        self.port: int = kwargs.get('port', 0)
        self.metadata: Dict[str, str] = kwargs.get('metadata', {})
        self.status: str = kwargs.get('status', 'unknown')
        self.expire_at: str = kwargs.get('expire_at', '')
        self.updated_at: str = kwargs.get('updated_at', '')

    def to_dict(self) -> Dict[str, Any]:
        result = {
            'id': self.id,
            'name': self.name,
            'address': self.address,
            'port': self.port,
            'expire_at': self.expire_at,
            'updated_at': self.updated_at,
            'metadata': self.metadata,
            'status': self.status  # 新增：添加status字段
        }
        # 新增：验证关键字段
        required_fields = ['id', 'name', 'address', 'port', 'expire_at', 'status']  # 补充status到必填项检查
        for field in required_fields:
            if field not in result or result[field] is None:
                logger.warning(f"服务信息缺少必要字段: {field}")
        return result

class ServiceDiscovery:
    """基于IPC的服务发现客户端"""
    def __init__(self, ipc_host: str = '127.0.0.1', ipc_port: int = 9090):
        self.ipc_host = ipc_host
        self.ipc_port = ipc_port
        self._reader: Optional[asyncio.StreamReader] = None
        self._writer: Optional[asyncio.StreamWriter] = None

    async def _connect(self) -> None:
        """建立IPC连接"""
        try:
            self._reader, self._writer = await asyncio.open_connection(
                self.ipc_host, self.ipc_port
            )
            logger.info(f"已连接到IPC服务: {self.ipc_host}:{self.ipc_port}")
        except Exception as e:
            raise IPCProtocolError(f"IPC连接失败: {str(e)}")

    async def _send_request(self, method: str, params: Dict[str, Any]) -> Dict[str, Any]:
        """发送IPC请求并处理响应"""
        try:
            if not self._writer or self._reader.at_eof():
                await self._connect()
            
            # 打包并发送请求
            packed_data = await pack_message(method, params)
            self._writer.write(packed_data)
            await self._writer.drain()
            
            # 读取响应
            response_data = await self._reader.read(4096)
            if not response_data:
                raise IPCProtocolError("未收到响应数据")
            
            return await unpack_response(response_data)
        except ConnectionRefusedError:
            raise IPCProtocolError(f"无法连接到服务端 {self.ipc_host}:{self.ipc_port}")
        except asyncio.TimeoutError:
            raise IPCProtocolError("请求超时")

    async def register_service(self, service: ServiceInfo) -> bool:
        """注册服务到IPC服务发现"""
        try:
            logger.info(f"开始注册服务: {service.name} (ID: {service.id})")
            response = await self._send_request('register', {
                'action': 'register',
                'service': service.to_dict(),
                'name': service.name,
                'id': service.id
            })
            
            # 新增：记录完整响应日志
            logger.debug(f"服务注册响应: {response}")
            
            if response.get('error'):
                logger.error(f"服务端返回错误: {response['error']}")
                return False
            logger.info(f"服务注册成功: {service.name}")
            return True
        except IPCProtocolError as e:
            logger.error(f"服务注册失败: {str(e)} (服务名: {service.name}, ID: {service.id})")
            return False
        # 新增：捕获其他可能的异常
        except Exception as e:
            logger.exception(f"注册服务时发生未预期错误: {str(e)}")
            return False

    async def deregister_service(self, service_id: str) -> bool:
        """从IPC服务发现注销服务"""
        try:
            response = await self._send_request('deregister', {
                'id': service_id
            })
            return response.get('error') is None
        except IPCProtocolError as e:
            logger.error(f"服务注销失败: {str(e)}")
            return False

    async def discover_service(self, service_name: str) -> List[ServiceInfo]:
        """发现指定名称的服务"""
        try:
            response = await self._send_request('discover', {
                'name': service_name
            })
            if 'result' in response and isinstance(response['result'], list):
                return [ServiceInfo(**item) for item in response['result']]
            return []
        except IPCProtocolError as e:
            logger.error(f"服务发现失败: {str(e)}")
            return []

    def close(self) -> None:
        """关闭IPC连接"""
        if self._writer:
            self._writer.close()
            self._writer = None
        self._reader = None