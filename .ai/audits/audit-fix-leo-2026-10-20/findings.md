# 问题、反证与修复验证

24 条问题记录（相关表现按组件契约归组）；完整字段见 [findings.json](findings.json)。全部已修复并验证，无待确认高危问题。

|ID|级别|问题|修复与验证|
|---|---|---|---|
|F01|P2|迁移历史遍历忽略末尾读取错误|遍历后返回 rows.Err，查询、扫描、未知版本及正常历史均覆盖；`TestMigrateValidatesCompleteHistory`|
|F02|P1|Iris 与 golog 接口不兼容|golog 调整至 v0.1.12，go mod vendor 正式生成，其他模块版本不变；`go build ./...`|
|F03|P2|文件读取与目录元数据结果错误|直接读取并传播不存在错误，补齐目录标记；`TestFileErrorsAndDirectoryInfo`|
|F04|P2|非覆盖创建存在检查与写入竞争|使用 O_EXCL 原子创建，仅 ForceReplace 使用截断写入；`TestConcurrentCreateDoesNotOverwrite`|
|F05|P1|符号链接绕过存储根目录边界|实际 I/O 使用 Go1.25 已具备的 os.Root；`TestStorageCannotFollowSymlinkOutsideRoot`|
|F06|P2|不同任务的字符串拼接键碰撞|以结构体二元组作为内部 map 键；`TestCompositeJobKeysDoNotCollide`|
|F07|P1|回调新增任务与主循环派发互相等待|待派发队列与请求接收共同参与 select；关闭时保留已到期派发；`TestCallbackCanAddWithUnbufferedQueues`|
|F08|P2|长时间重试退避发生整数溢出|浮点计算后先限幅再转换 duration；`TestBackoffRemainsBoundedAfterLongOutage`|
|F09|P1|连接代次、重连资源和并发生命周期不一致|单一循环拥有连接操作；锁内发布代次和快照；当前代次取消；启动关闭守卫及既有关闭超时；`TestHealthReconnectReplacesExistingClient`|
|F10|P2|构造失败未释放已取得的资源|失败时清理尚未交给引擎的资源，保留主错误与可返回的关闭错误；`TestBuilderClosesClientWhenConfigurerFails`|
|F11|P2|gRPC 关闭忽略已配置的超时|在截止时间触发 Stop 并返回 DeadlineExceeded；`TestCloseHonorsShutdownTimeout`|
|F12|P2|设置 HTTP 超时丢失已有客户端配置|按 SetX 约定修改当前客户端并返回自身；`TestSetTimeoutPreservesClientConfiguration`|
|F13|P2|查询协议错误被转换为空结果成功|在映射数据前验证响应；保留 RawMessage 大整数精度；`TestQueryRejectsFailedOrMalformedResponses`|
|F14|P1|重连复用 DeliveryTag 被误判为重复|缓存归属于当前 channel，旧 Message 只持有旧缓存；`TestDeliveryTagsAreIsolatedBetweenChannels`|
|F15|P2|RabbitMQ 启动失败与关闭状态不准确|失败启动回收；拨号/握手受组件取消；准确区分正常关闭和超时；非阻塞清空；`TestFailedStartStopsSubscription`|
|F16|P1|失去租约继续发号及状态/观察/关闭不一致|按用户选择保留 Generate() ID，失效 panic；节点私有；本地期限门控；单次续租；快照与修订号；完整取消/回收；`TestGenerateDoesNotContinueAfterLeaseRevoke`|
|F17|P2|日志管理器关闭后文件描述符仍打开|记录并关闭管理器自有 writer，保留 stdout 所有权；`TestCloseReleasesOwnedLogFile`|
|F18|P3|Wait 退出未解除信号订阅|增加对应 defer signal.Stop；`go test ./engine`|
|F19|P1|Iris 启动调用把 Runner 当作 Configurator|用一个 Runner 将既有 listener 交给带超时配置的 Server；`go build ./...`|
|F20|P2|InfluxDB 自有 HTTP 连接池未关闭|组件显式保存并关闭自有 http.Client 的 idle connections；`TestBuilderReleasesOwnedHTTPConnections`|
|F21|P2|重复 Close 向已关闭通道 Flush 导致 panic|删除多余 Flush，使用 once 统一资源关闭；`TestComponentCloseIsIdempotent`|
|F22|P1|NATS 异步错误日志泄露连接配置|仅记录错误，不格式化连接或订阅对象；`TestErrorHandlerDoesNotLogConnectionCredentials`|
|F23|P2|Iris 失败构建泄漏端口，启动超时被报告成功|同步 Build 后绑定，等待 Host 注册，超时失败并关闭资源，Close 保证释放 listener；`TestBuildFailureReleasesListener`|
|F24|P3|文档快速链接指向不存在的 README|改为现有 AGENTS.md 并添加本轮 QA 导航；`新建/修改 Markdown 目标存在性检查`|

## F01 迁移历史遍历忽略末尾读取错误

- 指纹：`db/migration-history/row-error`；级别：P2；状态：fixed-verified。
- 源码：[components/db/migrate.go](../../../components/db/migrate.go)
- 触发与原行为：历史记录在首行或中途读取失败仍执行、记录后续迁移。
- 根因修复：遍历后返回 rows.Err，查询、扫描、未知版本及正常历史均覆盖。
- 验证：真实 GORM + database/sql 故障夹具；原实现两个错误场景返回 nil 并执行迁移。测试/命令：`TestMigrateValidatesCompleteHistory`。
- 反证与约束：查询和 Scan 错误原本已处理；正常空历史和已知历史仍应成功。

## F02 Iris 与 golog 接口不兼容

- 指纹：`build/iris-golog/rich-option-types`；级别：P1；状态：fixed-verified。
- 源码：[go.mod](../../../go.mod), [vendor/modules.txt](../../../vendor/modules.txt)
- 触发与原行为：全量构建发生 RichOption 类型不匹配。
- 根因修复：golog 调整至 v0.1.12，go mod vendor 正式生成，其他模块版本不变。
- 验证：基线编译失败，修复后全量构建通过。测试/命令：`go build ./...`。
- 反证与约束：已核对 Iris v12.2.11 依赖声明与 golog 0.1.12 类型；无需升级 Iris 或手改依赖。

## F03 文件读取与目录元数据结果错误

- 指纹：`localfile/result-contract/missing-and-directory`；级别：P2；状态：fixed-verified。
- 源码：[components/storage/localfile/component.go](../../../components/storage/localfile/component.go)
- 触发与原行为：不存在文件返回 nil,nil；目录的 IsDir 未赋值。
- 根因修复：直接读取并传播不存在错误，补齐目录标记。
- 验证：本地文件系统反例失败，修复后通过。测试/命令：`TestFileErrorsAndDirectoryInfo`。
- 反证与约束：零字节文件仍是合法成功；Exists 的不存在语义仍为 false,nil。

## F04 非覆盖创建存在检查与写入竞争

- 指纹：`localfile/create/check-then-write`；级别：P2；状态：fixed-verified。
- 源码：[components/storage/localfile/component.go](../../../components/storage/localfile/component.go)
- 触发与原行为：并发创建同一路径时多个调用成功并覆盖。
- 根因修复：使用 O_EXCL 原子创建，仅 ForceReplace 使用截断写入。
- 验证：原始源码副本出现 8 个成功创建者；当前实现只有 1 个。测试/命令：`TestConcurrentCreateDoesNotOverwrite`。
- 反证与约束：保留自动建目录和显式覆盖行为。

## F05 符号链接绕过存储根目录边界

- 指纹：`localfile/io/symlink-root-escape`；级别：P1；状态：fixed-verified。
- 源码：[components/storage/localfile/component.go](../../../components/storage/localfile/component.go)
- 触发与原行为：根内链接可读取、写入、列出或删除根外目标。
- 根因修复：实际 I/O 使用 Go1.25 已具备的 os.Root。
- 验证：原实现六类越界操作成功，修复后拒绝；macOS 与 Linux 均通过。测试/命令：`TestStorageCannotFollowSymlinkOutsideRoot`, `TestCreateMissingRootAndInternalSymlink`, `TestRemoveConfiguredRoot`。
- 反证与约束：词法 ../ 校验已有效但不能覆盖链接；根内相对链接和删除配置根路径仍支持。

## F06 不同任务的字符串拼接键碰撞

- 指纹：`timewheel/job-index/composite-key-collision`；级别：P2；状态：fixed-verified。
- 源码：[pkg/timewheel/component.go](../../../pkg/timewheel/component.go)
- 触发与原行为：(a:b,c) 与 (a,b:c) 覆盖彼此。
- 根因修复：以结构体二元组作为内部 map 键。
- 验证：原实现丢失一项任务；修复后两项均派发。测试/命令：`TestCompositeJobKeysDoNotCollide`。
- 反证与约束：没有改变公开 Key/Id 字符串取值范围。

## F07 回调新增任务与主循环派发互相等待

- 指纹：`timewheel/dispatch/reentrant-request-deadlock`；级别：P1；状态：fixed-verified。
- 源码：[pkg/timewheel/component.go](../../../pkg/timewheel/component.go)
- 触发与原行为：零缓冲队列下回调 Add 无法返回。
- 根因修复：待派发队列与请求接收共同参与 select；关闭时保留已到期派发。
- 验证：原实现超时；修复后执行及关闭回归通过。测试/命令：`TestCallbackCanAddWithUnbufferedQueues`, `TestCloseWaitsMainloopAndDispatch`。
- 反证与约束：只加大缓冲不能消除循环等待；未改变 Close 等待回调结束的契约。

## F08 长时间重试退避发生整数溢出

- 指纹：`reconnect/backoff/duration-overflow`；级别：P2；状态：fixed-verified。
- 源码：[pkg/reconnect/backoff.go](../../../pkg/reconnect/backoff.go)
- 触发与原行为：Linux amd64 第 34 次重试返回负 duration。
- 根因修复：浮点计算后先限幅再转换 duration。
- 验证：Linux amd64 原始源码反例失败，当前源码 10000 次循环通过。测试/命令：`TestBackoffRemainsBoundedAfterLongOutage`。
- 反证与约束：Darwin arm64 的转换表现不同，不能只靠本机结果排除跨平台缺陷。

## F09 连接代次、重连资源和并发生命周期不一致

- 指纹：`reconnect/connection-generation/lifecycle-synchronization`；级别：P1；状态：fixed-verified。
- 源码：[pkg/reconnect/component.go](../../../pkg/reconnect/component.go), [pkg/reconnect/connector.go](../../../pkg/reconnect/connector.go)
- 触发与原行为：旧连接未断开；健康重连后 wrapper 仍引用旧对象；Raw/Do/seq 竞态；重复启动；健康检查不能随关闭取消。
- 根因修复：单一循环拥有连接操作；锁内发布代次和快照；当前代次取消；启动关闭守卫及既有关闭超时。
- 验证：旧连接计数、过期客户端及 race 检测均复现；当前全量 race 通过。测试/命令：`TestHealthReconnectReplacesExistingClient`, `TestConcurrentClientRefreshAndRaw`, `TestStartIsIdempotentAndCannotRestartAfterClose`, `TestCloseCancelsInflightHealthCheck`, `TestCloseTimeoutReportsBlockedConnector`, `TestConcurrentStartAndClose`。
- 反证与约束：README 明确支持并发 Client.Do；回调在锁外执行；测试夹具竞态独立修复。

## F10 构造失败未释放已取得的资源

- 指纹：`shared/builder/unpublished-resource-cleanup`；级别：P2；状态：fixed-verified。
- 源码：[components/db/component.go](../../../components/db/component.go), [components/db/builder.go](../../../components/db/builder.go), [components/etcd/builder.go](../../../components/etcd/builder.go), [components/grpc/client/builder.go](../../../components/grpc/client/builder.go), [components/grpc/server/builder.go](../../../components/grpc/server/builder.go), [components/nats/builder.go](../../../components/nats/builder.go), [components/influxdb/builder.go](../../../components/influxdb/builder.go), [components/zaplog/log/builder.go](../../../components/zaplog/log/builder.go)
- 触发与原行为：后续配置或从库初始化失败，先前连接/文件对象无人关闭。
- 根因修复：失败时清理尚未交给引擎的资源，保留主错误与可返回的关闭错误。
- 验证：SDK 状态、SQL pool、本地 NATS TCP 及真实 MySQL 均验证；MySQL 原始源码从 1 个连接增至 2 个，当前恢复至 1 个。测试/命令：`TestBuilderClosesClientWhenConfigurerFails`, `TestBuilderClosesConnectionWhenConfigurerFails`, `TestPartialInitializationClosesOpenedPools`。
- 反证与约束：引擎只能关闭成功返回并登记的实例，无法替 Builder 回收未返回资源。

## F11 gRPC 关闭忽略已配置的超时

- 指纹：`grpcserver/close/unused-shutdown-timeout`；级别：P2；状态：fixed-verified。
- 源码：[components/grpc/server/component.go](../../../components/grpc/server/component.go)
- 触发与原行为：活动 RPC 使 GracefulStop 超过 ShutdownTimeout。
- 根因修复：在截止时间触发 Stop 并返回 DeadlineExceeded。
- 验证：实际 TCP RPC 复现超时，修复后活动 RPC 被取消。测试/命令：`TestCloseHonorsShutdownTimeout`, `TestCloseWithoutActiveRPC`。
- 反证与约束：未配置正超时时保留原有等待行为；不把超时当作正常成功。

## F12 设置 HTTP 超时丢失已有客户端配置

- 指纹：`restyclient/set-timeout/config-loss`；级别：P2；状态：fixed-verified。
- 源码：[components/restyclient/component.go](../../../components/restyclient/component.go)
- 触发与原行为：新建空 Resty 导致 base URL、请求头和 hooks 丢失。
- 根因修复：按 SetX 约定修改当前客户端并返回自身。
- 验证：原实现相对 URL 请求失败；修复后实际 HTTP 请求及 hook 通过。测试/命令：`TestSetTimeoutPreservesClientConfiguration`。
- 反证与约束：Resty.Clone 是浅拷贝，会共享 http.Client；不采用它假装提供独立超时。

## F13 查询协议错误被转换为空结果成功

- 指纹：`influxdb/query/protocol-error-as-success`；级别：P2；状态：fixed-verified。
- 源码：[components/influxdb/query.go](../../../components/influxdb/query.go)
- 触发与原行为：无效 JSON、HTTP200 的 error、非法结构或行列数不一致被忽略。
- 根因修复：在映射数据前验证响应；保留 RawMessage 大整数精度。
- 验证：本地 HTTP 响应覆盖错误和正常空结果；9007199254740993 精度保持。测试/命令：`TestQueryRejectsFailedOrMalformedResponses`。
- 反证与约束：HTTP200 不保证查询成功，已核对官方 v1 API；QueryRaw 保留原始响应接口。

## F14 重连复用 DeliveryTag 被误判为重复

- 指纹：`rabbitmq/delivery/channel-scoped-ack-state`；级别：P1；状态：fixed-verified。
- 源码：[components/rabbitmq/subscription/component.go](../../../components/rabbitmq/subscription/component.go)
- 触发与原行为：不同 channel 从相同 delivery tag 起步时被旧确认缓存丢弃。
- 根因修复：缓存归属于当前 channel，旧 Message 只持有旧缓存。
- 验证：真实 Message.Ack 接口夹具验证；当前实现完整路径传入每个 channel 新建的缓存。测试/命令：`TestDeliveryTagsAreIsolatedBetweenChannels`, `TestAckFailureIsNotCached`。
- 反证与约束：DeliveryTag 只在 channel 内唯一；AutoAck 保留原有无需手动确认语义。

## F15 RabbitMQ 启动失败与关闭状态不准确

- 指纹：`rabbitmq/lifecycle/cancellation-and-start-failure`；级别：P2；状态：fixed-verified。
- 源码：[components/rabbitmq/subscription/component.go](../../../components/rabbitmq/subscription/component.go)
- 触发与原行为：启动首错后继续后台重连；关闭不能打断握手；常规 Close 返回 Canceled；ClearMsgs 竞争可能阻塞。
- 根因修复：失败启动回收；拨号/握手受组件取消；准确区分正常关闭和超时；非阻塞清空。
- 验证：本地不响应 AMQP 握手的 TCP 服务复现，修复后关闭及时完成。测试/命令：`TestFailedStartStopsSubscription`, `TestCloseCancelsAMQPHandshake`, `TestStartCloseRace`。
- 反证与约束：保留 URI connection_timeout；自定义阻塞回调仍可能超时，此时准确报错。

## F16 失去租约继续发号及状态/观察/关闭不一致

- 指纹：`uniqid/lease-ownership/generation-and-lifecycle`；级别：P1；状态：fixed-verified。
- 源码：[components/uniqid/component.go](../../../components/uniqid/component.go)
- 触发与原行为：租约释放后仍生成 ID；内部集合被调用方修改；状态竞态；订阅派发未退出；Watch 关闭空转及快照缺口。
- 根因修复：按用户选择保留 Generate() ID，失效 panic；节点私有；本地期限门控；单次续租；快照与修订号；完整取消/回收。
- 验证：故障接口夹具 red/green + race；官方 etcd3.6.7 双实例跨原 TTL 发号及释放通过。测试/命令：`TestGenerateDoesNotContinueAfterLeaseRevoke`, `TestGenerateFailsWhenRenewalFails`, `TestGenerateFailsAtExpiryWithoutRenewalResponse`, `TestRenewalExtendsValidity`, `TestOwnershipLossStopsGeneration`, `TestHistoricalDeleteBeforeClaimDoesNotInvalidateLease`, `TestCloseReportsRevokeFailure`, `TestEtcdLeaseAllocationRenewalAndRelease`。
- 反证与约束：Get 错误原本已经处理，不作为新缺陷；历史删除早于当前申请修订号时不得误伤新租约。

## F17 日志管理器关闭后文件描述符仍打开

- 指纹：`zaplog/close/owned-file-writer`；级别：P2；状态：fixed-verified。
- 源码：[components/zaplog/log/component.go](../../../components/zaplog/log/component.go)
- 触发与原行为：AddSync 给 lumberjack 的 Sync 不会关闭文件。
- 根因修复：记录并关闭管理器自有 writer，保留 stdout 所有权。
- 验证：Unix /dev/fd 与 Fstat 确认原实现留下 1 个描述符；macOS/Linux 当前实现为 0。测试/命令：`TestCloseReleasesOwnedLogFile`。
- 反证与约束：不关闭共享 stdout；独立 NewLog 的返回类型保持不变。

## F18 Wait 退出未解除信号订阅

- 指纹：`engine/wait/signal-subscription-lifetime`；级别：P3；状态：fixed-verified。
- 源码：[engine/engine.go](../../../engine/engine.go)
- 触发与原行为：signal.Notify 注册在 Wait 结束后仍存在。
- 根因修复：增加对应 defer signal.Stop。
- 验证：静态注册/释放配对及引擎回归通过。测试/命令：`go test ./engine`, `go vet ./...`。
- 反证与约束：未改动已有信号集合或关闭顺序；未把静态配对验证冒充信号端到端测试。

## F19 Iris 启动调用把 Runner 当作 Configurator

- 指纹：`iris/run/runner-configurator-type`；级别：P1；状态：fixed-verified。
- 源码：[components/iris/web/builder.go](../../../components/iris/web/builder.go)
- 触发与原行为：修复依赖类型后项目仍无法构建。
- 根因修复：用一个 Runner 将既有 listener 交给带超时配置的 Server。
- 验证：编译错误明确，修复后实际 HTTP 请求及超时配置验证通过。测试/命令：`go build ./...`, `TestServerUsesConfiguredTimeoutsAndCloses`。
- 反证与约束：没有删除 Read/Write/IdleTimeout 来迁就签名。

## F20 InfluxDB 自有 HTTP 连接池未关闭

- 指纹：`influxdb/transport/owned-http-pool`；级别：P2；状态：fixed-verified。
- 源码：[components/influxdb/component.go](../../../components/influxdb/component.go), [components/influxdb/builder.go](../../../components/influxdb/builder.go)
- 触发与原行为：成功关闭及两种初始化失败路径均遗留 idle connection。
- 根因修复：组件显式保存并关闭自有 http.Client 的 idle connections。
- 验证：本地服务 ConnState 观测三种路径；原实现都不关闭，当前均关闭。测试/命令：`TestBuilderReleasesOwnedHTTPConnections`。
- 反证与约束：SDK OwnHTTPClient=false 时不关闭外部传入的客户端，本例实际所有者是 Leo。

## F21 重复 Close 向已关闭通道 Flush 导致 panic

- 指纹：`influxdb/close/duplicate-flush`；级别：P2；状态：fixed-verified。
- 源码：[components/influxdb/component.go](../../../components/influxdb/component.go)
- 触发与原行为：第二次关闭触发 send on closed channel。
- 根因修复：删除多余 Flush，使用 once 统一资源关闭。
- 验证：原实现 panic 反例明确，修复后重复关闭通过。测试/命令：`TestComponentCloseIsIdempotent`。
- 反证与约束：Client.Close 本身已负责 Flush，不降低原有刷出要求。

## F22 NATS 异步错误日志泄露连接配置

- 指纹：`nats/error-log/connection-credentials`；级别：P1；状态：fixed-verified。
- 源码：[components/nats/builder.go](../../../components/nats/builder.go)
- 触发与原行为：打印完整 Conn，合成 Password 值进入日志。
- 根因修复：仅记录错误，不格式化连接或订阅对象。
- 验证：本地协议服务与合成值验证，原测试失败，当前通过；测试未输出凭据内容。测试/命令：`TestErrorHandlerDoesNotLogConnectionCredentials`。
- 反证与约束：保留错误详情，未简单禁用错误日志。

## F23 Iris 失败构建泄漏端口，启动超时被报告成功

- 指纹：`iris/startup/publication-and-owned-listener`；级别：P2；状态：fixed-verified。
- 源码：[components/iris/web/builder.go](../../../components/iris/web/builder.go), [components/iris/web/component.go](../../../components/iris/web/component.go)
- 触发与原行为：OnBuild 错误后端口不能重绑；Host 配置阻塞超过窗口仍返回成功。
- 根因修复：同步 Build 后绑定，等待 Host 注册，超时失败并关闭资源，Close 保证释放 listener。
- 验证：原行为均由 TCP 端口与 HTTP 测试复现；当前 race/Linux 回归通过。测试/命令：`TestBuildFailureReleasesListener`, `TestHostStartTimeoutIsFailure`, `TestServerUsesConfiguredTimeoutsAndCloses`, `TestImmediateCloseReleasesListener`。
- 反证与约束：历史 QA 已记录窗口只能降低误判概率；现在处理根因，保留配置字段和服务超时。

## F24 文档快速链接指向不存在的 README

- 指纹：`docs/index/missing-readme-target`；级别：P3；状态：fixed-verified。
- 源码：[docs/index.md](../../../docs/index.md)
- 触发与原行为：构建入口链接无法访问。
- 根因修复：改为现有 AGENTS.md 并添加本轮 QA 导航。
- 验证：静态文件存在性确认。测试/命令：`新建/修改 Markdown 目标存在性检查`。
- 反证与约束：历史 QA 报告作为历史事实保留，不改写其当时环境与结论。
