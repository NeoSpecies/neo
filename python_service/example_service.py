import asyncio
import logging
from neo_client import NeoIPCClient

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class PythonMathService:
    """示例Python服务，提供数学运算功能"""
    
    def __init__(self):
        # 尝试从环境配置文件读取端口
        ipc_port = 29999  # 默认端口
        
        import os
        env_file = os.path.join(os.environ.get('TEMP', '/tmp'), 'neo_ports.env')
        if os.path.exists(env_file):
            try:
                with open(env_file, 'r') as f:
                    for line in f:
                        if line.startswith('NEO_IPC_PORT='):
                            ipc_port = int(line.split('=')[1].strip())
                            logger.info(f"Using IPC port from config: {ipc_port}")
                            break
            except Exception as e:
                logger.warning(f"Failed to read port config: {e}")
        
        self.client = NeoIPCClient(port=ipc_port)
        
    async def start(self):
        """启动服务"""
        # 连接到Neo IPC服务器
        await self.client.connect()
        
        # 注册服务
        await self.client.register_service(
            "python.math",
            metadata={
                "version": "1.0.0",
                "language": "python",
                "async": "true"
            }
        )
        
        # 注册处理器
        self.client.register_handler("add", self.handle_add)
        self.client.register_handler("multiply", self.handle_multiply)
        self.client.register_handler("calculate", self.handle_calculate)
        
        # 开始监听
        await self.client.listen()
        
    async def handle_add(self, data: dict) -> dict:
        """处理加法请求"""
        a = data.get("a", 0)
        b = data.get("b", 0)
        result = a + b
        logger.info(f"Add: {a} + {b} = {result}")
        return {"result": result}
        
    async def handle_multiply(self, data: dict) -> dict:
        """处理乘法请求"""
        a = data.get("a", 0)
        b = data.get("b", 0)
        result = a * b
        logger.info(f"Multiply: {a} * {b} = {result}")
        return {"result": result}
        
    async def handle_calculate(self, data: dict) -> dict:
        """处理复杂计算请求"""
        expression = data.get("expression", "")
        try:
            # 注意：在生产环境中应该使用更安全的表达式求值方法
            result = eval(expression, {"__builtins__": {}}, {})
            logger.info(f"Calculate: {expression} = {result}")
            return {"result": result}
        except Exception as e:
            logger.error(f"Calculate error: {e}")
            return {"error": str(e)}


async def main():
    """主函数"""
    service = PythonMathService()
    
    try:
        logger.info("Starting Python Math Service...")
        await service.start()
    except KeyboardInterrupt:
        logger.info("Service interrupted by user")
    except Exception as e:
        logger.error(f"Service error: {e}")
    finally:
        await service.client.close()


if __name__ == "__main__":
    asyncio.run(main())