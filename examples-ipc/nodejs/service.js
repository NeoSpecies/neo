#!/usr/bin/env node
/**
 * Neo Framework IPC Node.js 示例服务
 * 演示如何使用 Node.js 创建一个 IPC 服务
 */

const net = require('net');
const { Buffer } = require('buffer');

// 消息类型定义
const MessageType = {
    REQUEST: 1,
    RESPONSE: 2,
    REGISTER: 3,
    HEARTBEAT: 4
};

class NeoIPCClient {
    constructor(host = 'localhost', port = 9999) {
        this.host = host;
        this.port = port;
        this.client = null;
        this.handlers = new Map();
        this.serviceName = null;
        this.buffer = Buffer.alloc(0);
    }

    connect() {
        return new Promise((resolve, reject) => {
            this.client = new net.Socket();
            
            this.client.connect(this.port, this.host, () => {
                console.log(`Connected to Neo IPC server at ${this.host}:${this.port}`);
                resolve();
            });

            this.client.on('data', (data) => {
                this.buffer = Buffer.concat([this.buffer, data]);
                this.processBuffer();
            });

            this.client.on('error', (error) => {
                console.error('Connection error:', error);
                reject(error);
            });

            this.client.on('close', () => {
                console.log('Connection closed');
            });
        });
    }

    async registerService(serviceName, metadata = {}) {
        this.serviceName = serviceName;
        
        const msg = {
            type: MessageType.REGISTER,
            id: '',
            service: serviceName,
            method: '',
            data: JSON.stringify({
                name: serviceName,
                metadata: metadata
            }),
            metadata: {}
        };

        await this.sendMessage(msg);
        console.log(`Service '${serviceName}' registered`);
    }

    addHandler(method, handler) {
        this.handlers.set(method, handler);
        console.log(`Handler registered for method: ${method}`);
    }

    sendMessage(msg) {
        return new Promise((resolve, reject) => {
            try {
                const buffer = this.encodeMessage(msg);
                this.client.write(buffer, () => resolve());
            } catch (error) {
                reject(error);
            }
        });
    }

    encodeMessage(msg) {
        const chunks = [];
        
        // 消息类型
        chunks.push(Buffer.from([msg.type]));
        
        // ID
        const idBuf = Buffer.from(msg.id, 'utf8');
        const idLenBuf = Buffer.alloc(4);
        idLenBuf.writeUInt32LE(idBuf.length, 0);
        chunks.push(idLenBuf, idBuf);
        
        // Service
        const serviceBuf = Buffer.from(msg.service, 'utf8');
        const serviceLenBuf = Buffer.alloc(4);
        serviceLenBuf.writeUInt32LE(serviceBuf.length, 0);
        chunks.push(serviceLenBuf, serviceBuf);
        
        // Method
        const methodBuf = Buffer.from(msg.method, 'utf8');
        const methodLenBuf = Buffer.alloc(4);
        methodLenBuf.writeUInt32LE(methodBuf.length, 0);
        chunks.push(methodLenBuf, methodBuf);
        
        // Metadata
        const metadataBuf = Buffer.from(JSON.stringify(msg.metadata), 'utf8');
        const metadataLenBuf = Buffer.alloc(4);
        metadataLenBuf.writeUInt32LE(metadataBuf.length, 0);
        chunks.push(metadataLenBuf, metadataBuf);
        
        // Data
        const dataBuf = Buffer.from(msg.data, 'utf8');
        const dataLenBuf = Buffer.alloc(4);
        dataLenBuf.writeUInt32LE(dataBuf.length, 0);
        chunks.push(dataLenBuf, dataBuf);
        
        // 合并所有内容
        const content = Buffer.concat(chunks);
        
        // 添加总长度前缀
        const lengthBuf = Buffer.alloc(4);
        lengthBuf.writeUInt32LE(content.length, 0);
        
        return Buffer.concat([lengthBuf, content]);
    }

    processBuffer() {
        while (this.buffer.length >= 4) {
            // 读取消息长度
            const msgLen = this.buffer.readUInt32LE(0);
            
            // 检查是否有完整消息
            if (this.buffer.length < 4 + msgLen) {
                break;
            }
            
            // 提取消息
            const msgData = this.buffer.slice(4, 4 + msgLen);
            this.buffer = this.buffer.slice(4 + msgLen);
            
            // 解码并处理消息
            const msg = this.decodeMessage(msgData);
            if (msg.type === MessageType.REQUEST) {
                this.handleRequest(msg);
            }
        }
    }

    decodeMessage(buffer) {
        let offset = 0;
        
        // 消息类型
        const type = buffer[offset];
        offset += 1;
        
        // ID
        const idLen = buffer.readUInt32LE(offset);
        offset += 4;
        const id = buffer.toString('utf8', offset, offset + idLen);
        offset += idLen;
        
        // Service
        const serviceLen = buffer.readUInt32LE(offset);
        offset += 4;
        const service = buffer.toString('utf8', offset, offset + serviceLen);
        offset += serviceLen;
        
        // Method
        const methodLen = buffer.readUInt32LE(offset);
        offset += 4;
        const method = buffer.toString('utf8', offset, offset + methodLen);
        offset += methodLen;
        
        // Metadata
        const metadataLen = buffer.readUInt32LE(offset);
        offset += 4;
        const metadataStr = buffer.toString('utf8', offset, offset + metadataLen);
        const metadata = metadataStr ? JSON.parse(metadataStr) : {};
        offset += metadataLen;
        
        // Data
        const dataLen = buffer.readUInt32LE(offset);
        offset += 4;
        const data = buffer.toString('utf8', offset, offset + dataLen);
        
        return { type, id, service, method, metadata, data };
    }

    async handleRequest(msg) {
        const handler = this.handlers.get(msg.method);
        
        if (!handler) {
            const errorResp = {
                type: MessageType.RESPONSE,
                id: msg.id,
                service: msg.service,
                method: msg.method,
                data: JSON.stringify({ error: `Method '${msg.method}' not found` }),
                metadata: { error: 'true' }
            };
            await this.sendMessage(errorResp);
            return;
        }

        try {
            // 解析请求参数
            const params = msg.data ? JSON.parse(msg.data) : {};
            
            // 调用处理器
            const result = await handler(params);
            
            // 发送响应
            const response = {
                type: MessageType.RESPONSE,
                id: msg.id,
                service: msg.service,
                method: msg.method,
                data: JSON.stringify(result),
                metadata: {}
            };
            await this.sendMessage(response);
            
        } catch (error) {
            const errorResp = {
                type: MessageType.RESPONSE,
                id: msg.id,
                service: msg.service,
                method: msg.method,
                data: JSON.stringify({ error: error.message }),
                metadata: { error: 'true' }
            };
            await this.sendMessage(errorResp);
        }
    }

    startHeartbeat() {
        setInterval(async () => {
            try {
                const msg = {
                    type: MessageType.HEARTBEAT,
                    id: '',
                    service: this.serviceName,
                    method: '',
                    data: '',
                    metadata: {}
                };
                await this.sendMessage(msg);
                console.log('Heartbeat sent');
            } catch (error) {
                console.error('Heartbeat error:', error);
            }
        }, 30000);
    }

    run() {
        this.startHeartbeat();
        console.log('Service is running...');
    }
}

// 主函数
async function main() {
    // 从环境变量读取配置
    const host = process.env.NEO_IPC_HOST || 'localhost';
    const port = parseInt(process.env.NEO_IPC_PORT || '9999');  // 使用正确的默认端口

    // 创建客户端
    const client = new NeoIPCClient(host, port);

    // 注册处理器
    client.addHandler('hello', async (params) => {
        const name = params.name || 'World';
        return {
            message: `Hello, ${name}!`,
            timestamp: new Date().toISOString(),
            service: 'Node.js Demo Service'
        };
    });

    client.addHandler('calculate', async (params) => {
        const a = params.a || 0;
        const b = params.b || 0;
        const operation = params.operation || 'add';

        let result;
        switch (operation) {
            case 'add':
                result = a + b;
                break;
            case 'subtract':
                result = a - b;
                break;
            case 'multiply':
                result = a * b;
                break;
            case 'divide':
                result = b !== 0 ? a / b : 'Cannot divide by zero';
                break;
            default:
                result = 'Unknown operation';
        }

        return {
            result,
            operation,
            a,
            b
        };
    });

    client.addHandler('echo', async (params) => {
        const message = params.message || '';
        return {
            echo: message,
            length: message.length,
            reversed: message.split('').reverse().join('')
        };
    });

    client.addHandler('getTime', async (params) => {
        const format = params.format || 'iso';
        const now = new Date();

        let timeStr;
        switch (format) {
            case 'unix':
                timeStr = Math.floor(now.getTime() / 1000).toString();
                break;
            case 'readable':
                timeStr = now.toLocaleString();
                break;
            default:
                timeStr = now.toISOString();
        }

        return {
            time: timeStr,
            timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            format
        };
    });

    client.addHandler('getInfo', async () => {
        return {
            service: 'demo-service-nodejs',
            language: 'Node.js',
            version: '1.0.0',
            handlers: Array.from(client.handlers.keys()),
            uptime: 'N/A',
            system: {
                platform: process.platform,
                node_version: process.version
            }
        };
    });

    try {
        // 连接并注册服务
        await client.connect();
        await client.registerService('demo-service-nodejs', {
            language: 'nodejs',
            version: '1.0.0',
            description: 'Node.js demo service for Neo Framework'
        });

        console.log('Node.js demo service is running...');
        console.log(`Listening on ${host}:${port}`);
        console.log('Available methods: hello, calculate, echo, getTime, getInfo');

        // 运行服务
        client.run();

        // 保持进程运行
        process.on('SIGINT', () => {
            console.log('\nShutting down...');
            process.exit(0);
        });

    } catch (error) {
        console.error('Service error:', error);
        process.exit(1);
    }
}

// 启动服务
if (require.main === module) {
    main();
}