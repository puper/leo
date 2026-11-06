# 开发方式与组件应用

返回 [文档索引](index.md)。本骨架以完整的配置 → 注册 → 业务 → 路由 → 测试链路演示 Leo 的应用方式。

## 已运行在骨架中的组件

| 能力 | 实现位置 | 说明 |
|---|---|---|
| 生命周期 | [Bootstrap](../internal/bootstrap/bootstrap.go) | 校验配置后 Register/Build/Wait；构建失败清理已创建资源 |
| 命名日志 | [logging](../internal/logging/builder.go)、[Container](../internal/app/app.go) | Leo Zap 组件提供 default/access 日志 |
| ID | [idgen Builder](../internal/components/idgen/builder.go) | 固定节点号 Snowflake，要求进程之间节点号不同 |
| 可选数据库 | [Bootstrap](../internal/bootstrap/bootstrap.go)、[迁移](../migrates/migrates.go) | 使用 Leo db.Builder 与 WithMigrateFs，失败阻止启动 |
| 应用自定义组件 | [Gin Web](../internal/components/gin/web/component.go) | 实现 `Close() error`，通过 engine.Builder 接入 |
| 业务模块 | [example](../internal/modules/example/builder.go) | 声明 Db/IdGen 依赖，注册创建/查询服务与路由 |

`internal/app.Container` 把引擎的名称与类型断言集中在一处。业务模块通过局部接口表达实际需要的能力；不要让每个 Service 到处查找字符串组件名。配置校验集中在 `internal/config`，新增字段同时更新示例配置。

## 增加一个业务模块

1. 参考 `internal/modules/example` 建立自己的模块目录。`model/dto` 放请求和响应，`model/db` 仅放实际需要持久化的模型，`service` 处理业务，`route` 绑定接口。
2. Service 的方法遵循当前 ServiceCaller 签名：`Method(context.Context, json.RawMessage, *Reply) error`。参数错误返回 `derrors.NewParamError`，不存在返回 `NewNotFoundError`，底层错误向上传播；JSON ID 用字符串。
3. 定义模块 `Container` 接口，只声明实际调用的方法。在 `internal/app` 实现类型化访问，必要时在 `internal/ifaces` 复用已有接口。
4. 在 `internal/components/names.go` 定义名称，在 Bootstrap 注册 Builder 并列出实际依赖。HTTP 路由依赖该模块时，同时加入 Web 的依赖列表。
5. 通过模块路由应用鉴权。当前示例只有认证，没有租户或记录归属规则；真实业务在 Service/数据库查询中实现自己的授权约束。
6. 添加正常、参数错误、未授权、未找到与外部失败测试。持久化变化添加新的 SQL 迁移，更新 Swagger 注释后执行 `make swagger`。

开发完成运行 `make build test lint`。涉及并发或生命周期时再运行 `go test -race -mod=readonly ./...`；数据库集成按 README 使用隔离实例。

## 接入 etcd 分布式 ID

当前骨架没有默认要求 etcd。需要自动分配节点号时，可以将固定 Snowflake Builder 替换为 Leo v2 的 `uniqid`。模板依赖 v2.0.0，包含租约校验、节点交接与慢订阅者隔离。以下是完整的注册函数示例，调用前由应用加载并校验两份配置：

```go
package componentexample

import (
	"github.com/puper/leo/v2/components/etcd"
	etcdconfig "github.com/puper/leo/v2/components/etcd/config"
	"github.com/puper/leo/v2/components/uniqid"
	idconfig "github.com/puper/leo/v2/components/uniqid/config"
	"github.com/puper/leo/v2/engine"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func RegisterID(e *engine.Engine, etcdCfg *etcdconfig.Config, idCfg *idconfig.Config) {
	e.Register("etcd", etcd.Builder(etcdCfg))
	e.Register("idgen", uniqid.Builder(idCfg, uniqid.WithEtcd(func() *clientv3.Client {
		return e.Get("etcd").(*clientv3.Client)
	})), "etcd")
}
```

接入时替换原有 IdGen 注册，避免同名重复注册被忽略；将 `Container.GenerateID` 的类型断言同步改成 `*uniqid.Component`。设置明确的 endpoints、KeyPrefix、节点范围、LeaseTimeout、InitTimeout、CloseTimeout。只有 Builder 成功后才能发号；失去有效租约后 `Generate()` 会 panic，应用应按该契约终止相关业务，不能绕过校验继续发号。系统时钟须准确，组件不解决跨机器时钟偏差。

依赖声明使关闭顺序为业务 → uniqid → etcd。`WithEtcd` 注入的客户端仍由 etcd 组件拥有，不由 uniqid 重复关闭。

需要观察事件时在 Builder 配置回调中调用 `Subscribe(capacity)`，应用从返回的订阅读取 `Events()`，循环结束后检查 `Err()`。容量满时以 `uniqid.ErrSlowSubscriber` 终止该订阅，不影响发号；v2 不再接受 `WithSubscriptions` 外部 channel。

## 选择其他 Leo 组件

| 场景 | 组件 | 应用应处理的边界 |
|---|---|---|
| 调用外部 HTTP | `components/restyclient` | 显式超时、检查请求错误和 HTTP 状态，结束使用后关闭空闲连接 |
| 消费 RabbitMQ | `components/rabbitmq/subscription` | Builder 已 Start；业务成功后手动 Ack，确认失败需处理；业务去重由应用实现 |
| NATS 消息 | `components/nats` | 返回原生 `*nats.Conn`，其 `Close()` 不符合引擎的 `Close() error`，应显式关闭 |
| 文件持久化 | `components/storage/localfile` | 配置可信 RootDir、使用逻辑相对名，明确覆盖和删除策略 |
| 定时回调 | `pkg/timewheel` | `Job.Time` 为 Unix 秒，显式 Close，回调应及时返回 |
| 自定义客户端重连 | `pkg/reconnect` | Connector 遵守 context 取消；Do 回调可重复执行，写操作须考虑幂等性 |

按实际需求引入，不把所有组件默认启动。每增加一种外部依赖，同时补配置校验、依赖声明、失败清理、就绪检查和相应测试。
