#!/usr/bin/env python3
"""
简单的IPC客户端测试，直接发送注册消息
"""
import socket
import struct
import json
import time

def send_register_message():
    """发送服务注册消息到Go IPC服务器"""
    try:
        # 连接到IPC服务器
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.connect(('localhost', 37999))
        print("✅ 连接到IPC服务器成功")
        
        # 构建注册消息数据
        service_name = "test.service"
        reg_data = {
            "name": service_name,
            "metadata": {"version": "1.0.0", "test": "true"}
        }
        
        # 构建IPC消息
        msg_type = 3  # REGISTER
        msg_id = ""
        method = ""
        metadata = {}
        data = json.dumps(reg_data).encode('utf-8')
        
        # 编码消息（使用小端字节序）
        buf = bytearray()
        buf.extend(struct.pack('<B', msg_type))
        
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
        total_len = len(buf)
        sock.send(struct.pack('<I', total_len))
        sock.send(buf)
        
        print(f"✅ 发送注册消息成功:")
        print(f"   服务名: {service_name}")
        print(f"   消息长度: {total_len} 字节")
        print(f"   数据: {reg_data}")
        
        # 保持连接一段时间
        print("⏳ 保持连接5秒...")
        time.sleep(5)
        
        sock.close()
        print("✅ 连接关闭")
        
    except Exception as e:
        print(f"❌ 发送失败: {e}")

if __name__ == "__main__":
    send_register_message()