import json
import struct
import uuid
import zlib  # 新增：用于计算CRC32校验和
from server import IpcServer
import socket
import os  # 确保已导入 os（文件处理需要）
import base64  # 将base64导入移至文件顶部，规范代码结构

def python_demo_func(params):
    print(f"Python 函数接收到参数：{params}")
    
    # 调用 Go 服务（显式解包返回的元组）
    go_result, err = call_go_service("go.service.test", {
        "input": params["input"] + " (Python 处理后)"
    })
    if err is not None:
        raise Exception(f"调用 Go 服务失败: {err}")  # 处理错误情况
    
    return {
        "python处理结果": "处理完成",
        "go调用结果": go_result
    }, None
    
    # 调用 Go 的测试函数（设计文档跨语言调用）
    # 新增：支持传递文件参数（示例）
    files = params.get("files", [])  # 假设参数中包含文件列表
    go_result = call_go_service("go.service.test", {
        "input": params["input"] + " (Python 处理后)",
        "files_info": [f["meta"] for f in files]  # 传递文件元数据摘要
    }, files)  # 新增：传递文件对象
    
    return {
        "python处理结果": "处理完成",
        "go调用结果": go_result
    }

def python_file_process_func(params):
    """处理文件并返回修改后的文件信息（注：content字段为Base64编码字符串，接收方需解码）"""
    print(f"Python 接收到文件处理请求，参数：{params}")
    files = params.get("files", [])
    if not files:
        return None, "未接收到文件数据"

    try:
        # 处理第一个文件（假设单次传一个）
        file = files[0]
        meta = file["meta"]
        content_str = file["content"]  # 接收到的Base64字符串
        
        # 解码为原始字节（验证输入合法性）
        content = base64.b64decode(content_str)  # 若content_str非法，此处会抛出异常
        
        # 保存处理后的文件
        original_name = meta["original_name"]
        new_name = f"processed_{original_name}"
        os.makedirs("./processed_files", exist_ok=True)
        with open(f"./processed_files/{new_name}", "wb") as f:
            f.write(content)
        print(f"文件已保存：{new_name}")

        # 重新编码为Base64字符串（标准编码，接收方需用base64.StdEncoding解码）
        encoded_content = base64.b64encode(content).decode()
        print(f"返回的content（Base64字符串前50字符）：{encoded_content[:50]}")  # 关键调试日志
        
        return {
            "processed_file": {
                "new_name": new_name,
                "mimetype": meta["mimetype"],
                "content": encoded_content  # 明确标注为Base64字符串
            }
        }, None
    except Exception as e:
        return None, f"文件处理失败: {str(e)}"

def call_go_service(method, params, files=None):
    """
    扩展支持文件传输的调用函数
    """
    print(f"Python 调用 Go 服务：method={method}, params={params}, files={len(files) if files else 0}")
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            # 完整逻辑放入 try 块（包括连接、发送、接收、解析）
            s.connect(("127.0.0.1", 9090))
            
            # 1. 序列化基础参数（JSON）
            param_data = json.dumps(params, ensure_ascii=False).encode()
            
            # 2. 生成消息ID（UUID）
            msg_id = str(uuid.uuid4()).encode()
            
            # 3. 初始化协议数据缓冲区
            request = bytearray()
            
            # 4. 写入魔数（2字节，大端）
            request.extend(struct.pack(">H", 0xAEBD))  # 与Go端魔数一致
            
            # 5. 写入版本号（1字节）
            request.append(0x01)  # 当前版本1
            
            # 6. 写入消息ID（长度2字节 + 内容）
            request.extend(struct.pack(">H", len(msg_id)))
            request.extend(msg_id)
            
            # 7. 写入方法名（长度2字节 + 内容）
            method_bytes = method.encode()
            request.extend(struct.pack(">H", len(method_bytes)))
            request.extend(method_bytes)
            
            # 8. 写入参数内容（长度4字节 + 内容）
            request.extend(struct.pack(">I", len(param_data)))
            request.extend(param_data)
            
            # 9. 写入文件数量（2字节，大端）
            file_count = len(files) if files else 0
            request.extend(struct.pack(">H", file_count))
            
            # 10. 写入文件元数据和内容（逐个处理）
            total_data = bytearray()  # 用于计算校验和的完整数据
            total_data.extend(request)  # 基础部分已包含魔数、版本等
            
            for file in files or []:
                # 元数据（长度2字节 + 内容）
                meta_data = json.dumps(file["meta"], ensure_ascii=False).encode()
                request.extend(struct.pack(">H", len(meta_data)))
                request.extend(meta_data)
                total_data.extend(struct.pack(">H", len(meta_data)))
                total_data.extend(meta_data)
                
                # 内容（长度4字节 + 内容）
                content = file["content"]
                request.extend(struct.pack(">I", len(content)))
                request.extend(content)
                total_data.extend(struct.pack(">I", len(content)))
                total_data.extend(content)
            
            # 11. 计算并写入校验和（4字节，大端）
            checksum = zlib.crc32(total_data)
            request.extend(struct.pack(">I", checksum))
            
            # 发送完整请求
            print(f"Python 发送的请求数据长度：{len(request)} bytes")
            s.send(request)
            s.shutdown(socket.SHUT_WR)

            # 接收响应数据（新增：打印完整响应的二进制信息）
            data = []
            while True:
                chunk = s.recv(1024)
                if not chunk:
                    break
                data.append(chunk)
            full_response = b''.join(data)
            print(f"Python 接收到完整响应（二进制）：长度={len(full_response)} bytes，内容（前50字节）={full_response[:50].hex()}")  # 新增日志

            # 解析协议头（新增：打印协议头关键字段）
            if len(full_response) < 7:
                raise Exception("响应数据不完整，协议头缺失")
            
            magic = full_response[0:2]
            print(f"解析魔数：实际值={magic.hex()}，预期值=AEBD（大端序）")  # 新增日志
            
            version = full_response[2]  # 版本号在第3字节（索引2）
            print(f"解析版本号：实际值={version}，预期值=1")  # 新增日志
            
            body_length = int.from_bytes(full_response[3:7], byteorder='big')
            print(f"解析响应体长度：实际值={body_length} bytes")  # 新增日志

            body_data = full_response[7:7+body_length]
            print(f"提取响应体：实际长度={len(body_data)} bytes，预期长度={body_length} bytes")  # 新增日志

            # 解析响应体内容（新增：打印解码后的原始字符串和JSON解析结果）
            response = body_data.decode('utf-8')
            print(f"Python 接收到 Go 响应（字符串）：{response}")  # 原有日志（保留）
            result = json.loads(response)
            print(f"Python 解析 Go 响应（JSON）：{result}")  # 新增日志

            return result, None
        except Exception as e:
            print(f"连接或发送数据失败：{e}")
            return None, str(e)

if __name__ == "__main__":
    # 创建并启动 IPC 服务器
    server = IpcServer(("127.0.0.1", 9091))  # 与 Go 端 callPythonIpcService 连接的地址一致
    
    # 注册服务（新增关键日志）
    server.register("python.service.demo", python_demo_func)
    print("成功注册服务: python.service.demo")  # 新增日志
    server.register("python.service.fileProcess", python_file_process_func)
    print("成功注册服务: python.service.fileProcess")  # 新增日志
    
    server.start()