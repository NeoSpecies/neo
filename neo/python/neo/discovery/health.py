import asyncio
import logging
from typing import Callable, Dict, Optional, Any
from dataclasses import dataclass
import time
from enum import Enum

logger = logging.getLogger(__name__)

class HealthStatus(Enum):
    """健康状态"""
    UNKNOWN = "unknown"
    HEALTHY = "healthy"
    UNHEALTHY = "unhealthy"

@dataclass
class HealthCheck:
    """健康检查配置"""
    interval: float = 10.0    # 检查间隔（秒）
    timeout: float = 5.0      # 超时时间（秒）
    retries: int = 3         # 重试次数
    initial_delay: float = 0  # 初始延迟（秒）

@dataclass
class HealthResult:
    """健康检查结果"""
    status: HealthStatus
    timestamp: float
    details: Dict[str, Any]
    error: Optional[str] = None

class HealthChecker:
    """健康检查器"""
    
    def __init__(self):
        self._checks: Dict[str, Callable] = {}
        self._results: Dict[str, HealthResult] = {}
        self._tasks: Dict[str, asyncio.Task] = {}
        self._stopped = False
    
    def add_check(self, 
                 name: str, 
                 check_func: Callable,
                 config: Optional[HealthCheck] = None) -> None:
        """
        添加健康检查
        
        Args:
            name: 检查名称
            check_func: 检查函数，必须是异步函数
            config: 检查配置
        """
        self._checks[name] = check_func
        self._results[name] = HealthResult(
            status=HealthStatus.UNKNOWN,
            timestamp=time.time(),
            details={}
        )
        
        # 启动检查任务
        if name not in self._tasks:
            config = config or HealthCheck()
            task = asyncio.create_task(
                self._run_check(name, check_func, config)
            )
            self._tasks[name] = task
    
    def remove_check(self, name: str) -> None:
        """
        移除健康检查
        
        Args:
            name: 检查名称
        """
        if name in self._tasks:
            self._tasks[name].cancel()
            del self._tasks[name]
        
        if name in self._checks:
            del self._checks[name]
            
        if name in self._results:
            del self._results[name]
    
    def get_result(self, name: str) -> Optional[HealthResult]:
        """
        获取检查结果
        
        Args:
            name: 检查名称
            
        Returns:
            检查结果
        """
        return self._results.get(name)
    
    def get_all_results(self) -> Dict[str, HealthResult]:
        """
        获取所有检查结果
        
        Returns:
            所有检查结果
        """
        return self._results.copy()
    
    async def _run_check(self, 
                        name: str, 
                        check_func: Callable,
                        config: HealthCheck):
        """
        运行健康检查
        
        Args:
            name: 检查名称
            check_func: 检查函数
            config: 检查配置
        """
        # 初始延迟
        if config.initial_delay > 0:
            await asyncio.sleep(config.initial_delay)
        
        while not self._stopped:
            try:
                # 执行检查
                details = {}
                error = None
                status = HealthStatus.HEALTHY
                
                for _ in range(config.retries):
                    try:
                        # 设置超时
                        details = await asyncio.wait_for(
                            check_func(),
                            timeout=config.timeout
                        )
                        error = None
                        break
                        
                    except asyncio.TimeoutError:
                        error = "Check timeout"
                        status = HealthStatus.UNHEALTHY
                        
                    except Exception as e:
                        error = str(e)
                        status = HealthStatus.UNHEALTHY
                        
                    # 重试前等待
                    if _ < config.retries - 1:
                        await asyncio.sleep(1)
                
                # 更新结果
                self._results[name] = HealthResult(
                    status=status,
                    timestamp=time.time(),
                    details=details or {},
                    error=error
                )
                
                # 记录日志
                if status == HealthStatus.UNHEALTHY:
                    logger.warning(
                        f"Health check failed: {name} "
                        f"(error={error})"
                    )
                else:
                    logger.debug(
                        f"Health check passed: {name}"
                    )
                
            except Exception as e:
                logger.error(f"Health check error: {e}")
                
            finally:
                # 等待下次检查
                await asyncio.sleep(config.interval)
    
    def close(self):
        """关闭健康检查器"""
        self._stopped = True
        for task in self._tasks.values():
            task.cancel()
        self._tasks.clear()
        self._checks.clear()
        self._results.clear() 