import asyncio
import logging
import uuid
import datetime  # 新增导入
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
                      address: str,
                      port: int,
                      metadata: Optional[Dict[str, str]] = None,
                      version: str = "1.0.0",
                      checks: Optional[List[Dict[str, Any]]] = None) -> str:
        """注册服务并启动健康检查"""
        # 生成服务实例ID
        service_id = str(uuid.uuid4())
        now = datetime.datetime.utcnow()
    
        # 创建服务信息（与test.py保持一致的字段）
        service_info = ServiceInfo(
            id=service_id,
            name=name,  # 修复：添加参数名称
            address=address,
            port=port,
            metadata=metadata or {},
            status="healthy",
            expire_at=(now + datetime.timedelta(seconds=30)).isoformat() + "Z",
            updated_at=now.isoformat() + "Z"
        )  # 确保此处有闭合括号
    
        # 注册服务
        success = await self.discovery.register_service(service_info)
        if not success:
            raise RuntimeError(f"Failed to register service: {name}")
        
        self._registered_services[service_id] = service_info  # 将service改为service_info
        
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
        """监控服务健康状态并同步到IPC服务发现"""
        service = self._registered_services[service_id]
        check_names = self._service_checks[service_id]
        
        while True:
            try:
                # 检查所有健康检查
                unhealthy = any(
                    self.health_checker.get_result(name).status == HealthStatus.UNHEALTHY
                    for name in check_names
                )

                # 更新服务状态并同步到服务发现
                new_status = "unhealthy" if unhealthy else "healthy"
                if service.status != new_status:
                    service.status = new_status
                    service.updated_at = datetime.datetime.utcnow().isoformat() + "Z"
                    # 通过IPC更新服务状态
                    await self.discovery.register_service(service)
                    logger.info(f"Service status updated: {service.name} -> {new_status}")

                await asyncio.sleep(5)  # 每5秒检查一次
            except Exception as e:
                logger.error(f"Health monitor error: {e}")
                await asyncio.sleep(1)
    
    def close(self):
        """关闭服务注册器"""
        self.health_checker.close()
        self.discovery.close()