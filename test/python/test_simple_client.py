#!/usr/bin/env python3
import socket
import struct
import json
import time

def test_ipc_connection():
    """简单测试IPC连接"""
    try:
        # 连接到IPC服务器
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 29999))
        print("✅ 成功连接到IPC服务器 29999")
        
        # 构建注册消息
        service_name = "test.service"
        reg_data = {
            "name": service_name,
            "metadata": {"version": "1.0.0"}
        }
        
        # 构建IPC消息
        msg_type = 3  # REGISTER
        msg_id = "test-123"
        method = "register"
        metadata = {}
        data = json.dumps(reg_data).encode('utf-8')
        
        # 编码消息
        buf = bytearray()
        buf.append(msg_type)
        
        # ID
        id_bytes = msg_id.encode('utf-8')
        buf.extend(struct.pack('<I', len(id_bytes)))
        buf.extend(id_bytes)
        
        # Service
        service_bytes = service_name.encode('utf-8')
        buf.extend(struct.pack('<I', len(service_bytes)))
        buf.extend(service_bytes)
        
        # Method
        method_bytes = method.encode('utf-8')
        buf.extend(struct.pack('<I', len(method_bytes)))
        buf.extend(method_bytes)
        
        # Metadata
        metadata_bytes = json.dumps(metadata).encode('utf-8')
        buf.extend(struct.pack('<I', len(metadata_bytes)))
        buf.extend(metadata_bytes)
        
        # Data
        buf.extend(struct.pack('<I', len(data)))
        buf.extend(data)
        
        # 发送消息长度和消息
        sock.send(struct.pack('<I', len(buf)))
        sock.send(buf)
        
        print(f"✅ 发送注册消息: {service_name}")
        
        # 等待一下让服务器处理
        time.sleep(1)
        
        sock.close()
        print("✅ 连接关闭")
        
    except Exception as e:
        print(f"❌ 连接失败: {e}")

if __name__ == "__main__":
    test_ipc_connection()