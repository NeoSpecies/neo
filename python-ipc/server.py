import socket
import json
import threading

MAGIC_NUMBER = 0xAEBD  # 与 Go 通信层一致（设计文档魔数）

class IpcServer:
    def __init__(self, addr):
        self.addr = addr
        self.services = {}  # 服务注册表（设计文档服务发现）
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.bind(addr)
        self.sock.listen(5)

    def register(self, name, handler):
        self.services[name] = handler  # 注册服务（设计文档动态注册）

    def start(self):
        print(f"Python IPC 服务启动，监听 {self.addr}")
        while True:
            conn, addr = self.sock.accept()
            threading.Thread(target=self.handle_connection, args=(conn,)).start()  # 多线程处理（模拟 Go 协程）

    def handle_connection(self, conn):
        with conn:
            # 简化协议解析（MVP 阶段仅处理方法名和参数）
            data = conn.recv(1024).decode()
            method, params = data.split("|", 1)
            params = json.loads(params)

            # 调用注册的服务（设计文档路由分发）
            handler = self.services.get(method)
            if handler:
                result = handler(params)
            else:
                result = {"error": f"服务 {method} 未找到"}

            # 返回结果给 Go 通信层
            conn.send(json.dumps(result).encode())