import unittest
import time
from protocol.protocol import (
    Protocol, Message, MessageType, Priority,
    MessageHeader, ProtocolError
)

class TestProtocol(unittest.TestCase):
    def setUp(self):
        self.protocol = Protocol()
        self.test_payload = b"Hello, World!"
    
    def test_message_header(self):
        """测试消息头打包和解包"""
        header = MessageHeader(
            msg_type=MessageType.REQUEST,
            compression='none',
            priority=Priority.NORMAL,
            trace_id='test-trace-id',
            timestamp=time.time(),
            payload_size=len(self.test_payload),
            checksum=0x12345678
        )
        
        # 测试打包
        packed = header.pack()
        self.assertEqual(len(packed), MessageHeader.HEADER_SIZE)
        
        # 测试解包
        unpacked = MessageHeader.unpack(packed)
        self.assertEqual(unpacked.msg_type, header.msg_type)
        self.assertEqual(unpacked.compression, header.compression)
        self.assertEqual(unpacked.priority, header.priority)
        self.assertEqual(unpacked.payload_size, header.payload_size)
        self.assertEqual(unpacked.checksum, header.checksum)
    
    def test_protocol_pack_unpack(self):
        """测试消息打包和解包"""
        # 打包消息
        packed = self.protocol.pack(
            self.test_payload,
            msg_type=MessageType.REQUEST,
            priority=Priority.HIGH
        )
        
        # 解包消息
        message = self.protocol.unpack(packed)
        
        # 验证消息内容
        self.assertEqual(message.msg_type, MessageType.REQUEST)
        self.assertEqual(message.priority, Priority.HIGH)
        self.assertEqual(message.payload, self.test_payload)
        self.assertTrue(message.trace_id)  # 确保生成了追踪ID
    
    def test_compression_algorithms(self):
        """测试不同压缩算法"""
        test_data = b"x" * 1000  # 创建可压缩的数据
        
        for algorithm in ['none', 'gzip', 'zstd', 'lz4']:
            with self.subTest(algorithm=algorithm):
                protocol = Protocol(compression=algorithm)
                
                # 打包消息
                packed = protocol.pack(test_data)
                
                # 解包消息
                message = protocol.unpack(packed)
                
                # 验证数据完整性
                self.assertEqual(message.payload, test_data)
    
    def test_message_types(self):
        """测试不同消息类型"""
        for msg_type in MessageType:
            with self.subTest(msg_type=msg_type):
                packed = self.protocol.pack(
                    self.test_payload,
                    msg_type=msg_type
                )
                message = self.protocol.unpack(packed)
                self.assertEqual(message.msg_type, msg_type)
    
    def test_priorities(self):
        """测试不同优先级"""
        for priority in Priority:
            with self.subTest(priority=priority):
                packed = self.protocol.pack(
                    self.test_payload,
                    priority=priority
                )
                message = self.protocol.unpack(packed)
                self.assertEqual(message.priority, priority)
    
    def test_checksum_verification(self):
        """测试校验和验证"""
        # 正常情况
        packed = self.protocol.pack(self.test_payload)
        message = self.protocol.unpack(packed)
        self.assertEqual(message.payload, self.test_payload)
        
        # 数据损坏
        packed = bytearray(packed)
        packed[-1] ^= 0xFF  # 修改最后一个字节
        with self.assertRaises(ProtocolError):
            self.protocol.unpack(bytes(packed))
    
    def test_trace_id_uniqueness(self):
        """测试追踪ID唯一性"""
        trace_ids = set()
        for _ in range(1000):
            packed = self.protocol.pack(self.test_payload)
            message = self.protocol.unpack(packed)
            self.assertNotIn(message.trace_id, trace_ids)
            trace_ids.add(message.trace_id)
    
    def test_response_creation(self):
        """测试响应消息创建"""
        # 创建请求消息
        request_packed = self.protocol.pack(
            b"request",
            priority=Priority.HIGH
        )
        request = self.protocol.unpack(request_packed)
        
        # 创建响应消息
        response_packed = self.protocol.create_response(
            request,
            b"response"
        )
        response = self.protocol.unpack(response_packed)
        
        # 验证响应
        self.assertEqual(response.msg_type, MessageType.RESPONSE)
        self.assertEqual(response.priority, request.priority)
        self.assertEqual(response.payload, b"response")
    
    def test_error_creation(self):
        """测试错误消息创建"""
        # 创建请求消息
        request_packed = self.protocol.pack(b"request")
        request = self.protocol.unpack(request_packed)
        
        # 创建错误消息
        error_packed = self.protocol.create_error(
            request,
            "Test error"
        )
        error = self.protocol.unpack(error_packed)
        
        # 验证错误消息
        self.assertEqual(error.msg_type, MessageType.ERROR)
        self.assertEqual(error.payload, b"Test error")
    
    def test_heartbeat(self):
        """测试心跳消息"""
        # 创建心跳消息
        heartbeat_packed = self.protocol.create_heartbeat()
        heartbeat = self.protocol.unpack(heartbeat_packed)
        
        # 验证心跳消息
        self.assertEqual(heartbeat.msg_type, MessageType.HEARTBEAT)
        self.assertEqual(heartbeat.priority, Priority.LOW)
        self.assertEqual(heartbeat.payload, b"")
    
    def test_invalid_data(self):
        """测试无效数据处理"""
        # 测试空数据
        with self.assertRaises(ProtocolError):
            self.protocol.unpack(b"")
        
        # 测试太短的数据
        with self.assertRaises(ProtocolError):
            self.protocol.unpack(b"too short")
        
        # 测试无效的消息类型
        with self.assertRaises(ProtocolError):
            self.protocol.unpack(b"\xFF" * MessageHeader.HEADER_SIZE)
        
        # 测试无效的压缩算法
        with self.assertRaises(ValueError):
            Protocol(compression="invalid")

if __name__ == '__main__':
    unittest.main() 