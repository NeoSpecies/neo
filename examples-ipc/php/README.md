# PHP IPC 客户端示例

这是一个 PHP 的 IPC 客户端示例，展示如何连接到 Neo Framework。

## 运行要求

- PHP >= 7.0
- 启用 sockets 扩展

## 快速开始

```bash
# 直接运行
php service.php

# 或添加执行权限后运行（Linux/Mac）
chmod +x service.php
./service.php
```

## 环境变量配置

```bash
# 设置 IPC 服务器地址（默认: localhost）
export NEO_IPC_HOST=localhost

# 设置 IPC 服务器端口（默认: 9999）
export NEO_IPC_PORT=9999
```

## 测试服务

```bash
# 测试 hello
curl -X POST http://localhost:8080/api/demo-service/hello \
  -H "Content-Type: application/json" \
  -d '{"name": "PHP"}'

# 测试计算
curl -X POST http://localhost:8080/api/demo-service/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 100, "b": 25, "operation": "subtract"}'

# 测试回显
curl -X POST http://localhost:8080/api/demo-service/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from PHP!"}'

# 获取时间
curl -X POST http://localhost:8080/api/demo-service/getTime \
  -H "Content-Type: application/json" \
  -d '{"format": "unix"}'

# 获取服务信息
curl -X POST http://localhost:8080/api/demo-service/getInfo
```

## 代码结构

- `service.php` - 主服务文件，包含：
  - `NeoIPCClient` - IPC 客户端类
  - Socket 通信实现
  - 处理器注册机制

## 扩展开发

1. 添加新的处理器：
```php
$client->addHandler('myMethod', function($params) {
    // 你的业务逻辑
    return ['result' => 'success'];
});
```

2. 修改服务名称：
```php
$client->registerService('my-service-name', [
    'version' => '1.0.0'
]);
```

## 依赖检查

确保 PHP 启用了 sockets 扩展：
```bash
php -m | grep sockets
```

如果未启用，请安装或在 php.ini 中启用：
```ini
extension=sockets
```

## 注意事项

- 使用小端序（Little Endian）进行二进制编码
- 心跳间隔为 30 秒
- 使用非阻塞 socket 避免 CPU 占用过高
- 支持 PHP 7.0+ 版本