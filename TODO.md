# Neo Framework 开发计划

## 开发原则

1. **循序渐进**：按照依赖关系，从基础包到上层包逐步开发
2. **接口先行**：先定义接口，明确输入输出，再进行实现
3. **测试驱动**：每个包都要有完整的单元测试和集成测试
4. **文档完善**：每个包都要有清晰的文档说明

## 包开发顺序和详细规划

### 第一阶段：基础包（无依赖）

#### 1. `internal/types` ✅ 优先级：最高
**功能定义**：定义框架通用的数据结构，作为所有包的基础类型定义

**详细功能规划**：
- 定义核心数据结构（Message、Request、Response）
- 支持 JSON 序列化/反序列化
- 提供结构体验证方法

**接口定义**：
```go
// Message 通用消息结构
type Message struct {
    ID        string                 `json:"id"`
    Type      MessageType           `json:"type"`
    Service   string                `json:"service"`
    Method    string                `json:"method"`
    Metadata  map[string]string     `json:"metadata,omitempty"`
    Body      []byte                `json:"body"`
    Timestamp time.Time             `json:"timestamp"`
}

// Request 请求结构
type Request struct {
    ID        string                 `json:"id"`
    Service   string                `json:"service"`
    Method    string                `json:"method"`
    Body      []byte                `json:"body"`
    Metadata  map[string]string     `json:"metadata,omitempty"`
}

// Response 响应结构
type Response struct {
    ID        string                 `json:"id"`
    Status    int                   `json:"status"`
    Body      []byte                `json:"body"`
    Error     string                `json:"error,omitempty"`
    Metadata  map[string]string     `json:"metadata,omitempty"`
}
```

**测试要求**：
- [ ] 结构体字段验证测试
- [ ] JSON 序列化/反序列化测试
- [ ] 边界值测试（空值、超大数据）
- [ ] 并发安全测试

#### 2. `internal/utils` ✅ 优先级：最高
**功能定义**：提供通用工具函数，被所有其他包使用

**详细功能规划**：
- 日志记录工具（支持级别、格式化）
- 字符串处理工具（验证、格式化）
- ID 生成工具（UUID、自增ID）
- 错误处理工具

**接口定义**：
```go
// Logger 接口
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
}

// StringUtils 字符串工具
func ValidateServiceName(name string) error
func FormatEndpoint(service, method string) string
func GenerateRequestID() string
```

**测试要求**：
- [ ] 日志输出格式测试
- [ ] 字符串验证边界测试
- [ ] ID 生成唯一性测试
- [ ] 并发场景下的线程安全测试

#### 3. `internal/config` ✅ 优先级：最高
**功能定义**：配置管理，支持多种配置源

**详细功能规划**：
- 支持文件配置（YAML/JSON）
- 支持环境变量覆盖
- 配置热更新（可选）
- 配置验证

**接口定义**：
```go
// ConfigProvider 配置提供者接口
type ConfigProvider interface {
    Load(source string) error
    Get(key string) interface{}
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    Watch(key string, callback func(value interface{}))
}

// Config 主配置结构
type Config struct {
    Transport TransportConfig `yaml:"transport"`
    Registry  RegistryConfig  `yaml:"registry"`
    Gateway   GatewayConfig   `yaml:"gateway"`
    IPC       IPCConfig       `yaml:"ipc"`
}
```

**测试要求**：
- [ ] 文件加载测试（有效/无效文件）
- [ ] 环境变量覆盖测试
- [ ] 配置验证测试
- [ ] 配置更新通知测试

### 第二阶段：协议和注册包

#### 4. `internal/protocol` ⬜ 优先级：高
**功能定义**：定义和实现通信协议

**详细功能规划**：
- 定义协议接口
- 实现 HTTP 协议编解码
- 实现 IPC 二进制协议编解码
- 协议版本管理

**接口定义**：
```go
// Codec 编解码器接口
type Codec interface {
    Encode(msg types.Message) ([]byte, error)
    Decode(data []byte) (types.Message, error)
    Version() string
}

// ProtocolFactory 协议工厂
func NewCodec(protocol string) (Codec, error)

// IPC 消息格式
type IPCMessage struct {
    Length    uint32
    Type      uint8
    ID        string
    Service   string
    Method    string
    Metadata  map[string]string
    Data      []byte
}
```

**测试要求**：
- [ ] HTTP 编解码对称性测试
- [ ] IPC 二进制协议测试
- [ ] 协议版本兼容性测试
- [ ] 大消息处理测试
- [ ] 错误数据处理测试

#### 5. `internal/registry` ⬜ 优先级：高
**功能定义**：服务注册与发现

**详细功能规划**：
- 内存注册中心实现
- 服务健康检查
- 服务元数据管理
- 负载均衡策略

**接口定义**：
```go
// ServiceInstance 服务实例
type ServiceInstance struct {
    ID          string
    Name        string
    Address     string
    Port        int
    Metadata    map[string]string
    HealthCheck HealthCheck
    RegisterTime time.Time
}

// ServiceRegistry 注册中心接口
type ServiceRegistry interface {
    Register(ctx context.Context, instance ServiceInstance) error
    Deregister(ctx context.Context, instanceID string) error
    Discover(ctx context.Context, serviceName string) ([]ServiceInstance, error)
    Watch(ctx context.Context, serviceName string) (<-chan ServiceEvent, error)
    HealthCheck(ctx context.Context, instanceID string) error
}

// LoadBalancer 负载均衡器
type LoadBalancer interface {
    Select(instances []ServiceInstance) (*ServiceInstance, error)
}
```

**测试要求**：
- [ ] 服务注册/注销测试
- [ ] 服务发现测试
- [ ] 健康检查测试
- [ ] 并发注册测试
- [ ] Watch 机制测试

### 第三阶段：传输层包

#### 6. `internal/transport/conn` ⬜ 优先级：中
**功能定义**：连接管理，支持连接池

**详细功能规划**：
- TCP 连接池实现
- Unix Socket 支持
- 连接健康检查
- 连接复用

**接口定义**：
```go
// Connection 连接接口
type Connection interface {
    Send(ctx context.Context, data []byte) error
    Receive(ctx context.Context) ([]byte, error)
    Close() error
    IsHealthy() bool
    RemoteAddr() string
}

// ConnectionPool 连接池接口
type ConnectionPool interface {
    Get(ctx context.Context, addr string) (Connection, error)
    Put(conn Connection) error
    Close() error
    Stats() PoolStats
}

// PoolConfig 连接池配置
type PoolConfig struct {
    MaxSize         int
    MinSize         int
    MaxIdleTime     time.Duration
    HealthCheckInterval time.Duration
}
```

**测试要求**：
- [ ] 连接创建和销毁测试
- [ ] 连接池扩缩容测试
- [ ] 连接健康检查测试
- [ ] 并发获取连接测试
- [ ] 连接泄露检测测试

#### 7. `internal/transport/codec` ⬜ 优先级：中
**功能定义**：传输层编解码

**详细功能规划**：
- 消息帧定义
- 流式传输支持
- 压缩支持（可选）

**接口定义**：
```go
// TransportCodec 传输层编解码器
type TransportCodec interface {
    EncodeFrame(msg []byte) ([]byte, error)
    DecodeFrame(reader io.Reader) ([]byte, error)
    SupportsStreaming() bool
}

// FrameHeader 帧头定义
type FrameHeader struct {
    Version     uint8
    Flags       uint8
    Length      uint32
    Checksum    uint32
}
```

**测试要求**：
- [ ] 帧编解码测试
- [ ] 流式传输测试
- [ ] 数据完整性测试
- [ ] 压缩功能测试

#### 8. `internal/transport/retry` ⬜ 优先级：中
**功能定义**：重试策略实现

**详细功能规划**：
- 指数退避策略
- 固定间隔策略
- 自定义重试条件
- 重试统计

**接口定义**：
```go
// RetryPolicy 重试策略接口
type RetryPolicy interface {
    Execute(ctx context.Context, fn func() error) error
    ShouldRetry(err error) bool
    NextInterval(attempt int) time.Duration
}

// RetryConfig 重试配置
type RetryConfig struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
    Multiplier      float64
    RetryableErrors []error
}

// RetryStats 重试统计
type RetryStats struct {
    TotalAttempts   int
    SuccessCount    int
    FailureCount    int
    LastError       error
}
```

**测试要求**：
- [ ] 不同重试策略测试
- [ ] 重试条件判断测试
- [ ] 超时控制测试
- [ ] 重试统计测试

#### 9. `internal/transport` ⬜ 优先级：中
**功能定义**：统一的传输层抽象

**详细功能规划**：
- 整合连接管理、编解码、重试
- 提供同步和异步传输接口
- 请求追踪和监控

**接口定义**：
```go
// Transport 传输层接口
type Transport interface {
    Send(ctx context.Context, req types.Request) (types.Response, error)
    Close() error
}

// AsyncTransport 异步传输接口
type AsyncTransport interface {
    Transport
    SendAsync(ctx context.Context, req types.Request) (<-chan types.Response, error)
    Subscribe(pattern string, handler func(msg types.Message)) error
}

// TransportMetrics 传输层指标
type TransportMetrics struct {
    RequestCount    int64
    ResponseCount   int64
    ErrorCount      int64
    AvgLatency      time.Duration
}
```

**测试要求**：
- [ ] 同步传输测试
- [ ] 异步传输测试
- [ ] 超时处理测试
- [ ] 并发请求测试
- [ ] 监控指标测试

### 第四阶段：核心服务包

#### 10. `internal/ipc` ⬜ 优先级：高
**功能定义**：进程间通信服务器

**详细功能规划**：
- TCP 服务器实现
- 客户端连接管理
- 消息路由
- 异步消息处理

**接口定义**：
```go
// IPCServer IPC服务器
type IPCServer struct {
    addr     string
    registry ServiceRegistry
    handlers map[string]Handler
    clients  map[string]*Client
}

// Handler 消息处理器
type Handler func(ctx context.Context, msg types.Message) (types.Message, error)

// Client 客户端连接
type Client struct {
    ID       string
    Conn     net.Conn
    Services []string
}

// IPCServer 方法
func (s *IPCServer) Start() error
func (s *IPCServer) Stop(ctx context.Context) error
func (s *IPCServer) RegisterHandler(service string, handler Handler)
func (s *IPCServer) SendRequest(clientID string, msg types.Message) error
```

**测试要求**：
- [ ] 服务器启动停止测试
- [ ] 客户端连接管理测试
- [ ] 消息路由测试
- [ ] 并发消息处理测试
- [ ] 异常断开处理测试

#### 11. `internal/core` ⬜ 优先级：高
**功能定义**：核心业务逻辑

**详细功能规划**：
- 请求处理流程
- 服务调用抽象
- 中间件支持
- 请求上下文管理

**接口定义**：
```go
// Service 服务接口
type Service interface {
    Name() string
    HandleRequest(ctx context.Context, req types.Request) (types.Response, error)
    Middleware() []Middleware
    Close() error
}

// Middleware 中间件接口
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc 处理函数
type HandlerFunc func(ctx context.Context, req types.Request) (types.Response, error)

// ServiceOptions 服务选项
type ServiceOptions struct {
    Name        string
    Transport   Transport
    Registry    ServiceRegistry
    Middlewares []Middleware
    Timeout     time.Duration
}
```

**测试要求**：
- [ ] 请求处理流程测试
- [ ] 中间件链测试
- [ ] 超时控制测试
- [ ] 并发请求测试
- [ ] 优雅关闭测试

#### 12. `internal/gateway` ⬜ 优先级：高
**功能定义**：HTTP 网关服务

**详细功能规划**：
- HTTP 服务器实现
- 路由解析（/api/{service}/{method}）
- 请求转换（HTTP → 内部协议）
- 响应转换（内部协议 → HTTP）
- 健康检查端点

**接口定义**：
```go
// HTTPGateway HTTP网关
type HTTPGateway struct {
    addr     string
    service  Service
    registry ServiceRegistry
    router   *mux.Router
}

// RouteConfig 路由配置
type RouteConfig struct {
    Path        string
    Service     string
    Method      string
    HTTPMethod  string
    Middlewares []Middleware
}

// HTTPGateway 方法
func (g *HTTPGateway) Start() error
func (g *HTTPGateway) Stop(ctx context.Context) error
func (g *HTTPGateway) RegisterRoute(config RouteConfig)
func (g *HTTPGateway) HealthCheck(w http.ResponseWriter, r *http.Request)
```

**测试要求**：
- [ ] HTTP 服务器测试
- [ ] 路由解析测试
- [ ] 请求转换测试
- [ ] 错误处理测试
- [ ] 健康检查测试

### 第五阶段：应用层包

#### 13. `cmd` ⬜ 优先级：低
**功能定义**：命令行入口和应用启动

**详细功能规划**：
- 应用初始化流程
- 信号处理
- 优雅关闭
- 命令行参数解析

**接口定义**：
```go
// Application 应用主体
type Application struct {
    config      *config.Config
    gateway     *gateway.HTTPGateway
    ipcServer   *ipc.IPCServer
    registry    registry.ServiceRegistry
}

// Application 方法
func (app *Application) Initialize() error
func (app *Application) Start() error
func (app *Application) Shutdown(ctx context.Context) error
```

**测试要求**：
- [ ] 启动流程测试
- [ ] 配置加载测试
- [ ] 信号处理测试
- [ ] 优雅关闭测试

#### 14. `pkg` ⬜ 优先级：低
**功能定义**：公开 API 包

**详细功能规划**：
- 客户端 SDK
- 简化的 API 接口
- 辅助工具函数

**接口定义**：
```go
// Client 客户端接口
type Client interface {
    Call(ctx context.Context, service, method string, req interface{}, resp interface{}) error
    Subscribe(service string, handler func(event Event)) error
    Close() error
}

// ClientConfig 客户端配置
type ClientConfig struct {
    Endpoints   []string
    Timeout     time.Duration
    Retry       RetryConfig
    LoadBalance string
}
```

**测试要求**：
- [ ] 客户端调用测试
- [ ] 订阅功能测试
- [ ] 错误处理测试
- [ ] 负载均衡测试

### 第六阶段：集成测试

#### 15. Python 服务集成测试 ⬜ 优先级：中
**测试内容**：
- [ ] Python 客户端与 Go 服务器通信测试
- [ ] 服务注册流程测试
- [ ] 异步消息处理测试
- [ ] 错误处理和重连测试
- [ ] 性能基准测试

#### 16. 端到端集成测试 ⬜ 优先级：低
**测试内容**：
- [ ] 完整调用链测试（HTTP → Gateway → IPC → Service）
- [ ] 多服务协同测试
- [ ] 故障恢复测试
- [ ] 性能压力测试
- [ ] 监控指标验证

## 测试标准

每个包必须达到以下标准：
1. **单元测试覆盖率** > 80%
2. **性能基准测试**：定义性能基线
3. **并发安全**：通过 race detector 测试
4. **错误处理**：所有错误情况都有测试覆盖
5. **文档完整**：包含包说明、示例代码、API 文档

## 开发流程

对于每个包的开发：
1. **设计评审**：确认接口设计符合整体架构
2. **测试先行**：先编写测试用例
3. **实现代码**：按照设计实现功能
4. **代码评审**：确保代码质量
5. **集成验证**：与依赖包集成测试
6. **文档更新**：更新相关文档

## 进度追踪

- ✅ 已完成
- 🚧 进行中
- ⬜ 未开始
- ❌ 阻塞

## 风险和依赖

1. **服务发现机制**：当前存在服务注册但无法发现的问题，需要在 registry 和 ipc 包中重点解决
2. **异步机制设计**：需要确保异步请求-响应匹配机制的可靠性
3. **性能要求**：IPC 协议需要达到低延迟、高吞吐的要求

## 里程碑

1. **M1 - 基础设施**（第1-3个包）：2周
2. **M2 - 协议和传输**（第4-9个包）：3周
3. **M3 - 核心服务**（第10-12个包）：3周
4. **M4 - 应用集成**（第13-16个包）：2周

总计：约10周完成整个框架开发