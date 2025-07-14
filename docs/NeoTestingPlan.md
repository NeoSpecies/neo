# Neo 项目测试计划

以下是针对 "neo" 项目每个包的详细测试计划，涵盖单元测试、集成测试、边界测试和性能测试，确保重构后的代码功能正确、性能可靠、异常处理得当。测试计划基于 `NeoPackageDetailedDesign.md` 的要求，细化测试步骤，提供具体测试用例和工具建议，方便您进行具体操作。

---

## 测试总体策略

1. **单元测试**：针对每个包的独立功能，验证函数和方法的正确性，覆盖正常场景、边界条件和异常情况。
2. **集成测试**：验证包与其他包的交互行为，确保依赖关系正确。
3. **边界测试**：测试包在极端输入或异常环境下的表现。
4. **性能测试**：评估包在高负载或并发场景下的性能。
5. **测试工具**：
   - **Go 标准库**：`testing` 包用于单元测试。
   - **Mock 工具**：`github.com/stretchr/testify/mock` 用于模拟依赖。
   - **性能测试工具**：`go test -bench` 用于基准测试。
   - **覆盖率工具**：`go test -cover` 检查测试覆盖率。

---

## 1. `internal/config` 测试计划

### 测试要求

- **正确性**：验证配置从文件和环境变量加载的准确性。
- **异常处理**：测试无效配置文件路径或格式的错误处理。
- **性能**：验证配置加载的效率。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **正确加载文件配置**：创建有效的 JSON/YAML 配置文件，调用 `LoadConfig`，验证 `Config` 对象的字段值。
     - **正确加载环境变量**：设置环境变量，调用 `LoadConfig`，验证返回的 `Config` 正确。
     - **无效路径**：传入不存在的配置文件路径，验证返回特定错误（如 `ErrFileNotFound`）。
     - **无效格式**：传入格式错误的配置文件，验证返回特定错误（如 `ErrInvalidFormat`）。
     - **获取配置项**：调用 `GetConfig` 获取特定键，验证返回值正确。
   - **实现**：
     ```go
     package config_test

     import (
         "testing"
         "github.com/stretchr/testify/assert"
         "neo/internal/config"
     )

     func TestLoadConfig(t *testing.T) {
         provider := config.NewFileConfigProvider()
         cfg, err := provider.LoadConfig("testdata/valid_config.json")
         assert.NoError(t, err)
         assert.Equal(t, 30, cfg.Transport.Timeout)

         _, err = provider.LoadConfig("testdata/invalid_path.json")
         assert.Error(t, err)
         assert.Contains(t, err.Error(), "文件未找到")
     }

     func TestGetConfig(t *testing.T) {
         provider := config.NewFileConfigProvider()
         cfg, _ := provider.LoadConfig("testdata/valid_config.json")
         value := provider.GetConfig("transport.timeout")
         assert.Equal(t, 30, value)
     }
     ```
2. **集成测试**：
   - 创建测试程序，使用 `config` 包加载配置并传递给其他包（如 `transport`），验证配置在实际场景中的正确性。
   - **实现**：
     ```go
     func TestConfigIntegration(t *testing.T) {
         provider := config.NewFileConfigProvider()
         cfg, err := provider.LoadConfig("testdata/valid_config.json")
         assert.NoError(t, err)

         // 模拟 transport 包使用配置
         transportCfg := cfg.GetTransportConfig()
         assert.Equal(t, 30, transportCfg.Timeout)
     }
     ```
3. **边界测试**：
   - 测试空配置文件、超大配置文件（>1MB）、非法字符等场景。
4. **性能测试**：
   - 使用 `go test -bench` 测量 `LoadConfig` 的执行时间，确保在合理范围内（如 <1ms）。
   - **实现**：
     ```go
     func BenchmarkLoadConfig(b *testing.B) {
         provider := config.NewFileConfigProvider()
         for i := 0; i < b.N; i++ {
             provider.LoadConfig("testdata/valid_config.json")
         }
     }
     ```

### 测试工具

- `testing`：Go 标准测试库。
- `github.com/stretchr/testify/assert`：断言库。
- Mock 文件：创建测试用的 JSON/YAML 文件。

---

## 2. `internal/types` 测试计划

### 测试要求

- **正确性**：验证结构体字段的定义和初始化。
- **兼容性**：测试结构体在序列化/反序列化中的一致性。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **Message 结构体**：验证 `Message.ID` 和 `Message.Content` 字段初始化。
     - **Request 结构体**：确保 `Request.Method` 和 `Request.Body` 字段正确。
     - **Response 结构体**：验证 `Response.Status` 和 `Response.Body` 字段。
     - **序列化/反序列化**：将结构体序列化为 JSON/Protobuf，验证反序列化后字段一致。
   - **实现**：
     ```go
     package types_test

     import (
         "testing"
         "encoding/json"
         "github.com/stretchr/testify/assert"
         "neo/internal/types"
     )

     func TestMessageStruct(t *testing.T) {
         msg := types.Message{ID: "123", Content: []byte("测试")}
         assert.Equal(t, "123", msg.ID)
         assert.Equal(t, []byte("测试"), msg.Content)

         // 测试 JSON 序列化
         data, err := json.Marshal(msg)
         assert.NoError(t, err)

         var deserialized types.Message
         err = json.Unmarshal(data, &deserialized)
         assert.NoError(t, err)
         assert.Equal(t, msg, deserialized)
     }

     func TestRequestStruct(t *testing.T) {
         req := types.Request{Method: "GET", Body: []byte("请求")}
         assert.Equal(t, "GET", req.Method)
         assert.Equal(t, []byte("请求"), req.Body)
     }

     func TestResponseStruct(t *testing.T) {
         resp := types.Response{Status: 200, Body: []byte("响应")}
         assert.Equal(t, 200, resp.Status)
         assert.Equal(t, []byte("响应"), resp.Body)
     }
     ```
2. **集成测试**：
   - 在 `protocol` 和 `transport` 包中使用 `types` 结构体，验证其在真实场景中的正确性。
   - **实现**：
     ```go
     func TestTypesIntegration(t *testing.T) {
         msg := types.Message{ID: "123", Content: []byte("测试")}
         data, err := json.Marshal(msg)
         assert.NoError(t, err)

         // 模拟 protocol 包使用
         var decoded types.Message
         err = json.Unmarshal(data, &decoded)
         assert.NoError(t, err)
         assert.Equal(t, msg, decoded)
     }
     ```
3. **边界测试**：
   - 测试空字段、超大负载或无效数据（如超大 `Content`）。
4. **性能测试**：
   - 测量大数据集的序列化/反序列化性能。
   - **实现**：
     ```go
     func BenchmarkMessageSerialization(b *testing.B) {
         msg := types.Message{ID: "123", Content: make([]byte, 1024)}
         for i := 0; i < b.N; i++ {
             json.Marshal(msg)
         }
     }
     ```

### 测试工具

- `testing`：Go 标准测试库。
- `github.com/stretchr/testify/assert`：断言库。
- `encoding/json`：用于 JSON 序列化测试。
- `google.golang.org/protobuf`：用于 Protobuf 序列化（若使用）。

---

## 3. `internal/utils` 测试计划

### 测试要求

- **正确性**：验证日志输出和字符串处理的正确性。
- **边界条件**：测试空输入或异常输入。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **日志函数**：验证 `Info` 和 `Error` 产生预期日志输出。
     - **字符串函数**：测试 `Format` 在不同输入下的正确格式化。
     - **空输入**：测试日志和字符串函数在空或无效输入下的表现。
   - **实现**：
     ```go
     package utils_test

     import (
         "testing"
         "bytes"
         "github.com/stretchr/testify/assert"
         "neo/internal/utils"
     )

     func TestLogInfo(t *testing.T) {
         var buf bytes.Buffer
         utils.SetLogOutput(&buf)
         utils.Info("测试消息 %s", "参数1")
         assert.Contains(t, buf.String(), "INFO: 测试消息 参数1")
     }

     func TestStringFormat(t *testing.T) {
         result := utils.Format("你好 %s", "世界")
         assert.Equal(t, "你好 世界", result)

         // 边界情况：空输入
         result = utils.Format("", "测试")
         assert.Equal(t, "", result)
     }
     ```
2. **集成测试**：
   - 验证日志输出在更大系统上下文（如 `core` 包）中的正确捕获。
   - **实现**：
     ```go
     func TestUtilsIntegration(t *testing.T) {
         var buf bytes.Buffer
         utils.SetLogOutput(&buf)
         // 模拟 core 包使用
         utils.Error("core 包中的错误")
         assert.Contains(t, buf.String(), "ERROR: core 包中的错误")
     }
     ```
3. **边界测试**：
   - 测试日志函数在超大字符串或无效格式说明符下的表现。
   - 测试字符串函数在格式错误的模板下的表现。
4. **性能测试**：
   - 在高负载下测试日志和字符串格式化的性能。
   - **实现**：
     ```go
     func BenchmarkStringFormat(b *testing.B) {
         for i := 0; i < b.N; i++ {
             utils.Format("测试 %s %d", "字符串", i)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `bytes.Buffer`：捕获日志输出以进行验证。
- `github.com/stretchr/testify/assert`：断言库。

---

## 4. `internal/protocol` 测试计划

### 测试要求

- **正确性**：验证编码和解码的对称性。
- **兼容性**：测试不同协议版本的兼容性。
- **性能**：验证编码和解码的效率。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **IPC 编码/解码**：测试 `Encode` 和 `Decode` 在 IPC 协议下对 `types.Message` 的处理。
     - **HTTP 编码/解码**：测试 HTTP 协议的 JSON 序列化。
     - **版本兼容性**：使用不同版本头部的消息，确保向后兼容。
     - **无效数据**：测试解码损坏或不完整的数据。
   - **实现**：
     ```go
     package protocol_test

     import (
         "testing"
         "context"
         "github.com/stretchr/testify/assert"
         "neo/internal/protocol"
         "neo/internal/types"
     )

     func TestIPCCodec(t *testing.T) {
         codec := protocol.NewCodec("ipc")
         msg := types.Message{ID: "123", Content: []byte("测试")}
         ctx := context.Background()

         data, err := codec.Encode(ctx, msg)
         assert.NoError(t, err)

         decoded, err := codec.Decode(ctx, data)
         assert.NoError(t, err)
         assert.Equal(t, msg, decoded)
     }

     func TestInvalidData(t *testing.T) {
         codec := protocol.NewCodec("ipc")
         ctx := context.Background()
         _, err := codec.Decode(ctx, []byte("无效"))
         assert.Error(t, err)
     }
     ```
2. **集成测试**：
   - 在 `transport` 包中测试协议编码/解码，确保消息端到端流动。
   - **实现**：
     ```go
     func TestProtocolIntegration(t *testing.T) {
         codec := protocol.NewCodec("http")
         msg := types.Message{ID: "123", Content: []byte("测试")}
         ctx := context.Background()

         data, err := codec.Encode(ctx, msg)
         assert.NoError(t, err)

         // 模拟传输层
         decoded, err := codec.Decode(ctx, data)
         assert.NoError(t, err)
         assert.Equal(t, msg, decoded)
     }
     ```
3. **边界测试**：
   - 测试超大消息、空消息或格式错误的头部。
4. **性能测试**：
   - 测试大消息的编码/解码性能。
   - **实现**：
     ```go
     func BenchmarkIPCCodec(b *testing.B) {
         codec := protocol.NewCodec("ipc")
         msg := types.Message{ID: "123", Content: make([]byte, 1024)}
         ctx := context.Background()
         for i := 0; i < b.N; i++ {
             codec.Encode(ctx, msg)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `context`：用于上下文相关测试。

---

## 5. `internal/transport/conn` 测试计划

### 测试要求

- **正确性**：验证连接获取和释放的正确性。
- **并发性**：测试连接池在高并发下的表现。
- **异常处理**：测试连接失败或中断的处理。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **连接获取**：测试 `GetConnection` 返回有效连接。
     - **连接释放**：验证 `ReleaseConnection` 正确将连接返回池中。
     - **无效目标**：测试 `GetConnection` 在无效地址下返回错误。
   - **实现**：
     ```go
     package conn_test

     import (
         "testing"
         "context"
         "github.com/stretchr/testify/assert"
         "neo/internal/transport/conn"
     )

     func TestConnectionPool(t *testing.T) {
         pool := conn.NewConnectionPool()
         ctx := context.Background()

         c, err := pool.GetConnection(ctx, "localhost:28080")
         assert.NoError(t, err)
         assert.NotNil(t, c)

         pool.ReleaseConnection(c)
         // 验证连接可重用
         c2, err := pool.GetConnection(ctx, "localhost:28080")
         assert.NoError(t, err)
         assert.NotNil(t, c2)
     }

     func TestInvalidTarget(t *testing.T) {
         pool := conn.NewConnectionPool()
         ctx := context.Background()
         _, err := pool.GetConnection(ctx, "无效:地址")
         assert.Error(t, err)
     }
     ```
2. **集成测试**：
   - 测试连接池与 `transport/codec` 结合，验证消息发送和接收。
   - **实现**：
     ```go
     func TestConnIntegration(t *testing.T) {
         pool := conn.NewConnectionPool()
         ctx := context.Background()
         c, err := pool.GetConnection(ctx, "localhost:28080")
         assert.NoError(t, err)

         // 模拟发送数据
         err = c.Send(ctx, []byte("测试"))
         assert.NoError(t, err)

         data, err := c.Receive(ctx)
         assert.NoError(t, err)
         assert.NotEmpty(t, data)
     }
     ```
3. **边界测试**：
   - 测试最大池大小、无效连接类型或超时情况。
4. **性能测试**：
   - 在高并发下测试连接获取/释放性能。
   - **实现**：
     ```go
     func BenchmarkConnectionPool(b *testing.B) {
         pool := conn.NewConnectionPool()
         ctx := context.Background()
         for i := 0; i < b.N; i++ {
             c, _ := pool.GetConnection(ctx, "localhost:28080")
             pool.ReleaseConnection(c)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `net`：用于模拟 TCP/Unix Socket 连接。

---

## 6. `internal/transport/codec` 测试计划

### 测试要求

- **正确性**：验证编码/解码的准确性。
- **协议切换**：测试不同协议（如 HTTP、IPC）的切换功能。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **编码器创建**：测试 `NewCodec` 创建指定协议的编码器。
     - **编码/解码**：验证 `Encode` 和 `Decode` 与 `protocol` 包的协作。
     - **无效协议**：测试 `NewCodec` 在不支持的协议下返回错误。
   - **实现**：
     ```go
     package codec_test

     import (
         "testing"
         "context"
         "github.com/stretchr/testify/assert"
         "neo/internal/transport/codec"
         "neo/internal/types"
     )

     func TestCodecCreation(t *testing.T) {
         c := codec.NewCodec("http")
         assert.NotNil(t, c)

         c = codec.NewCodec("无效")
         assert.Nil(t, c)
     }

     func TestCodecEncodeDecode(t *testing.T) {
         c := codec.NewCodec("http")
         ctx := context.Background()
         msg := types.Message{ID: "123", Content: []byte("测试")}

         data, err := c.Encode(ctx, msg)
         assert.NoError(t, err)

         decoded, err := c.Decode(ctx, data)
         assert.NoError(t, err)
         assert.Equal(t, msg, decoded)
     }
     ```
2. **集成测试**：
   - 测试编码器与 `transport/conn` 结合，确保消息端到端流动。
   - **实现**：
     ```go
     func TestCodecIntegration(t *testing.T) {
         c := codec.NewCodec("ipc")
         ctx := context.Background()
         msg := types.Message{ID: "123", Content: []byte("测试")}

         data, err := c.Encode(ctx, msg)
         assert.NoError(t, err)

         // 模拟传输层
         decoded, err := c.Decode(ctx, data)
         assert.NoError(t, err)
         assert.Equal(t, msg, decoded)
     }
     ```
3. **边界测试**：
   - 测试超大消息或损坏的数据。
4. **性能测试**：
   - 测试大消息的编码/解码性能。
   - **实现**：
     ```go
     func BenchmarkCodecEncode(b *testing.B) {
         c := codec.NewCodec("http")
         msg := types.Message{ID: "123", Content: make([]byte, 1024)}
         ctx := context.Background()
         for i := 0; i < b.N; i++ {
             c.Encode(ctx, msg)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `github.com/stretchr/testify/mock`：用于模拟 `protocol` 包。

---

## 7. `internal/transport/retry` 测试计划

### 测试要求

- **正确性**：验证重试策略的触发条件。
- **边界条件**：测试最大重试次数和间隔。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **重试成功**：测试 `Execute` 在初始失败后重试并成功。
     - **重试失败**：测试 `Execute` 在最大重试次数后失败。
     - **超时**：测试上下文超时下的重试行为。
   - **实现**：
     ```go
     package retry_test

     import (
         "testing"
         "context"
         "errors"
         "github.com/stretchr/testify/assert"
         "neo/internal/transport/retry"
         "neo/internal/config"
     )

     func TestRetryPolicy(t *testing.T) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         policy := retry.NewRetryPolicy(cfg)
         ctx := context.Background()

         var attempts int
         operation := func() error {
             attempts++
             if attempts < 3 {
                 return errors.New("临时失败")
             }
             return nil
         }

         err := policy.Execute(ctx, operation)
         assert.NoError(t, err)
         assert.Equal(t, 3, attempts)
     }

     func TestRetryFailure(t *testing.T) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         policy := retry.NewRetryPolicy(cfg)
         ctx := context.Background()

         operation := func() error {
             return errors.New("持续失败")
         }

         err := policy.Execute(ctx, operation)
         assert.Error(t, err)
     }
     ```
2. **集成测试**：
   - 测试重试策略与 `transport/conn` 在实际失败场景下的表现。
   - **实现**：
     ```go
     func TestRetryIntegration(t *testing.T) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         policy := retry.NewRetryPolicy(cfg)
         ctx := context.Background()

         // 模拟传输失败
         operation := func() error { return errors.New("连接失败") }
         err := policy.Execute(ctx, operation)
         assert.Error(t, err)
     }
     ```
3. **边界测试**：
   - 测试零次重试、最大重试次数或极短间隔。
4. **性能测试**：
   - 测试重复失败下的重试性能。
   - **实现**：
     ```go
     func BenchmarkRetryPolicy(b *testing.B) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         policy := retry.NewRetryPolicy(cfg)
         ctx := context.Background()
         operation := func() error { return errors.New("失败") }
         for i := 0; i < b.N; i++ {
             policy.Execute(ctx, operation)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `context`：用于超时测试。

---

## 8. `internal/registry` 测试计划

### 测试要求

- **正确性**：验证服务注册和发现的准确性。
- **并发性**：测试高并发下的注册和发现。
- **异常处理**：测试不可用服务场景。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **服务注册**：测试 `RegisterService` 正确添加服务到注册中心。
     - **服务发现**：测试 `DiscoverService` 返回正确实例。
     - **无效服务**：测试注册无效服务名称或地址。
   - **实现**：
     ```go
     package registry_test

     import (
         "testing"
         "context"
         "github.com/stretchr/testify/assert"
         "neo/internal/registry"
         "neo/internal/config"
     )

     func TestRegisterService(t *testing.T) {
         cfg := config.Config{}
         reg := registry.NewServiceRegistry(cfg)
         ctx := context.Background()

         instance := registry.ServiceInstance{Name: "测试服务", Address: "localhost:28080"}
         err := reg.RegisterService(ctx, instance)
         assert.NoError(t, err)

         instances, err := reg.DiscoverService(ctx, "测试服务")
         assert.NoError(t, err)
         assert.Contains(t, instances, instance)
     }

     func TestInvalidService(t *testing.T) {
         cfg := config.Config{}
         reg := registry.NewServiceRegistry(cfg)
         ctx := context.Background()

         instance := registry.ServiceInstance{Name: "", Address: "localhost:28080")
         err := reg.RegisterService(ctx, instance)
         assert.Error(t, err)
     }
     ```
2. **集成测试**：
   - 测试注册中心与 `core` 包结合，确保服务可被发现。
   - **实现**：
     ```go
     func TestRegistryIntegration(t *testing.T) {
         cfg := config.Config{}
         reg := registry.NewServiceRegistry(cfg)
         ctx := context.Background()

         instance := registry.ServiceInstance{Name: "测试服务", Address: "localhost:28080"}
         err := reg.RegisterService(ctx, instance)
         assert.NoError(t, err)

         instances, err := reg.DiscoverService(ctx, "测试服务")
         assert.NoError(t, err)
         assert.Len(t, instances, 1)
     }
     ```
3. **边界测试**：
   - 测试重复注册或不存在的服务名称。
4. **性能测试**：
   - 在高并发下测试注册/发现性能。
   - **实现**：
     ```go
     func BenchmarkDiscoverService(b *testing.B) {
         cfg := config.Config{}
         reg := registry.NewServiceRegistry(cfg)
         ctx := context.Background()
         instance := registry.ServiceInstance{Name: "测试服务", Address: "localhost:28080"}
         reg.RegisterService(ctx, instance)

         for i := 0; i < b.N; i++ {
             reg.DiscoverService(ctx, "测试服务")
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `sync`：用于并发测试。

---

## 9. `internal/core` 测试计划

### 测试要求

- **正确性**：验证请求处理的逻辑正确性。
- **异常处理**：测试无效请求的处理。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **请求处理**：测试 `HandleRequest` 正确处理有效请求。
     - **错误处理**：测试无效或格式错误的请求返回适当错误。
   - **实现**：
     ```go
     package core_test

     import (
         "testing"
         "context"
         "github.com/stretchr/testify/assert"
         "github.com/stretchr/testify/mock"
         "neo/internal/core"
         "neo/internal/transport"
         "neo/internal/registry"
         "neo/internal/types"
     )

     type mockTransport struct{ mock.Mock }
     type mockRegistry struct{ mock.Mock }

     func (m *mockTransport) Send(ctx context.Context, msg []byte) error { return m.Called(ctx, msg).Error(0) }
     func (m *mockRegistry) DiscoverService(ctx context.Context, name string) ([]registry.ServiceInstance, error) {
         args := m.Called(ctx, name)
         return args.Get(0).([]registry.ServiceInstance), args.Error(1)
     }

     func TestHandleRequest(t *testing.T) {
         tMock := new(mockTransport)
         rMock := new(mockRegistry)
         svc := core.NewService(tMock, rMock)
         ctx := context.Background()
         req := types.Request{Method: "GET", Body: []byte("测试")}

         rMock.On("DiscoverService", ctx, mock.Anything).Return([]registry.ServiceInstance{{Name: "测试服务", Address: "localhost:28080"}}, nil)
         tMock.On("Send", ctx, mock.Anything).Return(nil)

         resp, err := svc.HandleRequest(ctx, req)
         assert.NoError(t, err)
         assert.Equal(t, 200, resp.Status)
     }
     ```
2. **集成测试**：
   - 使用真实的 `transport` 和 `registry` 实现测试 `HandleRequest`。
   - **实现**：
     ```go
     func TestCoreIntegration(t *testing.T) {
         // 设置真实的 transport 和 registry
         cfg := config.Config{}
         reg := registry.NewServiceRegistry(cfg)
         tConn := conn.NewConnectionPool()
         tCodec := codec.NewCodec("http")
         svc := core.NewService(/* 真实传输层 */, reg)
         ctx := context.Background()
         req := types.Request{Method: "GET", Body: []byte("测试")}

         resp, err := svc.HandleRequest(ctx, req)
         assert.NoError(t, err)
         assert.NotNil(t, resp)
     }
     ```
3. **边界测试**：
   - 测试空请求或无效方法。
4. **性能测试**：
   - 测试高负载下的请求处理性能。
   - **实现**：
     ```go
     func BenchmarkHandleRequest(b *testing.B) {
         tMock := new(mockTransport)
         rMock := new(mockRegistry)
         svc := core.NewService(tMock, rMock)
         ctx := context.Background()
         req := types.Request{Method: "GET", Body: []byte("测试")}

         rMock.On("DiscoverService", ctx, mock.Anything).Return([]registry.ServiceInstance{{Name: "测试服务", Address: "localhost:28080"}}, nil)
         tMock.On("Send", ctx, mock.Anything).Return(nil)

         for i := 0; i < b.N; i++ {
             svc.HandleRequest(ctx, req)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `github.com/stretchr/testify/mock`：用于模拟依赖。

---

## 10. `cmd` 测试计划

### 测试要求

- **正确性**：验证启动和关闭流程的完整性。
- **健壮性**：测试异常情况下的关闭行为。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **启动**：测试 `main` 正确初始化依赖。
     - **关闭**：测试 `shutdown` 优雅关闭资源。
   - **实现**：
     ```go
     package main_test

     import (
         "testing"
         "github.com/stretchr/testify/assert"
         "neo/cmd"
     )

     func TestMain(t *testing.T) {
         // 模拟依赖以避免真实初始化
         assert.NotPanics(t, func() { cmd.Main() })
     }

     func TestShutdown(t *testing.T) {
         // 模拟关闭资源
         assert.NoError(t, cmd.Shutdown())
     }
     ```
2. **集成测试**：
   - 使用测试配置运行整个应用程序，验证启动和关闭流程。
   - **实现**：
     ```go
     func TestCmdIntegration(t *testing.T) {
         // 在 goroutine 中运行 main 并模拟关闭
         go cmd.Main()
         // 等待启动后触发关闭
         assert.NoError(t, cmd.Shutdown())
     }
     ```
3. **边界测试**：
   - 测试无效配置或缺失依赖的启动行为。
4. **性能测试**：
   - 测量启动时间，确保在可接受范围内。
   - **实现**：
     ```go
     func BenchmarkStartup(b *testing.B) {
         for i := 0; i < b.N; i++ {
             go cmd.Main()
             cmd.Shutdown()
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。

---

## 11. `pkg` 测试计划

### 测试要求

- **正确性**：验证 API 调用的功能。
- **兼容性**：测试 API 的向后兼容性。
- **性能**：验证 API 的响应时间。

### 测试步骤

1. **单元测试**：
   - **测试用例**：
     - **客户端创建**：测试 `NewClient` 正确初始化。
     - **API 调用**：测试 `Call` 发送请求并接收响应。
     - **错误处理**：测试无效请求或服务器错误。
   - **实现**：
     ```go
     package pkg_test

     import (
         "testing"
         "context"
         "github.com/stretchr/testify/assert"
         "neo/pkg"
         "neo/internal/types"
     )

     func TestNewClient(t *testing.T) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         client := pkg.NewClient(cfg)
         assert.NotNil(t, client)
     }

     func TestCall(t *testing.T) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         client := pkg.NewClient(cfg)
         ctx := context.Background()
         req := types.Request{Method: "GET", Body: []byte("测试")}

         resp, err := client.Call(ctx, req)
         assert.NoError(t, err)
         assert.Equal(t, 200, resp.Status)
     }
     ```
2. **集成测试**：
   - 使用真实服务器设置测试 API 调用，验证端到端功能。
   - **实现**：
     ```go
     func TestAPIIntegration(t *testing.T) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         client := pkg.NewClient(cfg)
         ctx := context.Background()
         req := types.Request{Method: "GET", Body: []byte("测试")}

         // 假设测试服务器正在运行
         resp, err := client.Call(ctx, req)
         assert.NoError(t, err)
         assert.NotNil(t, resp)
     }
     ```
3. **边界测试**：
   - 测试大请求、无效配置或服务器超时。
4. **性能测试**：
   - 测试高负载下的 API 调用性能。
   - **实现**：
     ```go
     func BenchmarkAPICall(b *testing.B) {
         cfg := config.Config{Transport: config.TransportConfig{Timeout: 30}}
         client := pkg.NewClient(cfg)
         ctx := context.Background()
         req := types.Request{Method: "GET", Body: []byte("测试")}

         for i := 0; i < b.N; i++ {
             client.Call(ctx, req)
         }
     }
     ```

### 测试工具

- `testing`：用于单元和性能测试。
- `github.com/stretchr/testify/assert`：断言库。
- `net/http/httptest`：用于模拟服务器响应。

---

## 额外测试注意事项

- **覆盖率分析**：运行 `go test -cover`，确保每个包的测试覆盖率至少达到 90%。
- **模拟依赖**：使用 `testify/mock` 在单元测试中隔离包的依赖。
- **错误注入**：模拟失败（如网络错误、超时）以测试健壮性。
- **持续集成**：设置 CI 流水线（如 GitHub Actions），在代码变更时自动运行测试。

## 下一步

1. **执行测试**：按照 `ProjectRefactorPlan.md` 的顺序，从 `internal/config` 开始逐一测试每个包。
2. **审查结果**：分析测试失败和覆盖率报告，识别差距。
3. **迭代改进**：修复测试中发现的问题并重新运行测试，确保修复有效。
4. **记录问题**：在问题跟踪系统中记录任何错误或性能问题，以供未来参考。

---

**当前日期和时间**：2025年7月10日星期四，东京时间上午10:54。
