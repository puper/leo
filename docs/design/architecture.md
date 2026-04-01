# 架构设计 (Architecture Design)

## 项目定位

Leo 是一个 Go 语言微服务组件框架，核心目标是**组件的依赖注入与生命周期管理**。

## 核心机制: Engine

Engine 通过**有向无环图 (DAG)** 实现组件的拓扑排序，确保：
1. 按依赖顺序构建组件
2. 按逆序关闭组件

### 关键接口

```go
type Builder func() (any, error)

type Closer interface {
    Close() error
}
```

### 生命周期

1. **Register**: 注册组件名称、构建器、依赖关系
2. **Build**: 按拓扑序构建所有组件
3. **Close**: 按逆拓扑序关闭所有组件

## 组件模式

每个组件提供 `Builder` 函数，符合 `engine.Builder` 接口签名：

```go
func Builder(cfg *Config, configurers ...func(*Component) error) engine.Builder {
    return func() (any, error) {
        // 创建实例
        // 应用配置器
        return instance, nil
    }
}
```

## 目录结构

```
├── engine/          # 核心引擎包
├── components/      # 组件集合
│   ├── db/          # 数据库（主从）
│   ├── etcd/        # 配置中心
│   ├── grpc/        # RPC
│   ├── influxdb/    # 时序数据库
│   ├── iris/        # Web 框架
│   ├── nats/         # 消息队列
│   ├── rabbitmq/    # 消息队列
│   ├── restyclient/ # HTTP 客户端
│   ├── storage/     # 文件存储
│   ├── uniqid/      # ID 生成
│   └── zaplog/      # 日志
└── pkg/             # 公共工具
    ├── mutexmanager
    └── timewheel
```

## 配置管理

使用 `github.com/spf13/viper` 进行配置管理，典型模式：
- 通过 `engine.New(config)` 传入配置
- 各组件从配置中读取对应节
