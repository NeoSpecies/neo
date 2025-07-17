#!/usr/bin/env php
<?php
/**
 * Neo Framework IPC PHP 示例服务
 * 演示如何使用 PHP 创建一个 IPC 服务
 */

// 消息类型定义
define('MESSAGE_REQUEST', 1);
define('MESSAGE_RESPONSE', 2);
define('MESSAGE_REGISTER', 3);
define('MESSAGE_HEARTBEAT', 4);

class NeoIPCClient {
    private $host;
    private $port;
    private $socket;
    private $handlers = [];
    private $serviceName;
    
    public function __construct($host = 'localhost', $port = 9999) {
        $this->host = $host;
        $this->port = $port;
    }
    
    public function connect() {
        $this->socket = socket_create(AF_INET, SOCK_STREAM, SOL_TCP);
        if ($this->socket === false) {
            throw new Exception("socket_create() failed: " . socket_strerror(socket_last_error()));
        }
        
        $result = socket_connect($this->socket, $this->host, $this->port);
        if ($result === false) {
            throw new Exception("socket_connect() failed: " . socket_strerror(socket_last_error($this->socket)));
        }
        
        echo "Connected to Neo IPC server at {$this->host}:{$this->port}\n";
    }
    
    public function registerService($serviceName, $metadata = []) {
        $this->serviceName = $serviceName;
        
        $data = json_encode([
            'name' => $serviceName,
            'metadata' => $metadata
        ]);
        
        $this->sendMessage(MESSAGE_REGISTER, '', $serviceName, '', [], $data);
        echo "Service '{$serviceName}' registered\n";
    }
    
    public function addHandler($method, $handler) {
        $this->handlers[$method] = $handler;
        echo "Handler registered for method: {$method}\n";
    }
    
    private function sendMessage($type, $id, $service, $method, $metadata, $data) {
        $content = '';
        
        // 消息类型
        $content .= chr($type);
        
        // ID
        $idBytes = $id;
        $content .= pack('V', strlen($idBytes));
        $content .= $idBytes;
        
        // Service
        $serviceBytes = $service;
        $content .= pack('V', strlen($serviceBytes));
        $content .= $serviceBytes;
        
        // Method
        $methodBytes = $method;
        $content .= pack('V', strlen($methodBytes));
        $content .= $methodBytes;
        
        // Metadata
        $metadataJson = json_encode($metadata);
        $content .= pack('V', strlen($metadataJson));
        $content .= $metadataJson;
        
        // Data
        $content .= pack('V', strlen($data));
        $content .= $data;
        
        // 发送总长度和消息
        $message = pack('V', strlen($content)) . $content;
        socket_write($this->socket, $message, strlen($message));
    }
    
    private function readMessage() {
        // 读取消息长度
        $lenData = socket_read($this->socket, 4);
        if ($lenData === false || strlen($lenData) < 4) {
            return null;
        }
        $msgLen = unpack('V', $lenData)[1];
        
        // 读取消息内容
        $msgData = '';
        $remaining = $msgLen;
        while ($remaining > 0) {
            $chunk = socket_read($this->socket, $remaining);
            if ($chunk === false) {
                return null;
            }
            $msgData .= $chunk;
            $remaining -= strlen($chunk);
        }
        
        // 解析消息
        $offset = 0;
        
        // 消息类型
        $type = ord($msgData[$offset]);
        $offset += 1;
        
        // ID
        $idLen = unpack('V', substr($msgData, $offset, 4))[1];
        $offset += 4;
        $id = substr($msgData, $offset, $idLen);
        $offset += $idLen;
        
        // Service
        $serviceLen = unpack('V', substr($msgData, $offset, 4))[1];
        $offset += 4;
        $service = substr($msgData, $offset, $serviceLen);
        $offset += $serviceLen;
        
        // Method
        $methodLen = unpack('V', substr($msgData, $offset, 4))[1];
        $offset += 4;
        $method = substr($msgData, $offset, $methodLen);
        $offset += $methodLen;
        
        // Metadata
        $metadataLen = unpack('V', substr($msgData, $offset, 4))[1];
        $offset += 4;
        $metadataJson = substr($msgData, $offset, $metadataLen);
        $metadata = $metadataJson ? json_decode($metadataJson, true) : [];
        $offset += $metadataLen;
        
        // Data
        $dataLen = unpack('V', substr($msgData, $offset, 4))[1];
        $offset += 4;
        $data = substr($msgData, $offset, $dataLen);
        
        return [
            'type' => $type,
            'id' => $id,
            'service' => $service,
            'method' => $method,
            'metadata' => $metadata,
            'data' => $data
        ];
    }
    
    private function handleRequest($msg) {
        if (!isset($this->handlers[$msg['method']])) {
            $errorData = json_encode(['error' => "Method '{$msg['method']}' not found"]);
            $this->sendMessage(
                MESSAGE_RESPONSE,
                $msg['id'],
                $msg['service'],
                $msg['method'],
                ['error' => 'true'],
                $errorData
            );
            return;
        }
        
        try {
            // 解析请求参数
            $params = $msg['data'] ? json_decode($msg['data'], true) : [];
            
            // 调用处理器
            $handler = $this->handlers[$msg['method']];
            $result = $handler($params);
            
            // 发送响应
            $responseData = json_encode($result);
            $this->sendMessage(
                MESSAGE_RESPONSE,
                $msg['id'],
                $msg['service'],
                $msg['method'],
                [],
                $responseData
            );
            
        } catch (Exception $e) {
            $errorData = json_encode(['error' => $e->getMessage()]);
            $this->sendMessage(
                MESSAGE_RESPONSE,
                $msg['id'],
                $msg['service'],
                $msg['method'],
                ['error' => 'true'],
                $errorData
            );
        }
    }
    
    private function sendHeartbeat() {
        $this->sendMessage(MESSAGE_HEARTBEAT, '', $this->serviceName, '', [], '');
        echo "Heartbeat sent\n";
    }
    
    public function run() {
        // 设置 socket 为非阻塞模式
        socket_set_nonblock($this->socket);
        
        $lastHeartbeat = time();
        
        echo "Service is running...\n";
        
        while (true) {
            // 检查是否有新消息
            $read = [$this->socket];
            $write = null;
            $except = null;
            
            if (socket_select($read, $write, $except, 0, 100000) > 0) {
                $msg = $this->readMessage();
                if ($msg && $msg['type'] == MESSAGE_REQUEST) {
                    $this->handleRequest($msg);
                }
            }
            
            // 发送心跳
            if (time() - $lastHeartbeat >= 30) {
                $this->sendHeartbeat();
                $lastHeartbeat = time();
            }
            
            // 小延迟避免 CPU 占用过高
            usleep(10000); // 10ms
        }
    }
}

// 主函数
function main() {
    // 从环境变量读取配置
    $host = getenv('NEO_IPC_HOST') ?: 'localhost';
    $port = getenv('NEO_IPC_PORT') ?: 9999;  // 使用正确的默认端口
    
    // 创建客户端
    $client = new NeoIPCClient($host, intval($port));
    
    // 注册处理器
    $client->addHandler('hello', function($params) {
        $name = $params['name'] ?? 'World';
        return [
            'message' => "Hello, {$name}!",
            'timestamp' => date('c'),
            'service' => 'PHP Demo Service'
        ];
    });
    
    $client->addHandler('calculate', function($params) {
        $a = $params['a'] ?? 0;
        $b = $params['b'] ?? 0;
        $operation = $params['operation'] ?? 'add';
        
        switch ($operation) {
            case 'add':
                $result = $a + $b;
                break;
            case 'subtract':
                $result = $a - $b;
                break;
            case 'multiply':
                $result = $a * $b;
                break;
            case 'divide':
                $result = $b != 0 ? $a / $b : 'Cannot divide by zero';
                break;
            default:
                $result = 'Unknown operation';
        }
        
        return [
            'result' => $result,
            'operation' => $operation,
            'a' => $a,
            'b' => $b
        ];
    });
    
    $client->addHandler('echo', function($params) {
        $message = $params['message'] ?? '';
        return [
            'echo' => $message,
            'length' => strlen($message),
            'reversed' => strrev($message)
        ];
    });
    
    $client->addHandler('getTime', function($params) {
        $format = $params['format'] ?? 'iso';
        
        switch ($format) {
            case 'unix':
                $timeStr = time();
                break;
            case 'readable':
                $timeStr = date('Y-m-d H:i:s');
                break;
            default:
                $timeStr = date('c');
        }
        
        return [
            'time' => $timeStr,
            'timezone' => date_default_timezone_get(),
            'format' => $format
        ];
    });
    
    $client->addHandler('getInfo', function() use ($client) {
        return [
            'service' => 'demo-service-php',
            'language' => 'PHP',
            'version' => '1.0.0',
            'handlers' => ['hello', 'calculate', 'echo', 'getTime', 'getInfo'],
            'uptime' => 'N/A',
            'system' => [
                'platform' => PHP_OS,
                'php_version' => PHP_VERSION
            ]
        ];
    });
    
    try {
        // 连接并注册服务
        $client->connect();
        $client->registerService('demo-service-php', [
            'language' => 'php',
            'version' => '1.0.0',
            'description' => 'PHP demo service for Neo Framework'
        ]);
        
        echo "PHP demo service is running...\n";
        echo "Listening on {$host}:{$port}\n";
        echo "Available methods: hello, calculate, echo, getTime, getInfo\n";
        
        // 运行服务
        $client->run();
        
    } catch (Exception $e) {
        echo "Service error: " . $e->getMessage() . "\n";
        exit(1);
    }
}

// 设置信号处理
if (function_exists('pcntl_signal')) {
    pcntl_signal(SIGINT, function() {
        echo "\nShutting down...\n";
        exit(0);
    });
}

// 启动服务
if (php_sapi_name() == 'cli') {
    main();
}