# 架构与生命周期

返回 [文档索引](index.md)。接口实现见 [engine.go](../engine/engine.go)，排序实现见 [graph.go](../engine/graph.go)。

## 分层

| 层 | 职责 | 调用关系 |
|---|---|---|
| `engine` | 保存 Builder、依赖图、实例和配置 | 不依赖具体组件 |
| `components` | 把第三方客户端或服务端封装成 Builder | 由应用注册到引擎 |
| `pkg` | 重连、定时任务、按键锁 | 可直接创建使用 |
| 应用骨架 | 配置、启动编排、路由、业务服务、模型、错误响应 | 使用 Leo 并定义自己的组件 |

## 引擎契约

1. **Register**：保存 `func() (any, error)` 和依赖名称。同名重复注册保留第一次注册；它不替换已有 Builder。
2. **Build**：先进行拓扑排序，然后构建组件。依赖环、未注册的依赖、nil Builder 和构建错误会返回错误。应用在一次生命周期中调用一次 `Build`；重复调用会再次构建，不提供热重载或幂等重建。
3. **Get**：返回已构建的实例。名称不存在或尚未构建时会 panic；调用方还需按实际类型断言。
4. **Close**：按依赖顺序的逆序移除实例，调用实现 `Close() error` 的组件。即使某个关闭失败，仍继续关闭其余组件，最后返回遇到的首个错误。
5. **Wait**：等待 Interrupt、HUP、INT、TERM 或 QUIT，解除信号订阅并调用 `Close`。

依赖关系保证先后次序；不要依赖互不关联组件的注册次序来表达业务约束。

## 资源所有权

`Build` 失败不会自动回滚先前成功构建的组件，应用应调用 `Close`。失败的 Builder 尚未把实例交给引擎，必须自行释放已取得的资源。注入的外部客户端则按组件的所有权契约处理，例如 `uniqid.WithEtcd` 不转移 etcd 客户端的所有权。

引擎只识别 `Close() error`。原生 NATS 连接和 `TimeWheel` 的 `Close()` 没有返回值，Resty 组件没有该关闭接口，不能假设注册后会自动回收；调用方应显式管理这些资源。Gin、Iris、gRPC 等服务先停止接受请求，再释放它们依赖的数据库和日志，依靠显式依赖关系保证顺序。

组件的启动方式并不相同：`uniqid` 和 RabbitMQ 的 Builder 会调用 `Start`，重连组件需要显式 `Start`，时间轮在 `New` 时已启动后台循环。重复启动、关闭超时等约束见 [组件指南](components.md)。

## 应用接入方式

模板中的 [Bootstrap](../examples/project-template/internal/bootstrap/bootstrap.go) 集中注册组件，[Container](../examples/project-template/internal/app/app.go) 提供带类型的方法，业务服务声明自己需要的接口。默认启动日志、ID 和 Web；只有显式配置数据库后才注册数据库与示例业务。引擎不替应用决定数据库、认证方式、业务模型或 HTTP 框架。
