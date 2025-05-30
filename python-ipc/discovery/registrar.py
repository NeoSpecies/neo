import asyncio
import logging
import uuid
from typing import Dict, Optional, Any, List, Callable
from .discovery import ServiceDiscovery, ServiceInfo
from .health import HealthChecker, HealthCheck, HealthStatus

logger = logging.getLogger(__name__)

class ServiceRegistrar:
    """服务注册器"""
    
    def __init__(self,
                 discovery: ServiceDiscovery,
                 health_checker: Optional[HealthChecker] = None):
        """
        初始化服务注册器
        
        Args:
            discovery: 服务发现实例
            health_checker: 健康检查器实例
        """
        self.discovery = discovery
        self.health_checker = health_checker or HealthChecker()
        self._registered_services: Dict[str, ServiceInfo] = {}
        self._service_checks: Dict[str, List[str]] = {}
    
    async def register(self,
                      name: str,
                      host: str,
                      port: int,
                      metadata: Optional[Dict[str, Any]] = None,
                      version: str = "1.0.0",
                      weight: int = 100,
                      checks: Optional[List[Dict[str, Any]]] = None) -> str:
        """
        注册服务
        
        Args:
            name: 服务名称
            host: 服务主机
            port: 服务端口
            metadata: 服务元数据
            version: 服务版本
            weight: 服务权重
            checks: 健康检查配置列表，每个配置包含：
                   - name: 检查名称
                   - check_func: 检查函数
                   - config: HealthCheck配置（可选）
            
        Returns:
            服务实例ID
        """
        # 生成服务实例ID
        service_id = str(uuid.uuid4())
        
        # 创建服务信息
        service = ServiceInfo(
            name=name,
            id=service_id,
            host=host,
            port=port,
            metadata=metadata or {},
            version=version,
            weight=weight
        )
        
        # 注册服务
        success = await self.discovery.register_service(service)
        if not success:
            raise RuntimeError(f"Failed to register service: {name}")
        
        self._registered_services[service_id] = service
        
        # 添加健康检查
        if checks:
            check_names = []
            for check in checks:
                check_name = f"{service_id}_{check['name']}"
                check_names.append(check_name)
                
                self.health_checker.add_check(
                    name=check_name,
                    check_func=check['check_func'],
                    config=check.get('config')
                )
            
            self._service_checks[service_id] = check_names
            
            # 启动健康状态监控
            asyncio.create_task(
                self._monitor_health(service_id)
            )
        
        logger.info(
            f"Registered service: {name} "
            f"(id={service_id}, checks={len(checks or [])})"
        )
        
        return service_id
    
    async def deregister(self, service_id: str) -> bool:
        """
        注销服务
        
        Args:
            service_id: 服务实例ID
            
        Returns:
            注销是否成功
        """
        if service_id not in self._registered_services:
            return False
            
        service = self._registered_services[service_id]
        
        # 移除健康检查
        if service_id in self._service_checks:
            for check_name in self._service_checks[service_id]:
                self.health_checker.remove_check(check_name)
            del self._service_checks[service_id]
        
        # 注销服务
        success = await self.discovery.deregister_service(service)
        if success:
            del self._registered_services[service_id]
            logger.info(f"Deregistered service: {service.name} (id={service_id})")
        
        return success
    
    async def _monitor_health(self, service_id: str):
        """
        监控服务健康状态
        
        Args:
            service_id: 服务实例ID
        """
        if service_id not in self._service_checks:
            return
            
        service = self._registered_services[service_id]
        check_names = self._service_checks[service_id]
        
        while True:
            try:
                # 检查所有健康检查结果
                unhealthy = False
                for check_name in check_names:
                    result = self.health_checker.get_result(check_name)
                    if result and result.status == HealthStatus.UNHEALTHY:
                        unhealthy = True
                        break
                
                # 如果服务不健康，重新注册
                if unhealthy:
                    logger.warning(
                        f"Service unhealthy: {service.name} "
                        f"(id={service_id})"
                    )
                    await self.discovery.register_service(service)
                
            except Exception as e:
                logger.error(f"Health monitor error: {e}")
                
            finally:
                await asyncio.sleep(5)  # 每5秒检查一次
    
    def close(self):
        """关闭服务注册器"""
        self.health_checker.close()
        self.discovery.close() 