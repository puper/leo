# 项目导航 (Project Map)

## 项目概述

Leo 是一个 Go 语言微服务组件框架，提供依赖注入和组件生命周期管理。核心是 `engine` 包，通过有向无环图 (DAG) 实现组件的拓扑排序和按序构建/关闭。

## 快速开始

- **查看组件示例**: `components/db/component.go`
- **了解 Engine 机制**: `engine/engine.go`
- **构建命令**: `go build ./...`

## 代码结构

```
├── engine/          # 核心引擎：依赖注入与生命周期管理
├── components/       # 组件集合
│   ├── db/          # MySQL/GORM 数据库组件（支持主从）
│   ├── etcd/        # etcd 配置中心客户端
│   ├── grpc/        # gRPC 服务端/客户端
│   ├── influxdb/    # 时序数据库客户端
│   ├── iris/        # Web 框架 (Iris)
│   ├── nats/        # NATS 消息队列
│   ├── rabbitmq/    # RabbitMQ 消息队列
│   ├── restyclient/ # HTTP 客户端
│   ├── storage/     # 文件存储
│   ├── uniqid/       # 分布式 ID 生成
│   └── zaplog/      # 日志组件
└── pkg/             # 公共工具包
    ├── mutexmanager # 互斥锁管理器
    ├── reconnectable # 可重连连接
    └── timewheel    # 时间轮定时器
```

## 关键约定

- **默认语言**: 本项目所有 AI 回复、代码注释、文档均使用**简体中文**
- **组件构建模式**: 每个组件提供 `Builder` 函数，符合 `engine.Builder` 接口签名
- **配置模式**: 使用 `github.com/spf13/viper` 进行配置管理

## 命令参考

| 命令 | 用途 |
|------|------|
| `go build ./...` | 构建所有包 |
| `go test ./...` | 运行测试 |
| `go mod tidy` | 整理依赖 |

## 文档索引

- [架构设计](docs/design/architecture.md)
- [组件文档](components/)
- [工作流规范](docs/workflows/)

## 变更守卫

- 提交前运行 `go build ./...` 和 `go vet ./...`
- 确保无循环依赖（Engine 的 DAG 会检测）
- 遵循 `engine.Builder` 签名规范

## 验证清单

- [ ] `go build ./...` 成功
- [ ] 新增组件符合 `Builder` 模式
- [ ] 配置通过 viper 读取
- [ ] 中文注释覆盖关键逻辑
