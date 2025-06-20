import gzip
import zstd
import lz4.frame
from typing import Callable, Dict, Optional
import logging

logger = logging.getLogger(__name__)

class CompressionError(Exception):
    """压缩/解压缩错误"""
    pass

class CompressionManager:
    """压缩算法管理器"""
    
    # 支持的压缩算法
    ALGORITHMS: Dict[str, Dict[str, Callable]] = {
        'none': {
            'compress': lambda x: x,
            'decompress': lambda x: x
        },
        'gzip': {
            'compress': lambda x: gzip.compress(x, compresslevel=6),
            'decompress': gzip.decompress
        },
        'zstd': {
            'compress': lambda x: zstd.compress(x, level=3),
            'decompress': zstd.decompress
        },
        'lz4': {
            'compress': lambda x: lz4.frame.compress(x, compression_level=0),
            'decompress': lz4.frame.decompress
        }
    }
    
    def __init__(self, algorithm: str = 'none'):
        """
        初始化压缩管理器
        
        Args:
            algorithm: 压缩算法名称，支持 'none', 'gzip', 'zstd', 'lz4'
        """
        if algorithm not in self.ALGORITHMS:
            raise ValueError(f"Unsupported compression algorithm: {algorithm}")
        
        self.algorithm = algorithm
        self._compress_func = self.ALGORITHMS[algorithm]['compress']
        self._decompress_func = self.ALGORITHMS[algorithm]['decompress']
    
    def compress(self, data: bytes) -> bytes:
        """
        压缩数据
        
        Args:
            data: 要压缩的数据
            
        Returns:
            压缩后的数据
            
        Raises:
            CompressionError: 压缩失败
        """
        try:
            compressed = self._compress_func(data)
            compression_ratio = len(compressed) / len(data) if data else 1.0
            logger.debug(
                f"Compressed {len(data)} bytes to {len(compressed)} bytes "
                f"using {self.algorithm} (ratio: {compression_ratio:.2f})"
            )
            return compressed
        except Exception as e:
            raise CompressionError(f"Compression failed: {e}")
    
    def decompress(self, data: bytes) -> bytes:
        """
        解压缩数据
        
        Args:
            data: 要解压缩的数据
            
        Returns:
            解压缩后的数据
            
        Raises:
            CompressionError: 解压缩失败
        """
        try:
            decompressed = self._decompress_func(data)
            logger.debug(
                f"Decompressed {len(data)} bytes to {len(decompressed)} bytes "
                f"using {self.algorithm}"
            )
            return decompressed
        except Exception as e:
            raise CompressionError(f"Decompression failed: {e}")
    
    @classmethod
    def get_supported_algorithms(cls) -> list:
        """获取支持的压缩算法列表"""
        return list(cls.ALGORITHMS.keys())
    
    def get_algorithm(self) -> str:
        """获取当前使用的压缩算法"""
        return self.algorithm
    
    def estimate_compression_ratio(self, data: bytes) -> float:
        """
        估算压缩比
        
        Args:
            data: 样本数据
            
        Returns:
            预计压缩比 (压缩后大小/原始大小)
        """
        if not data:
            return 1.0
        try:
            compressed = self.compress(data)
            return len(compressed) / len(data)
        except:
            return 1.0 