# 分布式 ID 组件

`Start` 从 etcd 租约中申请节点号，成功后可调用 `Generate() snowflake.ID`。
按本次确认的接口契约，未启动、租约失效、节点号被删除/替换或组件关闭后，`Generate` 会以 `uniqid: no valid lease` panic，不能继续返回 ID。需要恢复处理的调用方应在明确的业务边界捕获该 panic。

底层 Snowflake 节点改为私有字段，调用方不能再通过嵌入的 `Node` 绕过租约检查。节点号相同的重新申请保留本实例的序列计数。每次成功认领节点号后，组件等待至少一个完整毫秒，再检查取消、关闭状态和租约期限，最后开放发号；等待期间不能生成 ID，关闭可立即取消等待。这会给初次启动和重新申请各增加至少 1ms，不增加每次 Generate 的等待。

毫秒隔离解决节点号快速交接时的序列重置冲突；发号仍依赖各实例系统时间准确这一 Snowflake 前提。本方案没有覆盖跨机器时钟偏差或时钟回拨，也没有新增持久化发号上界。

组件用 Grant 和单次续租请求的开始时刻计算本地有效期；在续租阻塞时本地有效期仍会到期。生成前后均校验期限。续租失败后停止发号，并由原观察循环重新申请租约。`GetServiceId` 返回最后分配的节点号，仅供状态查看，不代表当前仍持有租约；`GetServiceIds` 返回独立快照。

`Start` 每个实例只能调用一次。启动结果使用独立通道返回，不经过事件订阅。`Close` 取消观察、关闭所有订阅、在 `CloseTimeout` 内等待退出，并撤销自有租约；撤销失败或关闭超时会返回错误。通过 `WithEtcd` 注入的客户端由调用方拥有，组件不会关闭它。

## v2 事件订阅

v2 移除 `WithSubscriptions`，改用 `component.Subscribe(capacity)`，返回 `(*Subscription, error)`。capacity 必须为正数，每个订阅有独立的有界队列。订阅只接收建立后的事件；需要初始节点通知时，在 `Start` 前调用，或在 Builder 的配置回调中调用。

```go
sub, err := component.Subscribe(16)
if err != nil {
	return err
}
defer sub.Close()
for event := range sub.Events() {
	if event.Type == uniqid.TypeError {
		// 将租约或观察错误交给应用的错误处理逻辑。
		return event.Error
	}
	// 节点通知发生后，通过组件的 GetServiceId/GetServiceIds 读取当前状态。
}
return sub.Err()
```

以上为应用消费循环片段；接收类型为 `Event` 值，调用方不能关闭或写入 `Events()`。事件是状态变化通知，不包含当时的状态快照；组件读取方法返回消费时的当前状态。

事件按发布顺序入队。队列满时该订阅立即终止，已入队事件仍可排空，`Err()` 返回 `ErrSlowSubscriber`；调用方必须检查并处理此错误，后续需要观察时重新订阅并读取当前状态。该失败不阻塞续租、节点所有权校验或其他订阅者，也不启动额外派发 goroutine。

`Subscription.Close()` 仅取消该订阅，组件 `Close()` 则关闭全部订阅。正常关闭的 `Err()` 为 nil；已经溢出的订阅在重复关闭后仍保留 `ErrSlowSubscriber`。订阅容量只控制通知缓冲，不改变租约期限或错误处理语义。
