### 协议模块迁移与目录关系规划 一、现有 transport 目录文件归属分析
根据代码功能与协议关联性，建议分类如下：

文件 归属模块 迁移目标路径 说明 tcp_server.go TCP协议 connection/tcp/server.go 明确TCP服务实现 connection_handler.go TCP协议 connection/tcp/handler.go 仅处理TCP连接生命周期 protocol_handler.go IPC协议 ipcprotocol/handler.go 独立为IPC协议包（跨传输层） service_registry.go 共享服务 transport/service_registry.go 保留在transport层 worker_pool.go 共享资源 transport/worker_pool.go 多协议共享协程池 metrics_collector.go 共享监控 metrics/collector.go 迁移至全局metrics包 error_handler.go 协议专用 拆分至各协议目录 TCP/HTTP错误响应格式不同 server.go 启动入口 transport/ipc_server.go 重命名明确为IPC服务入口
 二、 transport 与 connection 目录职能划分
```
internal/
├── transport/          # 传输层协调中心
│   ├── ipc_server.go   # IPC服务启动入口（原server.go）
│   ├── http_server.go  # HTTP服务启动入口
│   ├── service_registry.go # 跨协议服务注册
│   └── worker_pool.go  # 共享协程池
│
├── connection/         # 连接层实现
│   ├── tcp/            # TCP协议族
│   │   ├── server.go   # TCP监听与接受
│   │   ├── handler.go  # TCP连接处理
│   │   └── codec.go    # TCP协议编解码
│   │
│   ├── http/           # HTTP协议族
│   │   ├── server.go   # HTTP服务实现
│   │   ├── router.go   # 路由系统
│   │   └── middleware.go # 中间件链
│   │
│   └── common.go       # 连接共享工具（超时控制/流量统计）
│
└── ipcprotocol/        # IPC协议定义（独立于传输层）
    ├── protocol.go     # 消息结构定义
    └── handler.go      # 消息解析逻辑
``` 三、迁移实施步骤
1. 创建目标目录结构 ：
   
   ```
   mkdir -p neo/internal/connection/{tcp,http}
   ```
2. TCP模块迁移 ：
   
   ```
   # 移动TCP相关文件
   mv neo/internal/transport/tcp_server.go neo/internal/
   connection/tcp/server.go
   mv neo/internal/transport/connection_handler.go neo/internal/
   connection/tcp/handler.go
   
   # 调整包声明
   sed -i 's/package transport/package tcp/' neo/internal/
   connection/tcp/*.go
   ```
3. 依赖关系调整 ：
   
   - TCPServer 构造函数需从 transport 包迁移至 connection/tcp 包
   - 原 transport.StartIpcServer() 需修改为调用 connection/tcp.NewServer()
   - 服务注册逻辑保持在 transport 层，通过接口抽象隔离协议差异
4. 代码兼容性保障 ：
   
   - 保留 transport 包中原有导出函数（如 RegisterService ）作为兼容层
   - 使用特性开关控制新旧架构切换：
   ```
   // config.go
   type IPCConfig struct {
       // ...
       UseNewArchitecture bool `yaml:"use_new_architecture"`
   }
   ``` 四、关键架构优势
1. 关注点分离 ：
   
   - transport 层专注服务编排与资源调度
   - connection 层专注具体协议实现
   - ipcprotocol 层专注跨传输协议的消息格式
2. 演进兼容性 ：
   
   - 新协议（如WebSocket）可直接添加 connection/websocket 目录
   - IPC协议升级不影响传输层实现
   - 各协议可独立迭代优化
3. 符合项目既有规范 ：
   
   - 延续 discovery 、 metrics 等包的单一职责设计
   - 保持Go项目标准的目录分层习惯
   - 与Python-IPC模块的 pool / protocol 划分保持概念一致

## 六、后续优化计划（P1优先级）
1. 连接池隔离 ：为TCP/HTTP实现独立的连接池配置
2. 协议监控 ：添加协议维度的监控指标（延迟/错误率）
3. 配置热加载 ：支持HTTP服务器配置动态更新
4. TLS支持 ：实现HTTPS加密传输（基于HTTPConfig）