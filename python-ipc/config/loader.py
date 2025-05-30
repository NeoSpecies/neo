import os
from pathlib import Path
from typing import Optional
import logging
from .config import ConfigManager, Config

logger = logging.getLogger(__name__)

class ConfigLoader:
    """配置加载器"""
    _instance: Optional['ConfigLoader'] = None
    _config_manager: Optional[ConfigManager] = None

    def __new__(cls):
        if cls._instance is None:
            cls._instance = super(ConfigLoader, cls).__new__(cls)
            cls._instance._initialize()
        return cls._instance

    def _initialize(self):
        """初始化配置管理器"""
        self._config_manager = ConfigManager()
        
        # 1. 加载默认配置
        default_config_path = Path(__file__).parent / 'default.yml'
        if default_config_path.exists():
            self._config_manager.load_from_file(str(default_config_path))
        
        # 2. 加载环境变量配置
        self._config_manager.load_from_env()
        
        # 3. 加载自定义配置文件
        config_path = os.getenv('IPC_CONFIG_FILE')
        if config_path and Path(config_path).exists():
            self._config_manager.load_from_file(config_path)
        
        # 4. 验证配置
        self._validate_config()
        
        # 5. 尝试从etcd加载配置（如果启用）
        try:
            if self.config.discovery.etcd.hosts:
                self._config_manager.load_from_etcd()
        except Exception as e:
            logger.warning(f"Failed to load configuration from etcd: {e}")

    def _validate_config(self):
        """验证配置有效性"""
        errors = self._config_manager.validate_config()
        if errors:
            error_msg = "\n".join(errors)
            raise ValueError(f"Invalid configuration:\n{error_msg}")

    @property
    def config(self) -> Config:
        """获取当前配置"""
        return self._config_manager.config

    def get_value(self, key: str, default: Optional[str] = None) -> Optional[str]:
        """获取配置值"""
        return self._config_manager.get_value(key, default)

    def reload(self):
        """重新加载配置"""
        self._initialize()

# 全局配置实例
config_loader = ConfigLoader()

def get_config() -> Config:
    """获取配置实例"""
    return config_loader.config 