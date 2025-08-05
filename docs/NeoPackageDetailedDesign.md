# Neo 项目包详细设计与测试指南

以下是为 "neo" 项目的每个包提供的详细设计文档，包括包的结构、文件设计要求、函数说明、测试要求和测试方法。每个包的设计遵循高内聚、低耦合原则，确保独立开发和测试，最终实现整体框架的稳定性。

本文档包含了最新的`gateway`、`ipc`包设计，以及异步处理相关的更新。

---

## 1. `internal/config`

### 包说明

- **职责**: 负责加载和解析框架配置，支持从文件、环境变量或远程服务加载配置。
- **设计目标**: 提供统一的配置访问接口，解耦配置来源，确保易扩展和可测试。
- **依赖**: 无（独立包）。
- **被依赖**: `transport`, `registry`, `core`, `cmd`。

### 文件结构

- `provider.go`: 定义配置提供者接口和实现。
- `config.go`: 定义配置结构体和操作方法。
- `errors.go`: 定义配置相关的错误类型。

### 文件设计要求

1. **`provider.go`**
   - 定义 `ConfigProvider` 接口，提供加载和获取配置的抽象方法。
   - 实现 `FileConfigProvider` 和 `EnvConfigProvider`，分别支持文件和环境变量加载。
   - 确保接口支持未来扩展（如添加远程配置源）。

2. **`config.go`**
   - 定义 `Config` 结构体，包含框架的全局配置。
   - 提供方法访问特定配置项（如超时时间、服务地址等）。
   - 支持嵌套配置结构，方便扩展。

3. **`errors.go`**
   - 定义配置加载和解析的错误类型。
   - 提供清晰的错误信息，便于调试。

### 函数说明

- **provider.go**
  - `LoadConfig(path string) (Config, error)`
    - **功能**: 从指定路径或环境变量加载配置。
    - **入参**: `path`（配置文件路径）。
    - **出参**: `Config`（配置对象），`error`（错误信息）。
  - `GetConfig(key string) interface{}`
    - **功能**: 获取指定配置项的值。
    - **入参**: `key`（配置键名）。
    - **出参**: `interface{}`（配置值）。

- **config.go**
  - `NewConfig() *Config`
    - **功能**: 创建默认配置对象。
    - **入参**: 无。
    - **出参**: `*Config`（配置对象指针）。
  - `GetTransportConfig() TransportConfig`
    - **功能**: 获取传输层相关配置。
    - **入参**: 无。
    - **出参**: `TransportConfig`（传输层配置）。

### 测试要求

- **正确性**: 验证配置从文件和环境变量加载的准确性。
- **异常处理**: 测试无效配置文件路径或格式的错误处理。
- **性能**: 验证配置加载的效率。

### 测试方法

- **单元测试**: 使用 mock 文件和环境变量测试 `LoadConfig` 和 `GetConfig`。
- **集成测试**: 验证配置在其他包（如 `transport`）中的正确使用。

---

## 2. `internal/types`

### 包说明

- **职责**: 定义框架通用的数据结构。
- **设计目标**: 提供清晰、稳定的数据类型，方便序列化和跨包使用。
- **依赖**: 无（独立包）。
- **被依赖**: `protocol`, `transport`, `core`。

### 目录结构

- `message.go`: 定义通用消息结构。
- `request.go`: 定义请求结构。
- `response.go`: 定义响应结构。

### 文件设计要求

1. **`message.go`**
   - 定义 `Message` 结构体，包含消息 ID 和内容。
   - 支持序列化（如 JSON、Protobuf）。

2. **`request.go`**
   - 定义 `Request` 结构体，包含请求方法和正文。
   - 支持扩展字段（如元数据）。

3. **`response.go`**
   - 定义 `Response` 结构体，包含状态码和正文。
   - 支持错误信息字段。

### 函数说明

- 无函数，仅定义结构体。

### 测试要求

- **正确性**: 验证结构体字段的定义和初始化。
- **兼容性**: 测试结构体在序列化/反序列化中的一致性。

### 测试方法

- **单元测试**: 检查每个字段的类型和默认值。
- **集成测试**: 验证在 `protocol` 和 `transport` 中的使用。

---

## 3. `internal/utils`

### 包说明

- **职责**: 提供通用工具函数，如日志记录、字符串处理等。
- **设计目标**: 提供高效、复用的工具函数，保持独立性。
- **依赖**: 无（独立包）。
- **被依赖**: 所有包（可选）。

### 目录结构

- `log.go`: 日志记录工具。
- `string.go`: 字符串处理工具。

### 文件设计要求

1. **`log.go`**
   - 提供日志记录函数，支持不同级别（Info、Error 等）。
   - 支持配置日志输出目标（如文件、控制台）。

2. **`string.go`**
   - 提供字符串操作函数，如格式化、校验等。
   - 确保高效且线程安全。

### 函数说明

- **log.go**
  - `Info(msg string, args ...interface{})`
    - **功能**: 记录信息日志。
    - **入参**: `msg`（日志消息），`args`（附加参数）。
    - **出参**: 无。
  - `Error(msg string, args ...interface{})`
    - **功能**: 记录错误日志。
    - **入参**: `msg`（错误消息），`args`（附加参数）。
    - **出参**: 无。

- **string.go**
  - `Format(str string, args ...interface{}) string`
    - **功能**: 格式化字符串。
    - **入参**: `str`（模板字符串），`args`（参数）。
    - **出参**: `string`（格式化后的字符串）。

### 测试要求

- **正确性**: 验证日志输出和字符串处理的正确性。
- **边界条件**: 测试空输入或异常输入。

### 测试方法

- **单元测试**: 测试每个工具函数的输入输出。
- **集成测试**: 验证工具函数在其他包中的使用效果。

---

## 4. `internal/protocol`

### 包说明

- **职责**: 定义和实现通信协议（如 HTTP、IPC）。
- **设计目标**: 通过接口解耦协议实现，支持多协议扩展。
- **依赖**: `internal/types`。
- **被依赖**: `transport/codec`。

### 目录结构

- `codec.go`: 定义协议编码/解码接口。
- `ipc.go`: IPC 协议实现。
- `http.go`: HTTP 协议实现。

### 文件设计要求

1. **`codec.go`**
   - 定义 `Codec` 接口，支持编码和解码。
   - 提供工厂函数创建具体协议实现。

2. **`ipc.go`**
   - 实现 IPC 协议的编码和解码。
   - 支持自定义二进制协议。

3. **`http.go`**
   - 实现 HTTP 协议的编码和解码。
   - 支持 JSON 序列化。

### 函数说明

- **codec.go**
  - `NewCodec(protocol string) Codec`
    - **功能**: 创建指定协议的编码器。
    - **入参**: `protocol`（协议类型）。
    - **出参**: `Codec`（编码器接口）。

- **ipc.go**
  - `Encode(ctx context.Context, msg types.Message) ([]byte, error)`
    - **功能**: 编码 IPC 消息。
    - **入参**: `ctx`（上下文），`msg`（消息）。
    - **出参**: `[]byte`（编码数据），`error`（错误）。
  - `Decode(ctx context.Context, data []byte) (types.Message, error)`
    - **功能**: 解码 IPC 消息。
    - **入参**: `ctx`（上下文），`data`（数据）。
    - **出参**: `types.Message`（消息对象），`error`（错误）。

- **http.go**
  - `Encode(ctx context.Context, msg types.Message) ([]byte, error)`
    - **功能**: 编码 HTTP 消息。
    - **入参**: `ctx`（上下文），`msg`（消息）。
    - **出参**: `[]byte`（编码数据），`error`（错误）。
  - `Decode(ctx context.Context, data []byte) (types.Message, error)`
    - **功能**: 解码 HTTP 消息。
    - **入参**: `ctx`（上下文），`data`（数据）。
    - **出参**: `types.Message`（消息对象），`error`（错误）。

### 测试要求

- **正确性**: 验证编码和解码的对称性。
- **兼容性**: 测试不同协议版本的兼容性。
- **性能**: 验证编码和解码的效率。

### 测试方法

- **单元测试**: 使用 mock 数据测试 `Encode` 和 `Decode`。
- **集成测试**: 在 `transport` 包中验证协议的端到端功能。

---

## 5. `internal/transport/conn`

### 包说明

- **职责**: 管理连接池，支持 TCP 和 Unix Socket。
- **设计目标**: 提供高效的连接管理，支持动态扩展和回收。
- **依赖**: `internal/config`。
- **被依赖**: `transport`.

### 目录结构

- `pool.go`: 定义连接池接口和实现。
- `conn.go`: 定义连接接口和实现。

### 文件设计要求

1. **`pool.go`**
   - 定义 `ConnectionPool` 接口，提供连接获取和释放方法。
   - 实现连接池，支持并发访问。

2. **`conn.go`**
   - 定义 `Conn` 接口，提供发送和接收方法。
   - 实现具体连接类型（如 TCP、Unix Socket）。

### 函数说明

- **pool.go**
  - `GetConnection(ctx context.Context, target string) (Conn, error)`
    - **功能**: 从连接池获取连接。
    - **入参**: `ctx`（上下文），`target`（目标地址）。
    - **出参**: `Conn`（连接对象），`error`（错误）。
  - `ReleaseConnection(conn Conn)`
    - **功能**: 释放连接回池中。
    - **入参**: `conn`（连接对象）。
    - **出参**: 无。

- **conn.go**
  - `Send(ctx context.Context, msg []byte) error`
    - **功能**: 发送消息。
    - **入参**: `ctx`（上下文），`msg`（消息数据）。
    - **出参**: `error`（错误）。
  - `Receive(ctx context.Context) ([]byte, error)`
    - **功能**: 接收消息。
    - **入参**: `ctx`（上下文）。
    - **出参**: `[]byte`（接收数据），`error`（错误）。

### 测试要求

- **正确性**: 验证连接获取和释放的正确性。
- **并发性**: 测试连接池在高并发下的表现。
- **异常处理**: 测试连接失败或中断的处理。

### 测试方法

- **单元测试**: 使用 mock 连接测试 `GetConnection` 和 `ReleaseConnection`。
- **集成测试**: 在 `transport` 包中验证连接管理的实际效果。

---

## 6. `internal/transport/codec`

### 包说明

- **职责**: 处理消息的编码和解码。
- **设计目标**: 通过接口解耦协议实现，支持多协议。
- **依赖**: `internal/protocol`。
- **被依赖**: `transport`.

### 目录结构

- `codec.go`: 定义编码/解码接口和实现。

### 文件设计要求

1. **`codec.go`**
   - 定义 `Codec` 接口，调用 `protocol` 包的编码/解码逻辑。
   - 提供工厂函数选择具体协议。

### 函数说明

- `NewCodec(protocol string) Codec`
  - **功能**: 创建指定协议的编码器。
  - **入参**: `protocol`（协议类型）。
  - **出参**: `Codec`（编码器接口）。
- `Encode(ctx context.Context, msg interface{}) ([]byte, error)`
  - **功能**: 编码消息。
  - **入参**: `ctx`（上下文），`msg`（消息）。
  - **出参**: `[]byte`（编码数据），`error`（错误）。
- `Decode(ctx context.Context, data []byte) (interface{}, error)`
  - **功能**: 解码消息。
  - **入参**: `ctx`（上下文），`data`（数据）。
  - **出参**: `interface{}`（消息对象），`error`（错误）。

### 测试要求

- **正确性**: 验证编码和解码的准确性。
- **协议切换**: 测试不同协议的切换功能。

### 测试方法

- **单元测试**: 使用 mock 数据测试 `Encode` 和 `Decode`。
- **集成测试**: 在 `transport` 包中验证编码/解码的端到端功能。

---

## 7. `internal/transport/retry`

### 包说明

- **职责**: 实现消息传输的重试策略。
- **设计目标**: 提供灵活的重试机制，支持配置。
- **依赖**: `internal/config`。
- **被依赖**: `transport`.

### 目录结构

- `retry.go`: 定义重试策略接口和实现。

### 文件设计要求

1. **`retry.go`**
   - 定义 `RetryPolicy` 接口，支持不同重试策略。
   - 实现指数退避策略。

### 函数说明

- `NewRetryPolicy(config Config) RetryPolicy`
  - **功能**: 创建重试策略。
  - **入参**: `config`（配置）。
  - **出参**: `RetryPolicy`（重试策略接口）。
- `Execute(ctx context.Context, operation func() error) error`
  - **功能**: 执行带重试的操作。
  - **入参**: `ctx`（上下文），`operation`（操作函数）。
  - **出参**: `error`（错误）。

### 测试要求

- **正确性**: 验证重试策略的触发条件。
- **边界条件**: 测试最大重试次数和间隔。

### 测试方法

- **单元测试**: 使用 mock 操作测试 `Execute` 的重试逻辑。
- **集成测试**: 在 `transport` 包中验证重试的实际效果。

---

## 8. `internal/transport` (更新)

### 包说明

- **职责**: 管理传输层，包括同步和异步传输。
- **设计目标**: 提供统一的传输接口，支持多种传输方式。
- **依赖**: `conn`, `codec`, `retry` 子包。
- **被依赖**: `core`。

### 新增文件

- `ipc_transport.go`: IPC传输实现。
- `async_ipc_transport.go`: 异步IPC传输实现。

### 文件设计要求

1. **`ipc_transport.go`**
   - 实现 `IPCTransport` 结构体。
   - 与 `IPCServer` 集成。
   - 支持请求转发。

2. **`async_ipc_transport.go`**
   - 实现 `AsyncIPCTransport` 结构体。
   - 支持异步请求-响应模式。
   - 实现 `SendAndReceive` 方法。

### 函数说明

- **ipc_transport.go**
  - `NewIPCTransport(ipcServer *IPCServer) *IPCTransport`
    - **功能**: 创建IPC传输。
    - **入参**: `ipcServer`（IPC服务器）。
    - **出参**: `*IPCTransport`（传输实例）。

- **async_ipc_transport.go**
  - `SendAndReceive(ctx context.Context, serviceName string, method string, data []byte) ([]byte, error)`
    - **功能**: 发送请求并等待响应。
    - **入参**: `ctx`（上下文），`serviceName`（服务名），`method`（方法），`data`（数据）。
    - **出参**: `[]byte`（响应数据），`error`（错误）。

---

## 9. `internal/registry`

### 包说明

- **职责**: 管理服务注册和发现。
- **设计目标**: 提供高效的服务注册和发现机制，支持多种注册中心。
- **依赖**: `internal/config`。
- **被依赖**: `core`.

### 目录结构

- `registry.go`: 定义服务注册接口和实现。
- `instance.go`: 定义服务实例结构。

### 文件设计要求

1. **`registry.go`**
   - 定义 `ServiceRegistry` 接口，支持注册和发现。
   - 实现内存注册中心，支持扩展到其他注册中心（如 ZooKeeper）。

2. **`instance.go`**
   - 定义 `ServiceInstance` 结构体，包含服务名称和地址。

### 函数说明

- **registry.go**
  - `NewServiceRegistry(config Config) ServiceRegistry`
    - **功能**: 创建服务注册中心。
    - **入参**: `config`（配置）。
    - **出参**: `ServiceRegistry`（注册中心接口）。
  - `RegisterService(ctx context.Context, instance ServiceInstance) error`
    - **功能**: 注册服务实例。
    - **入参**: `ctx`（上下文），`instance`（服务实例）。
    - **出参**: `error`（错误）。
  - `DiscoverService(ctx context.Context, name string) ([]ServiceInstance, error)`
    - **功能**: 发现服务实例。
    - **入参**: `ctx`（上下文），`name`（服务名称）。
    - **出参**: `[]ServiceInstance`（服务实例列表），`error`（错误）。

### 测试要求

- **正确性**: 验证服务注册和发现的准确性。
- **并发性**: 测试高并发下的注册和发现。

### 测试方法

- **单元测试**: 使用 mock 服务实例测试 `RegisterService` 和 `DiscoverService`。
- **集成测试**: 在 `core` 包中验证服务注册和发现的完整流程。

---

## 10. `internal/core` (更新)

### 包说明

- **职责**: 处理核心业务逻辑，协调请求和响应，支持同步和异步处理。
- **设计目标**: 提供清晰的请求处理流程，解耦业务逻辑和传输层。
- **依赖**: `internal/transport`, `internal/registry`。
- **被依赖**: `cmd`, `gateway`。

### 目录结构

- `service.go`: 定义服务接口和同步实现。
- `async_service.go`: 异步服务实现。
- `service_test.go`: 服务测试。

### 文件设计要求

1. **`service.go`**
   - 定义 `Service` 接口，处理请求和生成响应。
   - 实现 `coreService` 结构体。
   - 提供 `Close()` 方法支持优雅关闭。

2. **`async_service.go`**
   - 实现 `AsyncService` 结构体。
   - 支持异步请求处理。
   - 使用 `AsyncTransport` 进行非阻塞通信。

### 函数说明

- `NewService(transport Transport, registry ServiceRegistry) Service`
  - **功能**: 创建同步服务实例。
  - **入参**: `transport`（传输层接口），`registry`（注册中心接口）。
  - **出参**: `Service`（服务接口）。
- `NewAsyncService(transport *AsyncTransport, registry ServiceRegistry) Service`
  - **功能**: 创建异步服务实例。
  - **入参**: `transport`（异步传输），`registry`（注册中心接口）。
  - **出参**: `Service`（服务接口）。
- `HandleRequest(ctx context.Context, req types.Request) (types.Response, error)`
  - **功能**: 处理请求并返回响应。
  - **入参**: `ctx`（上下文），`req`（请求）。
  - **出参**: `types.Response`（响应），`error`（错误）。
- `Close() error`
  - **功能**: 优雅关闭服务。
  - **出参**: `error`（错误）。

### 测试要求

- **正确性**: 验证请求处理的逻辑正确性。
- **异常处理**: 测试异常请求的处理。
- **并发性**: 测试异步服务的并发处理能力。

### 测试方法

- **单元测试**: 使用 mock 请求测试 `HandleRequest`。
- **集成测试**: 在 `cmd` 包中验证核心逻辑的运行效果。

---

## 11. `internal/gateway` (新增)

### 包说明

- **职责**: 提供HTTP网关服务，将外部HTTP请求转发到内部服务。
- **设计目标**: 实现HTTP到IPC的透明转换，支持RESTful API。
- **依赖**: `internal/core`, `internal/registry`, `net/http`。
- **被依赖**: `cmd/gateway`, `cmd/neo`。

### 目录结构

- `http_gateway.go`: HTTP网关实现。

### 文件设计要求

1. **`http_gateway.go`**
   - 定义 `HTTPGateway` 结构体。
   - 实现路由解析（`/api/{service}/{method}`）。
   - 提供健康检查端点（`/health`）。
   - 支持优雅关闭。

### 函数说明

- `NewHTTPGateway(service Service, registry ServiceRegistry, addr string) *HTTPGateway`
  - **功能**: 创建HTTP网关实例。
  - **入参**: `service`（核心服务），`registry`（服务注册中心），`addr`（监听地址）。
  - **出参**: `*HTTPGateway`（网关实例）。

- `Start() error`
  - **功能**: 启动HTTP服务器。
  - **出参**: `error`（错误信息）。

- `Stop(ctx context.Context) error`
  - **功能**: 优雅停止服务器。
  - **入参**: `ctx`（上下文）。
  - **出参**: `error`（错误信息）。

- `handleAPIRequest(w http.ResponseWriter, r *http.Request)`
  - **功能**: 处理API请求。
  - **入参**: `w`（响应写入器），`r`（HTTP请求）。

### 测试要求

- **正确性**: 验证路由解析和请求转发。
- **异常处理**: 测试服务不存在时的错误响应。
- **并发性**: 测试高并发请求处理。

### 测试方法

- **单元测试**: 使用 `httptest` 测试处理器。
- **集成测试**: 验证完整的HTTP到IPC流程。

---

## 12. `internal/ipc` (新增)

### 包说明

- **职责**: 管理进程间通信服务器，处理服务注册和消息路由。
- **设计目标**: 提供高性能的IPC通信，支持异步消息处理。
- **依赖**: `internal/registry`, `net`。
- **被依赖**: `transport`, `cmd`。

### 目录结构

- `server.go`: IPC服务器实现。
- `async_handler.go`: 异步请求处理器。

### 文件设计要求

1. **`server.go`**
   - 定义 `IPCServer` 结构体。
   - 实现TCP服务器，默认监听9999端口。
   - 管理客户端连接和服务映射。
   - 支持消息类型：REQUEST、RESPONSE、REGISTER、HEARTBEAT。

2. **`async_handler.go`**
   - 定义 `RequestHandler` 结构体。
   - 实现 `AsyncIPCServer` 结构体。
   - 管理异步请求-响应映射。
   - 支持超时处理。

### 消息格式

```
[消息长度:4字节][消息类型:1字节][ID长度:4字节][ID:N字节]
[服务名长度:4字节][服务名:N字节][方法名长度:4字节][方法名:N字节]
[元数据长度:4字节][元数据JSON:N字节][数据长度:4字节][数据:N字节]
```

### 函数说明

#### server.go

- `NewIPCServer(addr string, registry ServiceRegistry) *IPCServer`
  - **功能**: 创建IPC服务器实例。
  - **入参**: `addr`（监听地址），`registry`（服务注册中心）。
  - **出参**: `*IPCServer`（服务器实例）。

- `Start() error`
  - **功能**: 启动IPC服务器。
  - **出参**: `error`（错误信息）。

- `SendRequest(serviceName string, method string, data []byte) (*IPCMessage, error)`
  - **功能**: 向指定服务发送请求。
  - **入参**: `serviceName`（服务名），`method`（方法名），`data`（请求数据）。
  - **出参**: `*IPCMessage`（响应消息），`error`（错误）。

#### async_handler.go

- `NewAsyncIPCServer(ipcServer *IPCServer) *AsyncIPCServer`
  - **功能**: 创建异步IPC服务器。
  - **入参**: `ipcServer`（基础IPC服务器）。
  - **出参**: `*AsyncIPCServer`（异步服务器）。

- `ForwardRequest(ctx context.Context, serviceName string, method string, data []byte) ([]byte, error)`
  - **功能**: 转发请求并等待响应。
  - **入参**: `ctx`（上下文），`serviceName`（服务名），`method`（方法名），`data`（数据）。
  - **出参**: `[]byte`（响应数据），`error`（错误）。

### 测试要求

- **正确性**: 验证服务注册和消息路由。
- **异步处理**: 测试并发请求的正确处理。
- **超时处理**: 验证请求超时机制。
- **协议兼容**: 测试与Python客户端的互操作性。

### 测试方法

- **单元测试**: 测试消息编解码、连接管理。
- **集成测试**: 验证完整的IPC通信流程。
- **压力测试**: 测试高并发场景下的性能。

---

## 13. `cmd`

### 包说明

- **职责**: 框架的启动和关闭入口。
- **设计目标**: 提供清晰的启动流程，协调所有模块。
- **依赖**: `internal/core`, `internal/config`。

### 目录结构

- `main.go`: 基础框架启动入口。
- `neo/main.go`: 主应用程序启动器（推荐使用）。
- `gateway/main.go`: 独立的生产环境网关。
- `simple_gateway/main.go`: 测试用简化网关。
- `test_gateway/main.go`: 集成测试网关。

### 文件设计要求

1. **`main.go`**
   - 实现基础框架的启动和优雅关闭逻辑。
   - 协调配置加载、协议初始化、服务注册和监听器启动。

2. **`neo/main.go`**（推荐）
   - 完整的Application级别应用程序管理。
   - 集成所有组件：HTTP网关、IPC服务器、核心服务。
   - 支持命令行参数配置（-config, -log, -http, -ipc）。
   - 实现优雅启动和关闭流程。
   - 统一的日志和错误处理。
   - 端口：HTTP 8080, IPC 9999。

3. **`gateway/main.go`**
   - 专注于网关功能的独立应用。
   - 启动IPC服务器和HTTP网关。
   - 支持优雅关闭。
   - 端口：HTTP 18080, IPC 19999。

### 函数说明

- `main()`
  - **功能**: 框架启动入口。
  - **入参**: 无。
  - **出参**: 无。
- `shutdown()`
  - **功能**: 优雅关闭框架。
  - **入参**: 无。
  - **出参**: 无。

### 测试要求

- **正确性**: 验证启动和关闭流程的完整性。
- **健壮性**: 测试异常情况下的关闭行为。

### 测试方法

- **单元测试**: 测试启动和关闭的逻辑分支。
- **集成测试**: 验证整个框架的启动和关闭流程。

---

## 14. `pkg`

### 包说明

- **职责**: 提供公开 API，供外部调用。
- **设计目标**: 提供简洁、稳定的接口，隐藏内部实现。
- **依赖**: `internal` 的部分包。

### 目录结构

- `api.go`: 定义公开 API 接口和实现。

### 文件设计要求

1. **`api.go`**
   - 定义公开的结构体和接口，供外部调用。
   - 隐藏内部实现细节。

### 函数说明

- `NewClient(config Config) Client`
  - **功能**: 创建客户端实例。
  - **入参**: `config`（配置）。
  - **出参**: `Client`（客户端接口）。
- `Call(ctx context.Context, req types.Request) (types.Response, error)`
  - **功能**: 执行远程调用。
  - **入参**: `ctx`（上下文），`req`（请求）。
  - **出参**: `types.Response`（响应），`error`（错误）。

### 测试要求

- **正确性**: 验证 API 调用的功能。
- **兼容性**: 测试 API 的向后兼容性。
- **性能**: 验证 API 的响应时间。

### 测试方法

- **单元测试**: 测试每个 API 的输入输出。
- **集成测试**: 模拟外部调用，验证 API 的实际表现。

---

## 15. `examples-ipc` (新增)

### 包说明

- **职责**: 多语言服务集成示例，展示各种编程语言如何接入Neo框架。
- **设计目标**: 为每种语言提供完整的IPC客户端实现和示例服务。
- **支持语言**: Python、Go、Node.js、Java、PHP。
- **依赖**: 各语言的标准库（socket、asyncio等）。

### 目录结构

```
examples-ipc/
├── python/
│   ├── neo_client.py    # Python IPC客户端库
│   └── service.py       # Python示例服务
├── go/
│   └── service.go       # Go示例服务
├── nodejs/
│   └── service.js       # Node.js示例服务
├── java/
│   └── Service.java     # Java示例服务
├── php/
│   └── service.php      # PHP示例服务
└── README.md           # 多语言集成说明
```

### 各语言实现要求

1. **Python (`python/`)**
   - 实现 `NeoIPCClient` 类
   - 支持异步消息处理（asyncio）
   - 实现二进制协议编解码
   - 提供方法处理器注册机制

2. **Go (`go/`)**
   - 原生性能实现
   - 类型安全的消息处理
   - 支持并发请求

3. **Node.js (`nodejs/`)**
   - 事件驱动模型
   - 无额外依赖
   - Promise/async支持

4. **Java (`java/`)**
   - 面向对象设计
   - 使用Gson进行JSON处理
   - 线程安全实现

5. **PHP (`php/`)**
   - 需要sockets扩展
   - 同步消息处理
   - 支持Web集成

### 统一服务接口

所有语言都实现以下方法：
- `hello(name)`: 问候服务
- `calculate(expression)`: 数学计算
- `echo(message)`: 消息回显
- `getTime()`: 获取当前时间
- `getInfo()`: 获取服务信息

### 测试要求

- **协议兼容**: 验证与Go IPC服务器的通信
- **功能完整**: 确保所有方法正确实现
- **性能测试**: 验证各语言的性能表现
- **错误处理**: 测试异常情况的处理

---

# 总结

以上文档为每个包提供了详细的设计说明、文件结构、函数说明和测试指南，包含了最新的`gateway`、`ipc`包设计，以及异步处理支持。这些文档将为开发和测试提供清晰的指导，支撑项目的重构和扩展工作。

---

*文档编写：Cogito Yan (Neospecies AI)*  
*联系方式：neospecies@outlook.com*