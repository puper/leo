# 组件指南

返回 [文档索引](index.md)。配置名称以下列源码为准，应用负责读取与校验；不要将示例中的空地址、空凭据当作可用配置。

## 组件入口

| 组件 | Builder / 配置 | 主要能力 |
|---|---|---|
| 数据库 | [db](../components/db/builder.go) / [config](../components/db/config/config.go) | GORM、MySQL、多连接名、主从读写、迁移 |
| etcd | [etcd](../components/etcd/builder.go) / [config](../components/etcd/config/config.go) | etcd v3 客户端 |
| gRPC 客户端 | [client](../components/grpc/client/builder.go) / [config](../components/grpc/client/config/config.go) | `ClientConn`，地址 `Addr` |
| gRPC 服务端 | [server](../components/grpc/server/builder.go) / [config](../components/grpc/server/config/config.go) | 服务注册、监听、优雅关闭 |
| InfluxDB | [influxdb](../components/influxdb/builder.go) / [config](../components/influxdb/config/config.go) | v2 写入、v1 查询及记录映射 |
| Iris | [web](../components/iris/web/builder.go) / [config](../components/iris/web/config/config.go) | HTTP 服务、路由配置、启动检查 |
| NATS | [nats](../components/nats/builder.go) / [config](../components/nats/config/config.go) | 原生 `*nats.Conn`，自动重连 |
| RabbitMQ | [subscription](../components/rabbitmq/subscription/builder.go) / [config](../components/rabbitmq/subscription/config/config.go) | 消费订阅、声明/绑定、重连、手动确认 |
| HTTP 客户端 | [restyclient](../components/restyclient/builder.go) / [Config](../components/restyclient/component.go) | Resty、连接池、超时和限流 |
| 本地文件 | [localfile](../components/storage/localfile/builder.go) / [Config](../components/storage/localfile/config.go) | 根目录内文件操作，实现 `storage.Storage` |
| 分布式 ID | [uniqid](../components/uniqid/builder.go) / [config](../components/uniqid/config/config.go) | etcd 租约分配 Snowflake 节点 |
| 日志 | [log](../components/zaplog/log/builder.go) / [config](../components/zaplog/log/config/config.go) | 命名 Zap 日志、文件轮转和关闭 |

## 数据库与迁移

`Servers` 按连接名组织，`Write(name)` 使用主库，`Read(name)` 在配置的从库中选择一个；没有从库时读取主库。`ReadModel`、`WriteModel` 根据模型的 `ConnectionName()` 选择连接。实现固定使用 MySQL 驱动，修改 `Driver` 字段不会自动支持其他数据库。

`WithMigrateFs(embed.FS)` 从 `sqls/<连接名>/` 加载 `<数字序号>_<说明>.up.sql` 与可选 `.down.sql`。匹配连接名的迁移在 Builder 内执行，`Migrate` 配置需提供非 nil 的结构。迁移记录表及事务策略由 `MigrateConfig` 指定；新增版本不要覆盖已执行的 SQL。MySQL DDL 的事务能力由数据库决定，不能把 `UseTransaction` 视为所有 DDL 均可回滚的保证。

可执行的配置、模型和迁移示例见 [应用骨架](../examples/project-template/README.md)。迁移失败应阻止应用启动；此库不负责多实例迁移的分布式互斥。

## 网络组件

- gRPC 客户端当前使用明文 transport credentials；服务端 Builder 未配置 TLS。通过配置回调注册服务；不要将它描述为已内置 TLS 的生产入口。服务端正的 `ShutdownTimeout` 会限制优雅关闭等待，超时返回 `context.DeadlineExceeded`。
- Iris 通过 `GetApp()` 配置路由。`StartCheckTimeout` 默认 300ms，用于等待 Host 注册；启动超时返回错误。HTTP 读、写、空闲超时直接使用配置值。
- NATS 当前设为无限重连、间隔 3s。Builder 返回原生连接，调用方负责调用其 `Close()`。
- InfluxDB Builder 会查询指定 Bucket，失败则返回错误。配置中的连接超时字段拼写为 `DailTimeout`（JSON `dailTimeout`）。`Query`/`QueryRecords` 检查 HTTP 与语句错误；记录映射保留 JSON 原始数值，可映射到 `int64`。关闭时释放客户端和自有 HTTP 空闲连接。
- Resty `SetTimeout` 原地修改并保留已有配置。默认限流器为每秒 1 个令牌、突发容量 10。TLS 验证默认开启；由应用持有并在结束使用时调用底层 HTTP 客户端的 `CloseIdleConnections()`。

RabbitMQ 通过 `MsgCh()` 读取消息。`AutoAck=false` 时，业务成功后调用 `Message.Ack` 并检查错误；缓冲区满时产生背压，不会为腾出队列确认未处理消息。投递标签缓存仅在对应通道内有效，不代表业务级去重。Builder 已启动订阅；一个实例不应再次 `Start`。启动/关闭默认超时均为 10s，默认重连间隔从 1s 递增，最大 1min。应用自行结束消费循环，不应依赖 `MsgCh()` 在关闭时被关闭。

## 存储、日志与 ID

- 本地文件的 `RootDir` 是可信配置。普通文件操作使用 `os.Root` 限制目录与符号链接访问；默认创建不覆盖已有文件，需要时显式传 `WithForceReplace`、`WithAutoCreateDir`。`FileInfo.Path` 是本地路径，不是传入所有方法的逻辑相对名；删除根目录是现有显式能力，调用方应控制入口。
- 日志 `Logs` 按名称配置，`Get()` 使用 `default`，未知名称也返回默认日志。`Output` 为空时写标准输出；`Log.Close` 关闭它拥有的文件 writer，不关闭进程共享 stdout。
- `uniqid` 需要通过 `WithEtcd` 注入客户端，并声明对 etcd 的引擎依赖。未启动、租约失效或关闭后的 `Generate()` 会 panic，不能继续发号。系统时钟准确是前提；节点交接、有效期与关闭详情见 [分布式 ID 文档](../components/uniqid/README.md)。模板默认使用显式节点号的 Snowflake，未默认启用 etcd。

## 独立工具

| 包 | 用法与边界 |
|---|---|
| [mutexmanager](../pkg/mutexmanager/component.go) | 同一字符串键共享读写锁；加锁/解锁须使用同一键，Try 成功后才解锁 |
| [reconnect](../pkg/reconnect/README.md) | 实现 Connector，再 New/Start；`Client.Do` 可能重连并重试回调，回调应允许重复执行；关闭后不能重新启动 |
| [timewheel](../pkg/timewheel/component.go) | `New` 启动循环，`Job.Time` 为 Unix 秒；以 `(Key, Id)` 标识任务，按 Key 订阅；Close 等待已派发回调完成，回调内不要同步 Close |

重连包不自带 Redis 等 SDK 适配器。可运行的自定义 Connector 例子见 [TCP demo](../components/tcp/demo/main.go)，执行 `go run ./components/tcp/demo`，它使用本机 9998 端口。
