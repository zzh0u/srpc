# srpc - gRPC 微服务系统

一个完整的 gRPC 微服务系统，包含服务端和客户端，设计用于模拟生产环境的多机部署。项目使用 Go 工作区管理多个模块，并通过 Docker Compose 进行容器化部署。

## 功能特性

### 客户端特性
- 长期运行：作为主进程运行
- 优雅终止：捕获 `SIGTERM` 信号处理
- 定时驱动：基于固定时间间隔发起请求
- 结构化日志：JSON 格式日志输出
- 指标收集：请求统计、成功率、平均耗时
- 熔断器：`CircuitBreaker` 实现熔断机制
- 连接管理：长连接复用、健康检查、重连策略
- 压缩支持：支持 Snappy 压缩算法，减少网络传输数据量
- 请求追踪：为每个请求生成唯一 ID，便于分布式追踪

### 服务端特性
- 四种流模式：完整实现 gRPC 的四种通信模式
- 优雅关闭：捕获 `SIGINT` 和 `SIGTERM` 信号
- 简单日志：使用标准 slog 包
- 请求追踪：支持从 metadata 中读取请求 ID 并记录到日志

### 容器化部署
- 自定义网络：`srpc-network` 桥接网络
- 健康检查：客户端容器使用进程检查
- 资源限制：CPU 和内存限制配置
- 重启策略：`restart: unless-stopped`
- 服务发现：通过服务名 `grpc-server:50051` 访问
- 多实例支持：客户端支持通过 `scale` 配置扩展多个实例

## 架构概述

### 项目结构
```
srpc/
├── client/                    # 客户端模块（长期运行的服务）
├── server/                    # 服务端模块
├── proto/                     # Protobuf 定义
├── pkg/                       # 公共工具包
│   ├── compress/              # 压缩算法实现
│   ├── log/                   # 日志工具
│   └── tools/                 # 工具函数
├── go.mod                     # 根模块定义
├── go.work                    # Go 工作区配置
├── docker-compose.yml         # Docker Compose 编排配置
├── LICENSE                    # MIT 许可证
└── CLAUDE.md                  # Claude Code 项目指南
```

### 模块架构
项目使用 Go 工作区管理三个模块：
1. 根模块 (`srpc`)：包含 proto 定义和公共依赖
2. 客户端模块 (`srpc/client`)：长期运行的 gRPC 客户端服务
3. 服务端模块 (`srpc/server`)：gRPC 服务器

模块间通过 `replace` 指令和 Go 工作区进行依赖管理。

## 快速开始

### 前提条件
- Go 1.25.5+
- Docker & Docker Compose
- protoc 编译器（可选，用于重新生成 proto 代码）

### Docker 运行
```bash
# 克隆项目
git clone https://github.com/zzh0u/srpc
cd srpc

# 构建并启动所有服务
docker-compose up --build

# 在后台运行
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

## 配置说明

### 客户端环境变量
- `GRPC_SERVER_ADDR`: gRPC 服务器地址（默认: `grpc-server:50051`）
- `REQUEST_INTERVAL_SEC`: 请求间隔秒数（默认: 30）
- `MAX_RETRIES`: 最大重试次数（默认: 3）
- `MAX_CONCURRENT_REQUESTS`: 最大并发请求数（默认: 5）
- `JITTER_PERCENT`: 抖动百分比，避免请求同步（默认: 10）
- `KEEP_ALIVE_SEC`: 连接保活时间（默认: 20）
- `ENABLE_COMPRESSION`: 是否启用压缩（默认: `true`）
- `COMPRESSION_TYPE`: 压缩类型（默认: `snappy`）
- `GENERATE_REQUEST_ID`: 是否为每个请求生成唯一 ID（默认: `true`）
- `TZ`: 时区设置（默认: UTC）

### 服务端环境变量
- `TZ`: 时区设置（默认: UTC）

## gRPC 服务接口

定义在 `proto/helloworld.proto` 中的服务：
- `SayHello`: 普通 RPC
- `GetStream`: 服务端流模式
- `PutStream`: 客户端流模式
- `AllStream`: 双向流模式

### 重新生成 proto 代码
```bash
# 需要安装 protoc 和 protoc-gen-go 插件
protoc --go_out=. --go-grpc_out=. proto/helloworld.proto
```

## 开发指南

### 添加新功能
1. 添加新的 gRPC 方法：修改 `proto/helloworld.proto` 并重新生成代码
2. 添加配置：通过环境变量扩展客户端配置
3. 添加监控：扩展客户端的指标收集功能
4. 多实例部署：修改 `docker-compose.yml` 中的 `scale` 配置

### 开发注意事项
1. 模块依赖：客户端和服务端都依赖根模块的 proto 定义
2. 工作区使用：使用 `go.work` 进行多模块开发
3. 环境变量：客户端配置通过环境变量注入
4. Docker 构建：使用多阶段构建生成精简镜像
5. 连接管理：客户端实现连接池和健康检查
6. 错误处理：客户端包含完整的重试和熔断逻辑

## 技术栈

### 核心依赖
- Go: 1.25.5
- gRPC: 1.78.0
- Protocol Buffers: 1.36.11
- Snappy 压缩: 0.0.4

### 网络配置
- gRPC 端口: 50051
- 网络名称: srpc-network
- 服务发现: 通过 Docker 服务名访问

## 许可证

本项目基于 MIT 许可证开源。

## 待实现功能

详见 [todo.md](todo.md) 文件：
1. 事件驱动触发：支持 Kafka/RabbitMQ/Webhook 等外部事件源触发请求
2. 负载均衡：多服务端实例场景下的负载均衡策略
3. 高级资源管理：请求队列、连接池控制