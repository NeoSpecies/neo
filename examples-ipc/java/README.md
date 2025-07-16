# Java IPC 客户端示例

这是一个 Java 的 IPC 客户端示例，展示如何连接到 Neo Framework。

## 运行要求

- Java >= 8
- Gson 库 (用于 JSON 处理)

## 快速开始

### 1. 下载 Gson 依赖

```bash
# 下载 Gson (如果没有)
curl -L https://repo1.maven.org/maven2/com/google/code/gson/gson/2.8.9/gson-2.8.9.jar -o gson-2.8.9.jar
```

### 2. 编译和运行

```bash
# 编译
javac -cp ".:gson-2.8.9.jar" Service.java

# 运行 (Linux/Mac)
java -cp ".:gson-2.8.9.jar" Service

# 运行 (Windows)
java -cp ".;gson-2.8.9.jar" Service
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
  -d '{"name": "Java"}'

# 测试计算
curl -X POST http://localhost:8080/api/demo-service/calculate \
  -H "Content-Type: application/json" \
  -d '{"a": 50, "b": 10, "operation": "subtract"}'

# 测试回显
curl -X POST http://localhost:8080/api/demo-service/echo \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello from Java!"}'

# 获取时间
curl -X POST http://localhost:8080/api/demo-service/getTime \
  -H "Content-Type: application/json" \
  -d '{"format": "readable"}'

# 获取服务信息
curl -X POST http://localhost:8080/api/demo-service/getInfo
```

## 代码结构

- `Service.java` - 主服务文件，包含：
  - `NeoIPCClient` - 内部类实现 IPC 客户端
  - 线程池处理并发请求
  - 定时器实现心跳机制

## Maven 集成（可选）

如果使用 Maven，可以添加以下依赖：

```xml
<dependency>
    <groupId>com.google.code.gson</groupId>
    <artifactId>gson</artifactId>
    <version>2.8.9</version>
</dependency>
```

## 扩展开发

1. 添加新的处理器：
```java
service.addHandler("myMethod", params -> {
    // 你的业务逻辑
    Map<String, Object> response = new HashMap<>();
    response.put("result", "success");
    return response;
});
```

2. 修改服务名称：
```java
Map<String, String> metadata = new HashMap<>();
metadata.put("version", "1.0.0");
service.registerService("my-service-name", metadata);
```

## 性能特点

- 使用线程池处理并发请求
- 支持高并发场景
- 适合企业级应用

## 注意事项

- 使用小端序（Little Endian）进行二进制编码
- 心跳间隔为 30 秒
- 默认使用 CachedThreadPool 处理请求