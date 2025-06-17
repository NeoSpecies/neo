import socket
import json
import time
import struct
import uuid
import zlib

# 协议常量（与server.go完全一致）
MAGIC_NUMBER = 0xAEBD  # 2字节大端魔数
VERSION = 0x01         # 1字节协议版本
SERVER_ADDR = '127.0.0.1'
SERVER_PORT = 9090

def create_tcp_socket():
    """创建TCP连接并设置超时"""
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(10)  # 延长超时时间至10秒
    try:
        sock.connect((SERVER_ADDR, SERVER_PORT))
        print(f'成功连接服务端：{SERVER_ADDR}:{SERVER_PORT}')
        return sock
    except Exception as e:
        raise Exception(f'连接服务端失败: {str(e)}')

def pack_protocol_message(method: str, service_info: dict) -> bytes:
    """按server.go协议规范打包请求数据"""
    buffer = bytearray()

    # 1. 魔数（2字节大端）
    buffer.extend(struct.pack('>H', MAGIC_NUMBER))
    # 2. 版本（1字节）
    buffer.append(VERSION)
    # 3. 消息ID（UUID，变长字段，需先写长度）
    msg_id = str(uuid.uuid4()).encode()
    msg_id_len = len(msg_id)
    buffer.extend(struct.pack('>H', msg_id_len))  # 消息ID长度（2字节大端）
    buffer.extend(msg_id)                        # 消息ID内容
    # 4. 方法名（变长字段，需先写长度）
    method_bytes = method.encode()
    method_len = len(method_bytes)
    buffer.extend(struct.pack('>H', method_len))  # 方法名长度（2字节大端）
    buffer.extend(method_bytes)                  # 方法名内容
    # 5. 参数内容（变长字段，需先写长度）
    params = {
        'action': 'register',
        'service': service_info,
        'name': service_info['name'],
        'id': service_info['id']
    }
    param_data = json.dumps(params).encode()
    param_len = len(param_data)
    buffer.extend(struct.pack('>I', param_len))   # 参数长度（4字节大端）
    buffer.extend(param_data)                    # 参数内容
    # 6. 计算CRC32校验和（仅计算前5部分数据）
    raw_data = bytes(buffer)
    checksum = zlib.crc32(raw_data) & 0xFFFFFFFF  # 转换为无符号32位整数
    buffer.extend(struct.pack('>I', checksum))   # 校验和（4字节大端）

    print(f'打包完成，总长度：{len(buffer)}字节，CRC32: 0x{checksum:X}')
    return bytes(buffer)

def register_service(sock, service_info):
    """发送注册请求并解析响应"""
    try:
        # 打包并发送请求
        packed_msg = pack_protocol_message('register', service_info)
        sock.sendall(packed_msg)
        print(f'已发送请求，长度：{len(packed_msg)}字节')

        # 解析响应（按server.go的响应格式）
        # 1. 读取魔数（2字节）
        magic_data = sock.recv(2)
        print(f'接收到魔数字节：{magic_data.hex()}')  # 新增：打印原始字节
        if len(magic_data) != 2:
            raise Exception('接收魔数失败，连接可能已断开')
        magic = struct.unpack('>H', magic_data)[0]
        print(f'解析魔数值：0x{magic:X}（期望0xAEBD）')  # 新增：打印解析值
        if magic != MAGIC_NUMBER:
            raise Exception(f'魔数校验失败，期望0xAEBD，实际0x{magic:X}')

        # 2. 读取版本（1字节）
        version_data = sock.recv(1)
        print(f'接收到版本字节：{version_data.hex()}')  # 新增：打印原始字节
        if len(version_data) != 1:
            raise Exception('接收版本号失败')
        version = version_data[0]
        print(f'解析版本值：{version}（期望1）')  # 新增：打印解析值
        if version != VERSION:
            raise Exception(f'版本号不匹配，期望1，实际{version}')

        # 3. 读取响应体长度（4字节）
        body_len_data = sock.recv(4)
        print(f'接收到响应体长度字节：{body_len_data.hex()}')  # 新增：打印原始字节
        if len(body_len_data) != 4:
            raise Exception('接收响应体长度失败')
        body_len = struct.unpack('>I', body_len_data)[0]
        print(f'解析响应体长度：{body_len}字节')  # 新增：打印解析值

        # 4. 读取响应体内容
        body_data = sock.recv(body_len)
        print(f'接收到响应体字节（前50字节）：{body_data[:50].hex()}...')  # 新增：打印原始内容
        if len(body_data) != body_len:
            raise Exception(f'接收响应体不完整，期望{body_len}字节，实际{len(body_data)}字节')
        try:
            response = json.loads(body_data)
            print(f'解析响应JSON成功：{response}')  # 新增：打印完整JSON
        except json.JSONDecodeError as e:
            raise Exception(f'响应体非有效JSON（原始数据：{body_data.decode(errors="replace")}）: {str(e)}')

        # 解析服务端返回的IPCResponse结构
        # 修复错误判断逻辑
        if response.get('error') not in (None, {}):
            raise Exception(f'注册失败: {response.get("error", "未知错误")}（原始响应：{response}）')
        
        # 从result字段获取服务ID
        if response.get('result') is None:
            print('服务端返回result字段为空（原始响应：{response}）')
            return None
        return response['result'].get('id')

    except socket.timeout:
        raise TimeoutError('等待服务端响应超时，请检查服务端是否正常运行')
    except Exception as e:
        # 新增：打印异常发生时的完整上下文
        raise Exception(f'注册过程异常: {str(e)}（当前sock状态：{sock.fileno()}）')

def test_service_registration():
    """主测试逻辑"""
    sock = None
    try:
        sock = create_tcp_socket()
        # 构造测试服务元数据（与discovery.Service结构体完全一致）
        test_service = {
            'id': str(uuid.uuid4()),
            'name': 'test-service',
            'address': '127.0.0.1',
            'port': 9090,
            'metadata': {'env': 'test'}, 
            'status': 'healthy',
            'expire_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
            'updated_at': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
        }
        print(f'构造的服务元数据：{test_service}')  # 新增：打印测试数据

        service_id = register_service(sock, test_service)
        if service_id:
            print(f'服务注册成功，ID: {service_id}')
        else:
            print('服务注册成功，但未返回ID（原始响应可能缺少data字段）')

    except Exception as e:
        print(f'测试失败: {str(e)}')  # 新增：打印完整异常信息
    finally:
        if sock:
            sock.close()

if __name__ == '__main__':
    test_service_registration()