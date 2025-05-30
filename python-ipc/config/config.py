from dataclasses import dataclass, field
from typing import Dict, Any, Optional, List
import os
import yaml
import json
from pathlib import Path
import logging
from enum import Enum
import etcd3
from functools import lru_cache

logger = logging.getLogger(__name__)

class ConfigSource(Enum):
    """配置来源"""
    DEFAULT = "default"
    FILE = "file"
    ENV = "env"
    ETCD = "etcd"

@dataclass
class EtcdConfig:
    """Etcd配置"""
    hosts: List[str] = field(default_factory=lambda: ["localhost"])
    port: int = 2379
    prefix: str = "/config/ipc"
    timeout: float = 5.0
    username: str = ""
    password: str = ""

@dataclass
class PoolConfig:
    """连接池配置"""
    min_size: int = 5
    max_size: int = 20
    connection_timeout: float = 5.0
    idle_timeout: float = 60.0
    max_lifetime: float = 3600.0
    health_check_interval: float = 30.0
    balancer_strategy: str = "weighted_response_time"

@dataclass
class ProtocolConfig:
    """协议配置"""
    version: int = 1
    compression_algorithm: str = "none"
    max_message_size: int = 10 * 1024 * 1024  # 10MB
    enable_checksum: bool = True
    enable_tracing: bool = True

@dataclass
class MetricsConfig:
    """监控配置"""
    enable_prometheus: bool = True
    prometheus_port: int = 9090
    enable_tracing: bool = True
    tracing_sampler_rate: float = 0.1
    metrics_prefix: str = "ipc"

@dataclass
class DiscoveryConfig:
    """服务发现配置"""
    etcd: EtcdConfig = field(default_factory=EtcdConfig)
    service_ttl: int = 10
    refresh_interval: float = 5.0
    enable_health_check: bool = True

@dataclass
class Config:
    """IPC客户端配置"""
    pool: PoolConfig = field(default_factory=PoolConfig)
    protocol: ProtocolConfig = field(default_factory=ProtocolConfig)
    metrics: MetricsConfig = field(default_factory=MetricsConfig)
    discovery: DiscoveryConfig = field(default_factory=DiscoveryConfig)
    log_level: str = "INFO"
    environment: str = "development"

class ConfigManager:
    """配置管理器"""
    def __init__(self):
        self._config = Config()
        self._sources: Dict[str, ConfigSource] = {}
        self._etcd_client = None
        self._watchers = []

    @property
    def config(self) -> Config:
        """获取当前配置"""
        return self._config

    def load_from_file(self, file_path: str) -> None:
        """从文件加载配置"""
        try:
            with open(file_path, 'r') as f:
                if file_path.endswith('.yaml') or file_path.endswith('.yml'):
                    data = yaml.safe_load(f)
                elif file_path.endswith('.json'):
                    data = json.load(f)
                else:
                    raise ValueError(f"Unsupported file format: {file_path}")
                
                self._update_config(data, ConfigSource.FILE)
                logger.info(f"Loaded configuration from {file_path}")
        except Exception as e:
            logger.error(f"Failed to load configuration from {file_path}: {e}")
            raise

    def load_from_env(self, prefix: str = "IPC_") -> None:
        """从环境变量加载配置"""
        env_config = {}
        for key, value in os.environ.items():
            if key.startswith(prefix):
                config_key = key[len(prefix):].lower()
                try:
                    # 尝试解析JSON值
                    env_config[config_key] = json.loads(value)
                except json.JSONDecodeError:
                    # 如果不是JSON，则保持原始字符串
                    env_config[config_key] = value

        self._update_config(env_config, ConfigSource.ENV)
        logger.info("Loaded configuration from environment variables")

    def connect_etcd(self) -> None:
        """连接到etcd"""
        if not self._etcd_client:
            etcd_config = self._config.discovery.etcd
            self._etcd_client = etcd3.client(
                host=etcd_config.hosts[0],
                port=etcd_config.port,
                timeout=etcd_config.timeout,
                user=etcd_config.username or None,
                password=etcd_config.password or None
            )

    def load_from_etcd(self) -> None:
        """从etcd加载配置"""
        try:
            self.connect_etcd()
            prefix = self._config.discovery.etcd.prefix
            result = self._etcd_client.get_prefix(prefix)
            
            etcd_config = {}
            for value, metadata in result:
                if value:
                    key = metadata.key.decode('utf-8').replace(prefix + '/', '')
                    try:
                        etcd_config[key] = json.loads(value.decode('utf-8'))
                    except json.JSONDecodeError:
                        etcd_config[key] = value.decode('utf-8')

            self._update_config(etcd_config, ConfigSource.ETCD)
            logger.info("Loaded configuration from etcd")
            
            # 设置配置变更监听
            self._setup_etcd_watch()
        except Exception as e:
            logger.error(f"Failed to load configuration from etcd: {e}")
            raise

    def _setup_etcd_watch(self) -> None:
        """设置etcd配置变更监听"""
        def watch_callback(event):
            if event.events:
                for evt in event.events:
                    key = evt.key.decode('utf-8')
                    if evt.value:
                        try:
                            value = json.loads(evt.value.decode('utf-8'))
                            self._update_config({key: value}, ConfigSource.ETCD)
                            logger.info(f"Configuration updated from etcd: {key}")
                        except json.JSONDecodeError:
                            logger.warning(f"Invalid JSON in etcd value for key: {key}")

        watch_id = self._etcd_client.add_watch_prefix_callback(
            self._config.discovery.etcd.prefix,
            watch_callback
        )
        self._watchers.append(watch_id)

    def _update_config(self, data: Dict[str, Any], source: ConfigSource) -> None:
        """更新配置"""
        def update_nested(obj, path, value):
            if len(path) == 1:
                setattr(obj, path[0], value)
            else:
                update_nested(getattr(obj, path[0]), path[1:], value)

        for key, value in data.items():
            path = key.split('.')
            try:
                update_nested(self._config, path, value)
                self._sources[key] = source
            except AttributeError:
                logger.warning(f"Invalid configuration key: {key}")

    def get_value(self, key: str, default: Any = None) -> Any:
        """获取配置值"""
        try:
            path = key.split('.')
            value = self._config
            for part in path:
                value = getattr(value, part)
            return value
        except AttributeError:
            return default

    def get_source(self, key: str) -> Optional[ConfigSource]:
        """获取配置项的来源"""
        return self._sources.get(key)

    @lru_cache(maxsize=100)
    def validate_config(self) -> List[str]:
        """验证配置有效性"""
        errors = []
        
        # 验证连接池配置
        if self._config.pool.min_size < 1:
            errors.append("pool.min_size must be greater than 0")
        if self._config.pool.max_size < self._config.pool.min_size:
            errors.append("pool.max_size must be greater than or equal to pool.min_size")
        
        # 验证协议配置
        if self._config.protocol.max_message_size <= 0:
            errors.append("protocol.max_message_size must be greater than 0")
        if self._config.protocol.compression_algorithm not in ["none", "gzip", "zstd", "lz4"]:
            errors.append("protocol.compression_algorithm must be one of: none, gzip, zstd, lz4")
        
        # 验证监控配置
        if self._config.metrics.prometheus_port < 1024 or self._config.metrics.prometheus_port > 65535:
            errors.append("metrics.prometheus_port must be between 1024 and 65535")
        if not 0 <= self._config.metrics.tracing_sampler_rate <= 1:
            errors.append("metrics.tracing_sampler_rate must be between 0 and 1")
        
        return errors

    def __del__(self):
        """清理资源"""
        if self._etcd_client:
            for watch_id in self._watchers:
                self._etcd_client.cancel_watch(watch_id)
            self._etcd_client.close() 