# Neo Framework IPC 协议对接指南

## 目录
1. [概述](#概述)
2. [通信原理](#通信原理)
3. [协议规范](#协议规范)
4. [服务注册流程](#服务注册流程)
5. [请求响应机制](#请求响应机制)
6. [心跳机制](#心跳机制)
7. [错误处理](#错误处理)
8. [多语言客户端实现](#多语言客户端实现)
9. [最佳实践](#最佳实践)
10. [调试指南](#调试指南)

## 概述

Neo Framework 采用自定义的二进制 IPC（进程间通信）协议，支持多语言服务接入。该协议具有以下特点：

- **高性能**：二进制格式，减少序列化开销
- **跨语言**：支持任何能够进行 TCP 通信的编程语言
- **简单可靠**：固定的消息格式，易于实现和调试
- **异步支持**：支持异步请求响应模式

### 核心组件

1. **IPC Server**：运行在 Neo Framework 内部，监听 TCP 连接
2. **Service Registry**：服务注册中心，管理所有注册的服务
3. **Client Library**：各语言的客户端库，处理协议细节

## 通信原理

### 架构图

```
┌─────────────────┐     TCP/Binary Protocol    ┌──────────────────┐
│                 │◄────────────────────────────►│                  │
│  Neo Framework  │                              │  Your Service    │
│   (IPC Server)  │     1. Register Service      │  (IPC Client)    │
│                 │◄─────────────────────────────│                  │
│                 │                              │                  │
│                 │     2. Send Heartbeat        │                  │
│                 │◄─────────────────────────────│                  │
│                 │                              │                  │
│  HTTP Gateway   │     3. Forward Request       │                  │
│       ↓        │─────────────────────────────►│                  │
│                 │                              │                  │
│                 │     4. Return Response       │                  │
│                 │◄─────────────────────────────│                  │
└─────────────────┘                             └──────────────────┘
```

### 连接流程

1. **建立连接**：客户端通过 TCP 连接到 IPC Server（默认端口 9999）
2. **服务注册**：发送 REGISTER 消息，声明服务名称和元数据
3. **保持连接**：通过心跳消息维持连接活性
4. **处理请求**：接收并处理来自框架的请求，返回响应

## 协议规范

### 消息格式

所有消息采用二进制格式，结构如下：

```
[消息长度:4字节][消息内容:N字节]
```

#### 消息长度（4字节）
- **字节序**：小端序（Little Endian）
- **类型**：uint32
- **说明**：不包含长度字段本身，仅为消息内容的长度

#### 消息内容格式

```
[类型:1字节][ID长度:4字节][ID:N字节][服务名长度:4字节][服务名:N字节]
[方法名长度:4字节][方法名:N字节][元数据长度:4字节][元数据:N字节]
[数据长度:4字节][数据:N字节]
```

### 字段说明

| 字段 | 长度 | 类型 | 说明 |
|------|------|------|------|
| 消息类型 | 1字节 | uint8 | 1=REQUEST, 2=RESPONSE, 3=REGISTER, 4=HEARTBEAT |
| ID长度 | 4字节 | uint32 | 消息ID的字节长度 |
| ID | 变长 | string | 唯一标识符，用于匹配请求和响应 |
| 服务名长度 | 4字节 | uint32 | 服务名的字节长度 |
| 服务名 | 变长 | string | 注册的服务名称 |
| 方法名长度 | 4字节 | uint32 | 方法名的字节长度 |
| 方法名 | 变长 | string | 要调用的方法名称 |
| 元数据长度 | 4字节 | uint32 | 元数据JSON的字节长度 |
| 元数据 | 变长 | JSON | 键值对形式的元数据 |
| 数据长度 | 4字节 | uint32 | 业务数据的字节长度 |
| 数据 | 变长 | bytes | 实际的业务数据（通常为JSON） |

### 消息类型

1. **REQUEST (1)**：框架向服务发送的请求
2. **RESPONSE (2)**：服务返回给框架的响应
3. **REGISTER (3)**：服务注册消息
4. **HEARTBEAT (4)**：心跳消息

## 服务注册流程

### 1. 连接到 IPC Server

```
默认地址：localhost:9999
可通过配置文件修改
```

### 2. 发送注册消息

注册消息格式：
- **消息类型**：3 (REGISTER)
- **ID**：空字符串
- **服务名**：你的服务名称（如 "user-service"）
- **方法名**：空字符串
- **元数据**：空 JSON 对象 {}
- **数据**：JSON 格式的注册信息

注册数据结构：
```json
{
  "name": "your-service-name",
  "metadata": {
    "version": "1.0.0",
    "protocol": "http",
    "language": "python"
  }
}
```

### 3. 注册成功标志

服务端会将该连接与服务名关联，后续该服务的所有请求都会通过这个连接转发。

## 请求响应机制

### 接收请求

当框架需要调用你的服务时，会通过已建立的连接发送 REQUEST 消息：

```
消息类型: 1 (REQUEST)
ID: 唯一请求ID（需要在响应中返回相同ID）
服务名: 你注册的服务名
方法名: 要调用的方法
元数据: 请求相关的元数据
数据: 请求参数（JSON格式）
```

### 返回响应

处理完请求后，需要返回 RESPONSE 消息：

```
消息类型: 2 (RESPONSE)
ID: 与请求相同的ID（重要！）
服务名: 你的服务名
方法名: 被调用的方法名
元数据: 响应相关的元数据（如错误标记）
数据: 响应结果（JSON格式）
```

### 错误响应

如果处理出错，在元数据中设置错误标记：

```json
{
  "metadata": {
    "error": "true"
  },
  "data": {
    "error": "错误信息描述"
  }
}
```

## 心跳机制

### 心跳消息格式

```
消息类型: 4 (HEARTBEAT)
ID: 空字符串
服务名: 你的服务名
方法名: 空字符串
元数据: {}
数据: 空
```

### 心跳策略

- **发送间隔**：建议每 10-30 秒发送一次
- **超时时间**：默认 5 分钟无心跳视为服务不可用
- **自动重连**：连接断开后应自动重连并重新注册

## 错误处理

### 常见错误码

| 错误类型 | 描述 | 处理建议 |
|---------|------|----------|
| 连接错误 | 无法连接到 IPC Server | 检查端口和网络，实现重连机制 |
| 消息过大 | 消息超过 10MB 限制 | 拆分大消息或使用流式传输 |
| 格式错误 | 消息格式不符合协议 | 检查字节序和字段顺序 |
| 服务未找到 | 请求的服务未注册 | 确保服务已成功注册 |

### 错误恢复

1. **自动重连**：连接断开后自动重试
2. **指数退避**：重连失败时增加等待时间
3. **健康检查**：定期检查连接状态

## 多语言客户端实现

### Python 客户端示例

```python
import asyncio
import json
import struct
import logging
from typing import Dict, Any, Callable
from enum import IntEnum

class MessageType(IntEnum):
    REQUEST = 1
    RESPONSE = 2
    REGISTER = 3
    HEARTBEAT = 4

class NeoIPCClient:
    def __init__(self, host: str = "localhost", port: int = 9999):
        self.host = host
        self.port = port
        self.reader = None
        self.writer = None
        self.handlers = {}
        self.service_name = None
        
    async def connect(self):
        """建立连接"""
        self.reader, self.writer = await asyncio.open_connection(
            self.host, self.port
        )
        
    async def register_service(self, service_name: str, metadata: Dict = None):
        """注册服务"""
        self.service_name = service_name
        metadata = metadata or {}
        
        # 构建注册数据
        register_data = {
            "name": service_name,
            "metadata": metadata
        }
        
        # 发送注册消息
        await self._send_message(
            msg_type=MessageType.REGISTER,
            msg_id="",
            service=service_name,
            method="",
            metadata={},
            data=json.dumps(register_data).encode()
        )
        
    def add_handler(self, method: str, handler: Callable):
        """添加方法处理器"""
        self.handlers[method] = handler
        
    async def _send_message(self, msg_type, msg_id, service, method, metadata, data):
        """发送消息"""
        # 序列化元数据
        metadata_json = json.dumps(metadata).encode()
        
        # 构建消息内容
        content = bytearray()
        
        # 消息类型
        content.extend(struct.pack('<B', msg_type))
        
        # ID
        id_bytes = msg_id.encode()
        content.extend(struct.pack('<I', len(id_bytes)))
        content.extend(id_bytes)
        
        # 服务名
        service_bytes = service.encode()
        content.extend(struct.pack('<I', len(service_bytes)))
        content.extend(service_bytes)
        
        # 方法名
        method_bytes = method.encode()
        content.extend(struct.pack('<I', len(method_bytes)))
        content.extend(method_bytes)
        
        # 元数据
        content.extend(struct.pack('<I', len(metadata_json)))
        content.extend(metadata_json)
        
        # 数据
        content.extend(struct.pack('<I', len(data)))
        content.extend(data)
        
        # 发送消息长度和内容
        self.writer.write(struct.pack('<I', len(content)))
        self.writer.write(content)
        await self.writer.drain()
        
    async def start(self):
        """启动服务"""
        await self.connect()
        await self.register_service(self.service_name)
        
        # 启动心跳
        asyncio.create_task(self._heartbeat_loop())
        
        # 处理消息
        await self._message_loop()
        
    async def _heartbeat_loop(self):
        """心跳循环"""
        while True:
            await asyncio.sleep(30)
            await self._send_message(
                msg_type=MessageType.HEARTBEAT,
                msg_id="",
                service=self.service_name,
                method="",
                metadata={},
                data=b""
            )

# 使用示例
async def main():
    client = NeoIPCClient()
    
    # 添加处理器
    @client.add_handler("hello")
    async def hello_handler(params):
        name = params.get("name", "World")
        return {"message": f"Hello, {name}!"}
    
    # 启动服务
    await client.start()

if __name__ == "__main__":
    asyncio.run(main())
```

### Go 客户端示例

```go
package main

import (
    "encoding/binary"
    "encoding/json"
    "fmt"
    "net"
    "time"
)

type MessageType byte

const (
    REQUEST   MessageType = 1
    RESPONSE  MessageType = 2
    REGISTER  MessageType = 3
    HEARTBEAT MessageType = 4
)

type NeoIPCClient struct {
    conn        net.Conn
    serviceName string
    handlers    map[string]func([]byte) ([]byte, error)
}

func NewNeoIPCClient(addr string) (*NeoIPCClient, error) {
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        return nil, err
    }
    
    return &NeoIPCClient{
        conn:     conn,
        handlers: make(map[string]func([]byte) ([]byte, error)),
    }, nil
}

func (c *NeoIPCClient) RegisterService(name string, metadata map[string]string) error {
    c.serviceName = name
    
    // 构建注册数据
    registerData := map[string]interface{}{
        "name":     name,
        "metadata": metadata,
    }
    
    data, _ := json.Marshal(registerData)
    
    return c.sendMessage(REGISTER, "", name, "", nil, data)
}

func (c *NeoIPCClient) sendMessage(msgType MessageType, id, service, method string, 
    metadata map[string]string, data []byte) error {
    
    buf := &bytes.Buffer{}
    
    // 消息类型
    buf.WriteByte(byte(msgType))
    
    // ID
    binary.Write(buf, binary.LittleEndian, uint32(len(id)))
    buf.WriteString(id)
    
    // 服务名
    binary.Write(buf, binary.LittleEndian, uint32(len(service)))
    buf.WriteString(service)
    
    // 方法名
    binary.Write(buf, binary.LittleEndian, uint32(len(method)))
    buf.WriteString(method)
    
    // 元数据
    metadataJSON, _ := json.Marshal(metadata)
    binary.Write(buf, binary.LittleEndian, uint32(len(metadataJSON)))
    buf.Write(metadataJSON)
    
    // 数据
    binary.Write(buf, binary.LittleEndian, uint32(len(data)))
    buf.Write(data)
    
    // 发送消息长度
    content := buf.Bytes()
    binary.Write(c.conn, binary.LittleEndian, uint32(len(content)))
    
    // 发送消息内容
    _, err := c.conn.Write(content)
    return err
}

func (c *NeoIPCClient) Start() error {
    // 启动心跳
    go c.heartbeatLoop()
    
    // 处理消息
    return c.messageLoop()
}

func main() {
    client, err := NewNeoIPCClient("localhost:9999")
    if err != nil {
        panic(err)
    }
    
    // 注册服务
    err = client.RegisterService("my-service", map[string]string{
        "version": "1.0.0",
    })
    if err != nil {
        panic(err)
    }
    
    // 添加处理器
    client.AddHandler("hello", func(data []byte) ([]byte, error) {
        var params map[string]interface{}
        json.Unmarshal(data, &params)
        
        response := map[string]string{
            "message": fmt.Sprintf("Hello, %s!", params["name"]),
        }
        
        return json.Marshal(response)
    })
    
    // 启动服务
    client.Start()
}
```

### Java 客户端示例

```java
import java.io.*;
import java.net.Socket;
import java.nio.ByteBuffer;
import java.nio.ByteOrder;
import java.util.HashMap;
import java.util.Map;
import com.google.gson.Gson;

public class NeoIPCClient {
    private static final byte REQUEST = 1;
    private static final byte RESPONSE = 2;
    private static final byte REGISTER = 3;
    private static final byte HEARTBEAT = 4;
    
    private Socket socket;
    private DataInputStream input;
    private DataOutputStream output;
    private String serviceName;
    private Map<String, Handler> handlers = new HashMap<>();
    private Gson gson = new Gson();
    
    public interface Handler {
        Object handle(Map<String, Object> params) throws Exception;
    }
    
    public NeoIPCClient(String host, int port) throws IOException {
        this.socket = new Socket(host, port);
        this.input = new DataInputStream(socket.getInputStream());
        this.output = new DataOutputStream(socket.getOutputStream());
    }
    
    public void registerService(String name, Map<String, String> metadata) throws IOException {
        this.serviceName = name;
        
        Map<String, Object> registerData = new HashMap<>();
        registerData.put("name", name);
        registerData.put("metadata", metadata != null ? metadata : new HashMap<>());
        
        byte[] data = gson.toJson(registerData).getBytes();
        sendMessage(REGISTER, "", name, "", new HashMap<>(), data);
    }
    
    public void addHandler(String method, Handler handler) {
        handlers.put(method, handler);
    }
    
    private void sendMessage(byte type, String id, String service, String method,
                           Map<String, String> metadata, byte[] data) throws IOException {
        ByteArrayOutputStream content = new ByteArrayOutputStream();
        
        // 消息类型
        content.write(type);
        
        // ID
        byte[] idBytes = id.getBytes();
        writeInt(content, idBytes.length);
        content.write(idBytes);
        
        // 服务名
        byte[] serviceBytes = service.getBytes();
        writeInt(content, serviceBytes.length);
        content.write(serviceBytes);
        
        // 方法名
        byte[] methodBytes = method.getBytes();
        writeInt(content, methodBytes.length);
        content.write(methodBytes);
        
        // 元数据
        byte[] metadataBytes = gson.toJson(metadata).getBytes();
        writeInt(content, metadataBytes.length);
        content.write(metadataBytes);
        
        // 数据
        writeInt(content, data.length);
        content.write(data);
        
        // 发送消息
        byte[] contentBytes = content.toByteArray();
        writeInt(output, contentBytes.length);
        output.write(contentBytes);
        output.flush();
    }
    
    private void writeInt(OutputStream out, int value) throws IOException {
        ByteBuffer buffer = ByteBuffer.allocate(4);
        buffer.order(ByteOrder.LITTLE_ENDIAN);
        buffer.putInt(value);
        out.write(buffer.array());
    }
    
    public void start() throws IOException {
        // 启动心跳线程
        new Thread(this::heartbeatLoop).start();
        
        // 消息处理循环
        messageLoop();
    }
    
    private void heartbeatLoop() {
        while (!socket.isClosed()) {
            try {
                Thread.sleep(30000); // 30秒
                sendMessage(HEARTBEAT, "", serviceName, "", new HashMap<>(), new byte[0]);
            } catch (Exception e) {
                e.printStackTrace();
            }
        }
    }
    
    public static void main(String[] args) throws Exception {
        NeoIPCClient client = new NeoIPCClient("localhost", 9999);
        
        // 注册服务
        Map<String, String> metadata = new HashMap<>();
        metadata.put("version", "1.0.0");
        client.registerService("java-service", metadata);
        
        // 添加处理器
        client.addHandler("hello", params -> {
            String name = (String) params.getOrDefault("name", "World");
            Map<String, String> response = new HashMap<>();
            response.put("message", "Hello, " + name + "!");
            return response;
        });
        
        // 启动服务
        client.start();
    }
}
```

### Node.js 客户端示例

```javascript
const net = require('net');
const EventEmitter = require('events');

const MessageType = {
    REQUEST: 1,
    RESPONSE: 2,
    REGISTER: 3,
    HEARTBEAT: 4
};

class NeoIPCClient extends EventEmitter {
    constructor(host = 'localhost', port = 9999) {
        super();
        this.host = host;
        this.port = port;
        this.handlers = new Map();
        this.serviceName = null;
        this.socket = null;
    }
    
    connect() {
        return new Promise((resolve, reject) => {
            this.socket = net.createConnection(this.port, this.host, () => {
                console.log(`Connected to Neo IPC server at ${this.host}:${this.port}`);
                resolve();
            });
            
            this.socket.on('error', reject);
            this.socket.on('data', data => this._handleData(data));
        });
    }
    
    async registerService(serviceName, metadata = {}) {
        this.serviceName = serviceName;
        
        const registerData = {
            name: serviceName,
            metadata: metadata
        };
        
        await this._sendMessage({
            type: MessageType.REGISTER,
            id: '',
            service: serviceName,
            method: '',
            metadata: {},
            data: Buffer.from(JSON.stringify(registerData))
        });
        
        console.log(`Service '${serviceName}' registered`);
    }
    
    addHandler(method, handler) {
        this.handlers.set(method, handler);
    }
    
    async _sendMessage({type, id, service, method, metadata, data}) {
        const content = Buffer.alloc(1024 * 1024); // 1MB buffer
        let offset = 0;
        
        // 消息类型
        content.writeUInt8(type, offset);
        offset += 1;
        
        // ID
        const idBuffer = Buffer.from(id);
        content.writeUInt32LE(idBuffer.length, offset);
        offset += 4;
        idBuffer.copy(content, offset);
        offset += idBuffer.length;
        
        // 服务名
        const serviceBuffer = Buffer.from(service);
        content.writeUInt32LE(serviceBuffer.length, offset);
        offset += 4;
        serviceBuffer.copy(content, offset);
        offset += serviceBuffer.length;
        
        // 方法名
        const methodBuffer = Buffer.from(method);
        content.writeUInt32LE(methodBuffer.length, offset);
        offset += 4;
        methodBuffer.copy(content, offset);
        offset += methodBuffer.length;
        
        // 元数据
        const metadataBuffer = Buffer.from(JSON.stringify(metadata));
        content.writeUInt32LE(metadataBuffer.length, offset);
        offset += 4;
        metadataBuffer.copy(content, offset);
        offset += metadataBuffer.length;
        
        // 数据
        content.writeUInt32LE(data.length, offset);
        offset += 4;
        data.copy(content, offset);
        offset += data.length;
        
        // 发送消息
        const message = content.slice(0, offset);
        const lengthBuffer = Buffer.alloc(4);
        lengthBuffer.writeUInt32LE(message.length);
        
        this.socket.write(Buffer.concat([lengthBuffer, message]));
    }
    
    async start() {
        await this.connect();
        await this.registerService(this.serviceName);
        
        // 启动心跳
        setInterval(() => {
            this._sendMessage({
                type: MessageType.HEARTBEAT,
                id: '',
                service: this.serviceName,
                method: '',
                metadata: {},
                data: Buffer.alloc(0)
            });
        }, 30000);
    }
}

// 使用示例
async function main() {
    const client = new NeoIPCClient();
    
    // 添加处理器
    client.addHandler('hello', async (params) => {
        const name = params.name || 'World';
        return { message: `Hello, ${name}!` };
    });
    
    // 注册服务
    await client.registerService('nodejs-service', {
        version: '1.0.0'
    });
    
    // 启动服务
    await client.start();
}

main().catch(console.error);
```

### PHP 客户端示例

```php
<?php

class MessageType {
    const REQUEST = 1;
    const RESPONSE = 2;
    const REGISTER = 3;
    const HEARTBEAT = 4;
}

class NeoIPCClient {
    private $host;
    private $port;
    private $socket;
    private $serviceName;
    private $handlers = [];
    
    public function __construct($host = 'localhost', $port = 9999) {
        $this->host = $host;
        $this->port = $port;
    }
    
    public function connect() {
        $this->socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
        if (!$this->socket) {
            throw new Exception("Failed to create socket");
        }
        
        if (!socket_connect($this->socket, $this->host, $this->port)) {
            throw new Exception("Failed to connect to {$this->host}:{$this->port}");
        }
        
        echo "Connected to Neo IPC server at {$this->host}:{$this->port}\n";
    }
    
    public function registerService($serviceName, $metadata = []) {
        $this->serviceName = $serviceName;
        
        $registerData = [
            'name' => $serviceName,
            'metadata' => $metadata
        ];
        
        $this->sendMessage(
            MessageType::REGISTER,
            '',
            $serviceName,
            '',
            [],
            json_encode($registerData)
        );
        
        echo "Service '{$serviceName}' registered\n";
    }
    
    public function addHandler($method, $handler) {
        $this->handlers[$method] = $handler;
    }
    
    private function sendMessage($type, $id, $service, $method, $metadata, $data) {
        $content = '';
        
        // 消息类型
        $content .= pack('C', $type);
        
        // ID
        $idBytes = $id;
        $content .= pack('V', strlen($idBytes));
        $content .= $idBytes;
        
        // 服务名
        $serviceBytes = $service;
        $content .= pack('V', strlen($serviceBytes));
        $content .= $serviceBytes;
        
        // 方法名
        $methodBytes = $method;
        $content .= pack('V', strlen($methodBytes));
        $content .= $methodBytes;
        
        // 元数据
        $metadataJson = json_encode($metadata);
        $content .= pack('V', strlen($metadataJson));
        $content .= $metadataJson;
        
        // 数据
        $content .= pack('V', strlen($data));
        $content .= $data;
        
        // 发送消息长度和内容
        $message = pack('V', strlen($content)) . $content;
        socket_write($this->socket, $message, strlen($message));
    }
    
    private function readMessage() {
        // 读取消息长度
        $lengthData = socket_read($this->socket, 4);
        if (!$lengthData || strlen($lengthData) < 4) {
            return null;
        }
        
        $length = unpack('V', $lengthData)[1];
        
        // 读取消息内容
        $content = socket_read($this->socket, $length);
        $offset = 0;
        
        // 解析消息类型
        $type = unpack('C', substr($content, $offset, 1))[1];
        $offset += 1;
        
        // 解析ID
        $idLength = unpack('V', substr($content, $offset, 4))[1];
        $offset += 4;
        $id = substr($content, $offset, $idLength);
        $offset += $idLength;
        
        // 解析服务名
        $serviceLength = unpack('V', substr($content, $offset, 4))[1];
        $offset += 4;
        $service = substr($content, $offset, $serviceLength);
        $offset += $serviceLength;
        
        // 解析方法名
        $methodLength = unpack('V', substr($content, $offset, 4))[1];
        $offset += 4;
        $method = substr($content, $offset, $methodLength);
        $offset += $methodLength;
        
        // 解析元数据
        $metadataLength = unpack('V', substr($content, $offset, 4))[1];
        $offset += 4;
        $metadataJson = substr($content, $offset, $metadataLength);
        $metadata = json_decode($metadataJson, true);
        $offset += $metadataLength;
        
        // 解析数据
        $dataLength = unpack('V', substr($content, $offset, 4))[1];
        $offset += 4;
        $data = substr($content, $offset, $dataLength);
        
        return [
            'type' => $type,
            'id' => $id,
            'service' => $service,
            'method' => $method,
            'metadata' => $metadata,
            'data' => $data
        ];
    }
    
    public function start() {
        $this->connect();
        
        // 启动心跳
        $this->startHeartbeat();
        
        // 消息处理循环
        $this->messageLoop();
    }
    
    private function messageLoop() {
        while (true) {
            $message = $this->readMessage();
            if (!$message) {
                break;
            }
            
            if ($message['type'] === MessageType::REQUEST) {
                $this->handleRequest($message);
            }
        }
    }
    
    private function handleRequest($message) {
        $method = $message['method'];
        
        if (!isset($this->handlers[$method])) {
            $this->sendMessage(
                MessageType::RESPONSE,
                $message['id'],
                $message['service'],
                $method,
                ['error' => 'true'],
                json_encode(['error' => "Method '{$method}' not found"])
            );
            return;
        }
        
        try {
            $params = json_decode($message['data'], true);
            $handler = $this->handlers[$method];
            $result = $handler($params);
            
            $this->sendMessage(
                MessageType::RESPONSE,
                $message['id'],
                $message['service'],
                $method,
                [],
                json_encode($result)
            );
        } catch (Exception $e) {
            $this->sendMessage(
                MessageType::RESPONSE,
                $message['id'],
                $message['service'],
                $method,
                ['error' => 'true'],
                json_encode(['error' => $e->getMessage()])
            );
        }
    }
}

// 使用示例
$client = new NeoIPCClient();

// 添加处理器
$client->addHandler('hello', function($params) {
    $name = isset($params['name']) ? $params['name'] : 'World';
    return ['message' => "Hello, {$name}!"];
});

// 注册服务
$client->registerService('php-service', [
    'version' => '1.0.0'
]);

// 启动服务
$client->start();
```

## 最佳实践

### 1. 连接管理

- **自动重连**：实现断线重连机制
- **连接池**：对于高并发场景，考虑使用连接池
- **超时处理**：设置合理的读写超时

### 2. 错误处理

- **优雅降级**：服务不可用时的降级策略
- **错误日志**：记录详细的错误信息
- **监控告警**：关键错误触发告警

### 3. 性能优化

- **批量处理**：合并多个小请求
- **异步处理**：使用异步 I/O 提高吞吐量
- **缓存策略**：缓存频繁访问的数据

### 4. 安全考虑

- **认证授权**：在元数据中传递认证信息
- **数据加密**：敏感数据加密传输
- **访问控制**：限制服务访问权限

## 调试指南

### 1. 连接调试

```bash
# 检查端口是否监听
netstat -an | grep 9999

# 测试连接
telnet localhost 9999
```

### 2. 消息调试

启用详细日志，打印收发的原始字节：

```python
# Python 示例
import binascii

def debug_message(data):
    print(f"Message hex: {binascii.hexlify(data)}")
    print(f"Message length: {len(data)}")
```

### 3. 常见问题

**Q: 连接成功但服务未注册**
- 检查注册消息格式是否正确
- 确认服务名不包含特殊字符
- 查看服务端日志

**Q: 请求无响应**
- 确认消息 ID 在响应中保持一致
- 检查是否正确处理了请求
- 验证响应消息格式

**Q: 频繁断线**
- 实现心跳机制
- 检查网络稳定性
- 调整超时参数

## 总结

Neo Framework 的 IPC 协议设计简洁高效，通过标准的 TCP 连接和二进制协议，支持多语言服务的接入。遵循本指南的规范和示例，可以快速实现稳定可靠的服务集成。

如有问题，请参考项目源码或提交 Issue。