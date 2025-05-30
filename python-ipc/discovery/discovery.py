import etcd3
import asyncio
import logging
from typing import Dict, List, Optional, Any
from dataclasses import dataclass
import json
import time
import os
from concurrent.futures import ThreadPoolExecutor

logger = logging.getLogger(__name__)

# 从环境变量获取 etcd 配置
ETCD_PREFIX = os.getenv("ETCD_PREFIX", "/services")

@dataclass
class ServiceInfo:
    """服务信息"""
    name: str                # 服务名称
    id: str                  # 服务实例ID
    host: str               # 主机地址
    port: int               # 端口
    metadata: Dict[str, Any] # 元数据
    version: str = "1.0.0"   # 服务版本
    weight: int = 100        # 服务权重
    
    def to_dict(self) -> Dict[str, Any]:
        """转换为字典"""
        return {
            'name': self.name,
            'id': self.id,
            'host': self.host,
            'port': self.port,
            'metadata': self.metadata,
            'version': self.version,
            'weight': self.weight
        }
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'ServiceInfo':
        """从字典创建实例"""
        return cls(**data)

class ServiceDiscovery:
    """服务发现核心"""
    
    def __init__(self, 
                 host: str = 'localhost',
                 port: int = 2379,
                 timeout: float = 5.0,
                 executor: Optional[ThreadPoolExecutor] = None):
        """
        初始化服务发现
        
        Args:
            host: etcd 主机地址
            port: etcd 端口
            timeout: 操作超时时间
            executor: 线程池执行器
        """
        self.client = etcd3.client(host=host, port=port, timeout=timeout)
        self.executor = executor or ThreadPoolExecutor(max_workers=1)
        self._watchers = {}  # 服务监听器
        self._cache = {}     # 服务缓存
        self._stopped = False
    
    async def register_service(self, service: ServiceInfo, ttl: int = 10) -> bool:
        """
        注册服务
        
        Args:
            service: 服务信息
            ttl: 租约时间（秒）
            
        Returns:
            注册是否成功
        """
        try:
            # 创建租约
            lease = await self._run_in_executor(
                self.client.lease,
                ttl
            )
            
            # 注册服务，使用与 Go 服务相同的前缀
            key = f"{ETCD_PREFIX}/{service.name}/{service.id}"
            value = json.dumps(service.to_dict())
            
            success = await self._run_in_executor(
                self.client.put,
                key,
                value,
                lease=lease
            )
            
            if success:
                logger.info(
                    f"Registered service: {service.name} "
                    f"(id={service.id}, ttl={ttl}s)"
                )
                
                # 启动续约任务
                asyncio.create_task(self._keep_alive(lease, service))
                return True
            
            return False
            
        except Exception as e:
            logger.error(f"Failed to register service: {e}")
            return False
    
    async def deregister_service(self, service: ServiceInfo) -> bool:
        """
        注销服务
        
        Args:
            service: 服务信息
            
        Returns:
            注销是否成功
        """
        try:
            key = f"{ETCD_PREFIX}/{service.name}/{service.id}"
            success = await self._run_in_executor(
                self.client.delete,
                key
            )
            
            if success:
                logger.info(
                    f"Deregistered service: {service.name} "
                    f"(id={service.id})"
                )
            return success
            
        except Exception as e:
            logger.error(f"Failed to deregister service: {e}")
            return False
    
    async def discover_service(self, service_name: str) -> List[ServiceInfo]:
        """
        发现服务
        
        Args:
            service_name: 服务名称
            
        Returns:
            服务实例列表
        """
        try:
            # 先查询缓存
            if service_name in self._cache:
                return self._cache[service_name]
            
            # 从 etcd 查询，使用与 Go 服务相同的前缀
            prefix = f"{ETCD_PREFIX}/{service_name}/"
            response = await self._run_in_executor(
                self.client.get_prefix,
                prefix
            )
            
            services = []
            for value, _ in response:
                if value:
                    data = json.loads(value.decode())
                    services.append(ServiceInfo.from_dict(data))
            
            # 更新缓存
            self._cache[service_name] = services
            
            # 启动监听
            if service_name not in self._watchers:
                asyncio.create_task(
                    self._watch_service(service_name)
                )
            
            return services
            
        except Exception as e:
            logger.error(f"Failed to discover service: {e}")
            return []
    
    async def _keep_alive(self, lease: Any, service: ServiceInfo):
        """
        保持租约活跃
        
        Args:
            lease: 租约对象
            service: 服务信息
        """
        while not self._stopped:
            try:
                # 续约
                await self._run_in_executor(
                    self.client.refresh_lease,
                    lease
                )
                await asyncio.sleep(1)  # 每秒续约一次
                
            except Exception as e:
                logger.error(f"Failed to refresh lease: {e}")
                # 重新注册服务
                await self.register_service(service)
                break
    
    async def _watch_service(self, service_name: str):
        """
        监听服务变化
        
        Args:
            service_name: 服务名称
        """
        prefix = f"{ETCD_PREFIX}/{service_name}/"
        
        while not self._stopped:
            try:
                # 创建监听器
                events_iterator = self.client.watch_prefix(prefix)
                self._watchers[service_name] = events_iterator
                
                # 处理事件
                for event in events_iterator:
                    if self._stopped:
                        break
                        
                    # 更新缓存
                    await self.discover_service(service_name)
                    
                    # 记录变更
                    if event.value:
                        data = json.loads(event.value.decode())
                        service = ServiceInfo.from_dict(data)
                        logger.info(
                            f"Service changed: {service.name} "
                            f"(id={service.id})"
                        )
                    
            except Exception as e:
                logger.error(f"Watch error: {e}")
                await asyncio.sleep(1)  # 等待后重试
    
    async def _run_in_executor(self, func, *args, **kwargs):
        """在线程池中执行同步操作"""
        return await asyncio.get_event_loop().run_in_executor(
            self.executor,
            func,
            *args,
            **kwargs
        )
    
    def close(self):
        """关闭服务发现"""
        self._stopped = True
        for watcher in self._watchers.values():
            watcher.close()
        self._watchers.clear()
        self._cache.clear()
        self.executor.shutdown() 