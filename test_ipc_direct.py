"""直接测试IPC连接"""
import socket
import struct
import json
import time

def test_service_registration():
    """测试服务是否成功注册"""
    try:
        # 连接到Neo IPC服务器
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 9999))
        print("✓ Connected to Neo IPC server")
        
        # 检查是否能发送心跳
        # 格式: [Length:4][Type:1][ID:4][Service:4][Method:4][Metadata:4][Data:4]
        msg_type = 4  # HEARTBEAT
        content = bytearray()
        content.extend(struct.pack('<B', msg_type))  # 消息类型
        content.extend(struct.pack('<I', 0))  # ID长度
        content.extend(struct.pack('<I', 11))  # Service长度
        content.extend(b'test-client')  # Service名称
        content.extend(struct.pack('<I', 0))  # Method长度
        content.extend(struct.pack('<I', 2))  # Metadata长度
        content.extend(b'{}')  # Metadata
        content.extend(struct.pack('<I', 0))  # Data长度
        
        # 发送消息
        sock.send(struct.pack('<I', len(content)))  # 消息总长度
        sock.send(content)
        
        print("✓ Heartbeat sent successfully")
        
        sock.close()
        return True
        
    except Exception as e:
        print(f"✗ Error: {e}")
        return False

if __name__ == "__main__":
    print("Testing Neo IPC Protocol...")
    print("=" * 40)
    test_service_registration()