import json
import zlib
import uuid
import struct  # 新增：导入struct模块
from typing import Dict, Any

# IPC协议常量（与Go服务端保持一致）
MAGIC_NUMBER = 0xAEBD
VERSION = 0x01

class IPCProtocolError(Exception):
    """IPC协议处理异常"""

async def pack_message(method: str, params: Dict[str, Any]) -> bytes:
    """打包IPC协议消息（参考test.py实现）"""
    buffer = bytearray()
    
    # 1. 魔数(2字节大端)
    buffer.extend(struct.pack('>H', MAGIC_NUMBER))
    # 2. 版本(1字节)
    buffer.append(VERSION)
    # 3. 消息ID(UUID)
    msg_id = str(uuid.uuid4()).encode()
    buffer.extend(struct.pack('>H', len(msg_id)))
    buffer.extend(msg_id)
    # 4. 方法名
    method_bytes = method.encode()
    buffer.extend(struct.pack('>H', len(method_bytes)))
    buffer.extend(method_bytes)
    # 5. 参数内容
    param_data = json.dumps(params).encode()
    buffer.extend(struct.pack('>I', len(param_data)))
    buffer.extend(param_data)
    # 6. CRC32校验和
    checksum = zlib.crc32(buffer) & 0xFFFFFFFF
    buffer.extend(struct.pack('>I', checksum))
    
    return bytes(buffer)

async def unpack_response(data: bytes) -> Dict[str, Any]:
    "解包IPC响应消息"
    # 实现参考test.py中的响应解析逻辑
    offset = 0
    
    # 1. 魔数校验
    if len(data) < 2:
        raise IPCProtocolError("响应数据过短，无法解析魔数")
    magic = struct.unpack('>H', data[offset:offset+2])[0]
    offset += 2
    if magic != MAGIC_NUMBER:
        raise IPCProtocolError(f"魔数校验失败，期望0x{MAGIC_NUMBER:X}，实际0x{magic:X}")
    
    # 2. 版本校验
    if len(data) < offset + 1:
        raise IPCProtocolError("无法解析版本号")
    version = data[offset]
    offset += 1
    if version != VERSION:
        raise IPCProtocolError(f"版本不匹配，期望{VERSION}，实际{version}")
    
    # 3. 响应体长度
    if len(data) < offset + 4:
        raise IPCProtocolError("无法解析响应体长度")
    body_len = struct.unpack('>I', data[offset:offset+4])[0]
    offset += 4
    
    # 4. 响应体内容
    if len(data) < offset + body_len:
        raise IPCProtocolError(f"响应体不完整，期望{body_len}字节")
    body_data = data[offset:offset+body_len]
    
    try:
        return json.loads(body_data)
    except json.JSONDecodeError as e:
        raise IPCProtocolError(f"响应体JSON解析失败: {str(e)}")