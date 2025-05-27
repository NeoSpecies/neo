import json

from server import IpcServer
import socket

def python_demo_func(params):
    print(f"Python 函数接收到参数：{params}")
    
    # 调用 Go 的测试函数（设计文档跨语言调用）
    go_result = call_go_service("go.service.test", {"input": params["input"] + " (Python 处理后)"})
    # 返回字典（由 IpcServer 自动序列化，避免手动转义）
    return {
        "python处理结果": "处理完成",
        "go调用结果": go_result
    }  # 关键修改：返回字典而非 json.dumps 后的字符串

def call_go_service(method, params):
    print(f"Python 调用 Go 服务：method={method}, params={params}")
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            s.connect(("127.0.0.1", 9090))
            
            # 1. 序列化参数（设计文档参数序列化）
            param_data = json.dumps(params, ensure_ascii=False).encode()
            
            # 2. 生成消息ID（设计文档异步模式）
            msg_id = str(uuid.uuid4()).encode()  # 需要import uuid
            
            # 3. 按协议格式封装请求（魔数+版本+消息ID+方法名+参数）
            request = bytearray()
            # 魔数（2字节，大端）
            request.extend(struct.pack(">H", 0xAEBD))  # 需要import struct
            # 版本（1字节）
            request.append(0x01)
            # 消息ID长度（2字节）+ 消息ID
            request.extend(struct.pack(">H", len(msg_id)))
            request.extend(msg_id)
            # 方法名长度（2字节）+ 方法名
            method_bytes = method.encode()
            request.extend(struct.pack(">H", len(method_bytes)))
            request.extend(method_bytes)
            # 参数长度（4字节）+ 参数内容
            request.extend(struct.pack(">I", len(param_data)))
            request.extend(param_data)
            
            print(f"Python 发送的请求数据：{request.hex()}")
            s.send(request)
            s.shutdown(socket.SHUT_WR)
        except Exception as e:
            print(f"连接或发送数据失败：{e}")
            return ""
        
        data = []
        while True:
            chunk = s.recv(1024)
            if not chunk:
                break
            data.append(chunk)
        response = b''.join(data).decode()
        print(f"Python 接收到 Go 响应：{response}")
        return response  # 假设 Go 响应是含中文的字符串（如 "Go 测试函数返回：Hello (Python 处理后)"）

if __name__ == "__main__":
    server = IpcServer(("127.0.0.1", 9091))
    server.register("python.service.demo", python_demo_func)
    server.start()