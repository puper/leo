# Reconnect 重连工具库

## 概述

通用的连接重连框架，支持多种 SDK（Redis、Etcd、NATS、RabbitMQ 等），提供标准化的连接管理、健康检查和优雅关闭。

**核心设计**：通过序列号机制避免多协程重复刷新，通过 Client wrapper 提供简洁的自动重试 API。

## 核心接口

### Connector 接口

```go
type Connector interface {
    Connect(ctx context.Context) error
    Disconnect() error
    IsConnected() bool
}
```

### ConfigProvider 接口（可选）

如果 Connector 实现此接口，自动使用其配置：

```go
type ConfigProvider interface {
    Config() ReconnectConfig
}
```

### ClientGetter 接口（可选）

如果 Connector 实现此接口，Client 可以获取原始连接：

```go
type ClientGetter interface {
    GetClient() interface{}
}
```

### ReconnectConfig 接口

每个字段都有独立的 Get 方法，可动态返回值：

```go
type ReconnectConfig interface {
    GetMaxRetries() int
    GetInitialInterval() time.Duration
    GetMaxInterval() time.Duration
    GetMultiplier() float64
    GetCloseTimeout() time.Duration
    GetHealthCheckInterval() time.Duration
}
```

### EventHandler 接口

```go
type EventHandler interface {
    OnConnected()
    OnDisconnected(err error)
    OnReconnecting(attempt int, delay time.Duration)
    OnError(err error)
}
```

## 核心设计

### 序列号机制

避免多个协程同时刷新连接：

- `clientSeq >= 0`：正常状态
- 刷新时原子递增 `clientSeq`
- 通过 CAS 确保只有一个协程执行刷新
- 其他协程通过对比序列号得知已被刷新

### Client Wrapper

封装原始连接，提供自动重试：

```go
client := comp.GetClient()

// 方式 1：自动重试
err := client.Do(func(raw interface{}) error {
    rdb := raw.(*redis.Client)
    return rdb.Get(ctx, "key").Err()
})

// 方式 2：手动获取原始连接
rdb := client.Raw().(*redis.Client)
```

### tryRefresh 机制

当操作失败时：
1. `tryRefresh` 通过 CAS 确保只有一个协程执行刷新
2. 断开连接并重建 context
3. `WaitReconnect` 等待重连完成
4. 其他协程通过序列号对比得知已被刷新

## 使用示例

### 基本用法

```go
type myConnector struct {
    addr   string
    client *redis.Client
}

func (c *myConnector) Connect(ctx context.Context) error {
    c.client = redis.NewClient(&redis.Options{Addr: c.addr})
    return c.client.Ping(ctx).Err()
}

func (c *myConnector) Disconnect() error {
    return c.client.Close()
}

func (c *myConnector) IsConnected() bool {
    return c.client.Ping(context.Background()).Err() == nil
}

func (c *myConnector) GetClient() interface{} {
    return c.client
}

func (c *myConnector) Config() reconnect.ReconnectConfig {
    return &reconnect.DefaultReconnectConfig{
        MaxRetries: -1,
    }
}

// 使用
connector := &myConnector{addr: "localhost:6379"}
comp := reconnect.New(connector, nil, nil)

comp.Start()
defer comp.Close()

client := comp.GetClient()
err := client.Do(func(raw interface{}) error {
    rdb := raw.(*redis.Client)
    return rdb.Get(ctx, "key").Err()
})
```

### 带事件回调

```go
type MyHandler struct{}

func (h *MyHandler) OnConnected() {
    log.Println("已连接")
}

func (h *MyHandler) OnDisconnected(err error) {
    log.Printf("断开连接: %v", err)
}

func (h *MyHandler) OnReconnecting(attempt int, delay time.Duration) {
    log.Printf("正在重连 (尝试 %d, 等待 %v)", attempt, delay)
}

func (h *MyHandler) OnError(err error) {
    log.Printf("错误: %v", err)
}

comp := reconnect.New(connector, &MyHandler{}, nil)
```

### 自定义配置

```go
cfg := &reconnect.DefaultReconnectConfig{
    MaxRetries:          10,
    InitialInterval:     500 * time.Millisecond,
    MaxInterval:         10 * time.Second,
    HealthCheckInterval: 5 * time.Second,
}
comp := reconnect.New(connector, nil, cfg)
```

### 动态配置策略

```go
type DynamicConfig struct {
    *reconnect.DefaultReconnectConfig
}

func (c *DynamicConfig) GetMaxRetries() int {
    hour := time.Now().Hour()
    if hour >= 22 || hour < 6 {
        return -1
    }
    return 10
}

func (c *DynamicConfig) GetInitialInterval() time.Duration {
    if isHighLoad() {
        return 5 * time.Second
    }
    return 500 * time.Millisecond
}
```

## 完整示例

参考 `components/tcp/demo/main.go` - 一个基于 TCP 连接的演示：

```bash
go run components/tcp/demo/main.go
```

演示内容：
1. 正常连接和操作
2. 服务端断开连接
3. 客户端自动重连
4. 重连后继续正常工作

## 文件结构

```
pkg/reconnect/
├── connector.go      # Connector/ConfigProvider/ClientGetter 接口
├── config.go         # ReconnectConfig 接口 + 默认实现
├── component.go      # Component + Client 实现
├── backoff.go        # 指数退避算法
├── event.go          # 事件处理器
├── README.md         # 使用文档
└── component_test.go # 单元测试
```

## 默认配置值

| 字段 | 默认值 |
|------|--------|
| MaxRetries | -1 (无限) |
| InitialInterval | 1s |
| MaxInterval | 30s |
| Multiplier | 2.0 |
| CloseTimeout | 10s |
| HealthCheckInterval | 0 (不检查) |

## 线程安全

- Component 内部使用 mutex + cond 实现线程安全
- 序列号机制确保多协程不会重复刷新
- Client.Do() 自动处理并发场景
