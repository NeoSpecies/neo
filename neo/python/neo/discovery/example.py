import asyncio
import logging
import os
from discovery import ServiceDiscovery, ServiceRegistrar
from health import HealthCheck

# 配置日志
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# 从环境变量获取 etcd 配置，如果没有则使用默认值
ETCD_HOST = os.getenv("ETCD_HOST", "localhost")
ETCD_PORT = int(os.getenv("ETCD_PORT", "2379"))
ETCD_PREFIX = os.getenv("ETCD_PREFIX", "/services")

async def tcp_health_check():
    """TCP 端口健康检查"""
    # 模拟检查过程
    await asyncio.sleep(0.1)
    return {"status": "ok", "latency_ms": 100}

async def http_health_check():
    """HTTP 健康检查"""
    # 模拟检查过程
    await asyncio.sleep(0.2)
    return {"status": "ok", "response_code": 200}

async def main():
    # 创建服务发现，使用与 Go 服务相同的 etcd 配置
    discovery = ServiceDiscovery(
        host=ETCD_HOST,
        port=ETCD_PORT
    )
    
    # 创建服务注册器
    registrar = ServiceRegistrar(discovery)
    
    try:
        # 注册服务
        service_id = await registrar.register(
            name="example-service",
            host="localhost",  # 这里应该使用实际的服务主机地址
            port=8080,        # 这里应该使用实际的服务端口
            metadata={
                "version": "1.0.0",
                "environment": "development",
                "language": "python"
            },
            checks=[
                {
                    "name": "tcp",
                    "check_func": tcp_health_check,
                    "config": HealthCheck(
                        interval=5.0,
                        timeout=2.0
                    )
                },
                {
                    "name": "http",
                    "check_func": http_health_check,
                    "config": HealthCheck(
                        interval=10.0,
                        timeout=3.0
                    )
                }
            ]
        )
        
        logger.info(f"Service registered with ID: {service_id}")
        
        # 发现服务
        services = await discovery.discover_service("example-service")
        logger.info(f"Discovered services: {services}")
        
        # 运行一段时间
        await asyncio.sleep(30)
        
        # 注销服务
        success = await registrar.deregister(service_id)
        logger.info(f"Service deregistered: {success}")
        
    finally:
        # 关闭
        registrar.close()

if __name__ == "__main__":
    asyncio.run(main()) 