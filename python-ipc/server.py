import socket
import json
import threading
import struct
from threading import Thread

class IpcServer:
    def __init__(self, addr):
        self.addr = addr
        self.services = {}

    def register(self, name, handler):
        self.services[name] = handler

    def start(self):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.bind(self.addr)
        self.sock.listen(5)
        print(f"Python IPC 服务启动，监听 {self.addr}")
        while True:
            conn, addr = self.sock.accept()
            Thread(target=self.handle_connection, args=(conn,)).start()

    # 假设 server.py 中存在 handle_connection 函数，修改响应构造部分
    def handle_connection(self, conn):
        try:
            reader = conn.makefile('rb')
            
            # 1. 读取并验证魔数（2字节，大端）
            magic_bytes = reader.read(2)
            if len(magic_bytes) != 2:
                raise Exception("魔数缺失")
            magic = struct.unpack(">H", magic_bytes)[0]
            if magic != 0xAEBD:
                raise Exception(f"无效魔数: {magic:x}")

            # 2. 读取版本（1字节）
            version = reader.read(1)[0]
            if version > 1:
                raise Exception(f"不支持的协议版本: {version}")

            # 3. 读取消息ID长度（2字节，大端）
            msg_id_len = struct.unpack(">H", reader.read(2))[0]
            msg_id = reader.read(msg_id_len).decode()

            # 4. 读取方法名长度（2字节，大端）
            method_len = struct.unpack(">H", reader.read(2))[0]
            method = reader.read(method_len).decode()

            # 5. 读取参数长度（4字节，大端）
            param_len = struct.unpack(">I", reader.read(4))[0]
            param_data = reader.read(param_len)
            params = json.loads(param_data)

            # 6. 调用注册的服务
            if method not in self.services:
                raise Exception(f"服务 {method} 未注册")
            # 调用服务并获取结果
            result, err = self.services[method](params)  # 假设服务函数返回 (result, error)
            
            # 构造响应（关键修复：明确将服务结果赋值给 "result" 字段）
            response_body = json.dumps({
                "msg_id": msg_id,
                "result": result,  # 确保此处使用服务函数的返回结果
                "error": str(err) if err else None
            }).encode()
            response = bytearray()
            # 魔数（2字节，大端）
            response.extend(struct.pack(">H", 0xAEBD))
            # 版本（1字节）
            response.append(0x01)
            # 响应体长度（4字节，大端）
            response.extend(struct.pack(">I", len(response_body)))
            # 响应体内容
            response.extend(response_body)

            # 构造响应（关键修复：使用 sendall 确保完整发送）
            response_body = json.dumps({
                "msg_id": msg_id,
                "result": result,
                "error": str(err) if err else None
            }).encode()
            response = bytearray()
            response.extend(struct.pack(">H", 0xAEBD))  # 魔数（2字节，大端）
            response.append(0x01)  # 版本（1字节）
            response.extend(struct.pack(">I", len(response_body)))  # 响应体长度（4字节）
            response.extend(response_body)  # 响应体内容

            # 新增：打印发送给 Go 的完整响应（二进制和 JSON 内容）
            print(f"Python 发送给 Go 的响应（二进制）: {response.hex()}")
            print(f"Python 发送给 Go 的响应（JSON 内容）: {response_body.decode('utf-8')}")

            conn.sendall(response)  # 发送响应
        except Exception as e:
            # 错误响应也需包含协议头
            error_body = json.dumps({"error": str(e)}).encode()
            error_response = bytearray()
            error_response.extend(struct.pack(">H", 0xAEBD))
            error_response.append(0x01)
            error_response.extend(struct.pack(">I", len(error_body)))
            error_response.extend(error_body)
            # 新增：打印错误响应（二进制和 JSON 内容）
            print(f"Python 发送给 Go 的错误响应（二进制）: {error_response.hex()}")
            print(f"Python 发送给 Go 的错误响应（JSON 内容）: {error_body.decode('utf-8')}")
            conn.sendall(error_response)
        finally:
            conn.close()