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
            param_data = json.dumps(params, ensure_ascii=False)  # 关闭转义（确保发送的参数含中文）
            request = f"{method}|{param_data}".encode()
            print(f"Python 发送的请求数据：{request}")
            s.send(request)
            s.shutdown(socket.SHUT_WR)  # 关闭写端，通知 Go 服务数据已发送完成
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