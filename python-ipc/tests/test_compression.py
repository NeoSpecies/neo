import unittest
import os
from protocol.compression import (
    CompressionType,
    create_compressor,
    GzipCompression,
    ZstdCompression,
    Lz4Compression
)

class TestCompression(unittest.TestCase):
    def setUp(self):
        # Create test data with some repetitive content for better compression
        self.test_data = b"Hello World! " * 1000

    def test_compression_types(self):
        """Test all compression types"""
        for comp_type in CompressionType:
            compressor = create_compressor(comp_type)
            compressed = compressor.compress(self.test_data)
            decompressed = compressor.decompress(compressed)
            self.assertEqual(self.test_data, decompressed)

    def test_gzip_compression(self):
        """Test Gzip compression specifically"""
        compressor = GzipCompression()
        compressed = compressor.compress(self.test_data)
        self.assertLess(len(compressed), len(self.test_data))
        decompressed = compressor.decompress(compressed)
        self.assertEqual(self.test_data, decompressed)

    def test_zstd_compression(self):
        """Test Zstandard compression specifically"""
        compressor = ZstdCompression()
        compressed = compressor.compress(self.test_data)
        self.assertLess(len(compressed), len(self.test_data))
        decompressed = compressor.decompress(compressed)
        self.assertEqual(self.test_data, decompressed)

    def test_lz4_compression(self):
        """Test LZ4 compression specifically"""
        compressor = Lz4Compression()
        compressed = compressor.compress(self.test_data)
        self.assertLess(len(compressed), len(self.test_data))
        decompressed = compressor.decompress(compressed)
        self.assertEqual(self.test_data, decompressed)

    def test_compression_ratio(self):
        """Test compression ratios for different algorithms"""
        results = {}
        for comp_type in CompressionType:
            if comp_type == CompressionType.NONE:
                continue
            compressor = create_compressor(comp_type)
            compressed = compressor.compress(self.test_data)
            ratio = len(compressed) / len(self.test_data)
            results[comp_type.name] = ratio
            # All compression algorithms should achieve some compression
            self.assertLess(ratio, 1.0)

if __name__ == '__main__':
    unittest.main() 