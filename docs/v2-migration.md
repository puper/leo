# v2.0.0 迁移与发布

返回 [文档索引](index.md)。

## 模块路径

v2 的模块路径是 `github.com/puper/leo/v2`，使用：

```bash
go get github.com/puper/leo/v2@v2.0.0
```

将应用的 `github.com/puper/leo/...` 导入统一改为 `github.com/puper/leo/v2/...`，再执行应用的依赖整理、测试与构建。v1 模块和既有发布标签保持原样，不会自动升级到 v2，也不提供旧路径别名或双版本适配层。

库内 protobuf 的 Go 包路径同步迁移；protobuf package、字段编号和服务方法未变。使用这些生成类型的 Go 调用方需更新 import。

## uniqid 订阅变更

移除 `WithSubscriptions(channels...)`。改为在 `Start` 前或 Builder 的配置回调里调用 `component.Subscribe(capacity)`，从返回的 `Subscription.Events()` 读取 `Event` 值。channel 由组件创建和关闭，调用方不能写入或关闭它。

- capacity 必须为正数，表示该订阅可缓冲的事件数量。
- 每个订阅独立，事件只在订阅建立后交付；启动通常发布节点号与节点列表两条通知。
- 队列满时关闭该订阅，排空已入队事件后 `Err()` 返回 `ErrSlowSubscriber`。不能把这个结果当作正常退出；需要继续观察时重新订阅并读取组件当前状态。
- `Subscription.Close()` 只取消该订阅；组件 `Close()` 关闭所有订阅。重复关闭保留已有错误。
- 租约维护与启动结果不依赖消费者速度；失去有效租约后 `Generate()` 仍 panic，不能绕过保护继续发号。

完整说明见 [uniqid 文档](../components/uniqid/README.md)。默认应用骨架继续使用显式节点号的 Snowflake，没有新增 etcd 运行依赖。

## 服务关闭修复

- gRPC：组件持有 listener，立即关闭时同步释放端口，不依赖 `Serve` goroutine 的启动时机。
- Iris：优雅关闭失败后强制关闭所属 HTTP Server 的普通活动连接，保留超时和关闭错误。
- 强制关闭会取消连接关联的请求 context；业务代码必须响应取消。它不保证强行终止忽略 context 的 handler，也不接管已 hijack 的连接。
- engine 的 Register、Build、Get、Close、Wait 接口及依赖顺序不变。

## 模板与发布验证

`examples/project-template` 的核心用法延续来源模板，Leo 依赖更新为 `/v2 v2.0.0`。根 `go.work` 只用于联合开发；复制的模板和 Docker 镜像使用发布模块及校验和，不包含本机 replace。

发布前分别验证根模块和模板的测试、race、vet、build，以及隔离服务下的 etcd/MySQL 集成。根模块还应执行 `GOWORK=off go test -mod=readonly ./...`。发布时只创建新提交和 `v2.0.0` 标签，不移动既有标签。

发布后从干净缓存下载 `github.com/puper/leo/v2@v2.0.0` 并确认与发布源码一致，再在模板目录验证：

```bash
GOWORK=off make build test lint
make docker-build
```

镜像运行需要应用自己的本地配置；按模板 README 检查 `/readiness` 和退出信号。完整测试命令与外部服务条件见 [开发文档](development.md)。
