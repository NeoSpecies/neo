import json
import struct
import uuid
import zlib  # 新增：用于计算CRC32校验和
from server import IpcServer
import socket
import os  # 确保已导入 os（文件处理需要）
import base64  # 将base64导入移至文件顶部，规范代码结构
import io  # 新增：用于构造请求
def python_demo_func(params):
    # print(f"Python 函数接收到参数：{params}")
    
    try:
        # 调用 Go 服务
        go_result, err = call_go_service("go.service.test", {
            "input": params["input"] + " (Python 处理后)"
        })
        if err is not None:
            # print(f"调用 Go 服务失败: {err}")
            return {
                "python处理结果": "处理完成",
                "go调用结果": None,
                "error": err
            }, None
        
        return {
            "python处理结果": "处理完成",
            "go调用结果": go_result
        }, None
    except Exception as e:
        # print(f"Python 服务异常: {str(e)}")
        return {
            "python处理结果": "处理完成",
            "go调用结果": None,
            "error": str(e)
        }, None

def python_file_process_func(params):
    """处理文件并返回修改后的文件信息（注：content字段为Base64编码字符串，接收方需解码）"""
    # print(f"Python 接收到文件处理请求，参数：{params}")
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
        # print(f"文件已保存：{new_name}")

        # 重新编码为Base64字符串（标准编码，接收方需用base64.StdEncoding解码）
        encoded_content = base64.b64encode(content).decode()
        # print(f"返回的content（Base64字符串前50字符）：{encoded_content[:50]}")  # 关键调试日志
        
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
    # print(f"[DEBUG] Python 调用 Go 服务：method={method}, params={params}, files={len(files) if files else 0}")
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        try:
            # 完整逻辑放入 try 块（包括连接、发送、接收、解析）
            # print("[DEBUG] 正在连接 Go 服务...")
            s.connect(("127.0.0.1", 9090))
            # print("[DEBUG] 成功连接到 Go 服务")
            
            # 1. 序列化基础参数（JSON）
            param_data = json.dumps(params, ensure_ascii=False).encode()
            # print(f"[DEBUG] 参数序列化成功，长度: {len(param_data)} bytes")
            
            # 2. 生成消息ID（UUID）
            msg_id = str(uuid.uuid4()).encode()
            # print(f"[DEBUG] 生成消息ID: {msg_id.decode()}")
            
            # 3. 初始化协议数据缓冲区
            request = io.BytesIO()
            total_data = io.BytesIO()
            
            # 4. 写入魔数（2字节，大端）
            magic = struct.pack(">H", 0xAEBD)  # 与Go端魔数一致
            request.write(magic)
            total_data.write(magic)
            # print("[DEBUG] 写入魔数: 0xAEBD")
            
            # 5. 写入版本号（1字节）
            version = bytes([0x01])  # 当前版本1
            request.write(version)
            total_data.write(version)
            # print("[DEBUG] 写入版本号: 1")
            
            # 6. 写入消息ID（长度2字节 + 内容）
            msg_id_len = struct.pack(">H", len(msg_id))
            request.write(msg_id_len)
            request.write(msg_id)
            total_data.write(msg_id_len)
            total_data.write(msg_id)
            
            # print(f"[DEBUG] 写入消息ID: {msg_id.decode()}")
            
            # 7. 写入方法名（长度2字节 + 内容）
            method_bytes = method.encode()
            method_len = struct.pack(">H", len(method_bytes))
            request.write(method_len)
            request.write(method_bytes)
            total_data.write(method_len)
            total_data.write(method_bytes)
            # print(f"[DEBUG] 写入方法名: {method}")
            
            # 8. 写入参数内容（长度4字节 + 内容）
            param_len = struct.pack(">I", len(param_data))
            request.write(param_len)
            request.write(param_data)
            total_data.write(param_len)
            total_data.write(param_data)
            # print(f"[DEBUG] 写入参数内容，长度: {len(param_data)} bytes")
            
            # 9. 写入文件数量（2字节，大端）
            file_count = len(files) if files else 0
            file_count_bytes = struct.pack(">H", file_count)
            request.write(file_count_bytes)
            total_data.write(file_count_bytes)
            # print(f"[DEBUG] 写入文件数量: {file_count}")
            
            # 10. 写入文件元数据和内容（逐个处理）
            for file in files or []:
                # 元数据（长度2字节 + 内容）
                meta_data = json.dumps(file["meta"], ensure_ascii=False).encode()
                meta_len = struct.pack(">H", len(meta_data))
                request.write(meta_len)
                request.write(meta_data)
                total_data.write(meta_len)
                total_data.write(meta_data)
                # print(f"[DEBUG] 写入文件元数据，长度: {len(meta_data)} bytes")
                
                # 内容（长度4字节 + 内容）
                content = file["content"]
                content_len = struct.pack(">I", len(content))
                request.write(content_len)
                request.write(content)
                total_data.write(content_len)
                total_data.write(content)
                # print(f"[DEBUG] 写入文件内容，长度: {len(content)} bytes")
            
            # 11. 计算并写入校验和（4字节，大端）
            total_data_bytes = total_data.getvalue()
            checksum = zlib.crc32(total_data_bytes)
            request.write(struct.pack(">I", checksum))
            # print(f"[DEBUG] 写入校验和: 0x{checksum:08X}")
            # print(f"[DEBUG] 校验和计算数据长度: {len(total_data_bytes)} bytes")
            
            # 发送完整请求
            request_data = request.getvalue()
            # print(f"[DEBUG] 发送请求，总长度: {len(request_data)} bytes")
            s.sendall(request_data)
            s.shutdown(socket.SHUT_WR)
            # print("[DEBUG] 请求发送完成")

            # 接收响应数据
            response_data = bytearray()
            while True:
                chunk = s.recv(1024)
                if not chunk:
                    break
                response_data.extend(chunk)
            
            #print(f"[DEBUG] 接收到响应，长度: {len(response_data)} bytes")
            
            # 解析响应头
            if len(response_data) < 7:
                raise Exception("响应数据不完整，协议头缺失")
            
            # 解析魔数
            magic = struct.unpack(">H", response_data[0:2])[0]
            # print(f"[DEBUG] 解析魔数：实际值={magic:04x}，预期值=AEBD")
            if magic != 0xAEBD:
                raise Exception(f"无效的魔数：{magic:04x}")
            
            # 解析版本号
            version = response_data[2]
            # print(f"[DEBUG] 解析版本号：实际值={version}，预期值=1")
            if version != 0x01:
                raise Exception(f"不支持的版本号：{version}")
            
            # 解析响应体长度
            body_length = struct.unpack(">I", response_data[3:7])[0]
            # print(f"[DEBUG] 解析响应体长度：实际值={body_length} bytes")
            
            # 提取响应体
            if len(response_data) < 7 + body_length:
                raise Exception(f"响应体数据不完整：需要 {body_length} 字节，实际只有 {len(response_data)-7} 字节")
            
            body_data = response_data[7:7+body_length]
            # print(f"[DEBUG] 提取响应体：实际长度={len(body_data)} bytes，预期长度={body_length} bytes")
            
            # 解析响应体
            try:
                response = json.loads(body_data.decode('utf-8'))
                # print(f"[DEBUG] 解析响应体（JSON）：{response}")
                if response.get("error") is not None: 
                    # print(f"[ERROR] Go 服务返回错误: {response['error']}")
                    return None, response["error"]
                # print(f"[DEBUG] 成功获取 Go 服务响应: {response.get('result')}")
                return response.get("result"), None
            except json.JSONDecodeError as e:
                # print(f"[ERROR] 响应体解析失败：{str(e)}")
                # print(f"[DEBUG] 响应体内容：{body_data.decode('utf-8', errors='replace')}")
                raise Exception(f"响应体解析失败：{str(e)}")
        except Exception as e:
            # print(f"[ERROR] 连接或发送数据失败：{e}")
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