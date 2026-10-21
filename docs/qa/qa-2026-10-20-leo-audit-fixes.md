# Leo 审查修复与验证（2026-10-20）

## 结论与范围

基线 `d602fea`，修复保留在工作区，未提交或发布。18 个覆盖单元均已完成，包含 49 个非生成 Go 生产/配置/示例文件、30 个测试文件以及 2 个生成协议文件的接口/构建检查。共 24 条问题记录：P1 8 条、P2 14 条、P3 2 条，全部修复验证；无 P0、待修复或待用户决定项。相关表现按组件契约归组，条数不是“全项目无缺陷”的证明。

R3/R4 连续两次全范围复核无新增高置信 P0/P1，满足本轮范围的收敛条件。第三方源码仅追踪必要契约，不声称完成全部 vendor/CVE 审计。本仓库没有前端 UI，浏览器、设备方向等平台项不适用。已有 `.opencode` 的 67 项删除完整保留。

## 主要改动

|领域|结果|
|---|---|
|迁移与数据库|历史读取中断立即返回错误，不再执行后续迁移；构造和 Builder 失败释放已打开的连接池|
|构建与服务|修复 Iris/golog 类型冲突与 Runner 调用；构建失败不占用端口，启动超时返回失败；gRPC 关闭期限生效|
|重连与任务队列|连接代次与原始客户端一致发布，消除并发竞态与旧连接残留；退避限幅先于整数转换；时间轮键不碰撞、回调可新增任务|
|消息与发号|RabbitMQ 去重状态按 channel 隔离，队列满时保留背压；失败启动可回收，握手可取消；uniqid 租约失效停止发号并完整结束观察和派发|
|文件与 HTTP|文件操作受根句柄约束、非覆盖创建原子化、缺失文件正确报错；HTTP 超时保留配置；InfluxDB 协议错误不再伪成功，自有连接池可关闭|
|资源与日志|日志管理器释放文件描述符，NATS 错误日志不再输出连接配置，Engine 退出解除信号注册|

## 兼容性说明

- 按用户选择，`uniqid.Generate() snowflake.ID` 签名保持，未启动、关闭或无有效租约时 panic：`uniqid: no valid lease`。底层 `Node` 变为私有；直接访问它的外部调用方须改用 Generate。GetServiceIds 返回快照。
- `restyclient.SetTimeout` 现在修改当前客户端，保留鉴权、base URL、headers、hooks 等配置；需要独立实例的调用方应独立创建客户端。
- `reconnect.Start` 运行期间重复调用幂等，关闭后不能重启同一实例；Do/WaitRefresh 等待连接，关闭超时返回错误。
- 本地存储拒绝逃出根目录的链接；`os.Root` 也不接受绝对符号链接。根内相对链接、自动建目录、显式覆盖及删除配置根路径均有回归。
- Iris 的启动成功以 Host 注册为依据，StartCheckTimeout 到期返回错误。自定义配置回调本身应能结束；Go 无法强制终止任意忽略取消的用户回调。
- golog 从 v0.1.15 调整到兼容的 v0.1.12；其余模块版本不变，vendor 正式重新生成，非相关换行改动已撤回。

## 已运行且通过

全部使用 Go1.27.2，`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOFLAGS=-mod=vendor`；本机缓存位于任务项目目录，容器缓存位于容器临时目录。

|验证|实际命令/方式|结果|
|---|---|---|
|最终构建|`go build ./...`|Darwin arm64 通过|
|最终静态检查|`go vet ./...`|Darwin arm64 通过|
|全量竞争检查|`go test -race -count=1 -timeout=90s ./...`|Darwin arm64 通过；同时注入本轮临时 MySQL/etcd 端点，两项真实集成没有跳过|
|第二平台全量测试|`go test -p=2 -count=1 -timeout=90s ./...`|Linux amd64 容器、非 root、无外部网络，通过；该轮的真实服务测试按环境缺失跳过，已由上一轮执行|
|MySQL 部分初始化|`go test -race -count=1 -timeout=30s ./components/db`，通过 `LEO_TEST_MYSQL_DSN` 注入隔离实例|MySQL5.7.44 通过；原始源码副本反例失败|
|etcd 多实例与续租|`go test -race -count=1 -timeout=45s ./components/uniqid -run TestEtcdLeaseAllocationRenewalAndRelease`，通过 `LEO_TEST_ETCD_ENDPOINT` 注入隔离实例|etcd3.6.7 通过：不同节点号、跨初始 TTL 发号、释放与其他实例独立性|
|退避跨架构|`TestBackoffRemainsBoundedAfterLongOutage`|原始 amd64 第34次变负；当前10000次均有界|
|边界与故障|迁移 SQL 行错误、链接越界、并发创建、AMQP 握手、Ack/缓存、HTTP协议、活动RPC、描述符等定向测试|先复现再验证修复；具体见问题账本|
|格式与差异|gofmt、`git diff --check`、Git 原状态比较|通过；保留原有67项删除，无非相关 vendor 变化|

## 已运行但失败

基线全量构建因两个独立类型/调用错误失败。原始源码及修改前故障回归分别暴露迁移伪成功、并发覆盖、路径越界、任务丢失/死锁、旧连接/竞态、失败构造泄漏、关闭超时/重复关闭、日志凭据和文件句柄问题。这些失败是反例证据；当前最终工作树的上述检查均通过。

例如：迁移首行故障仍执行并记录 2 项迁移；并发文件创建有 8 个成功创建者；MySQL 部分初始化后连接数从 1 变 2；重复 InfluxDB Close 触发 `send on closed channel`。所有这些当前回归都已通过。

## 未运行与限制

未执行真实 RabbitMQ broker、真实 NATS/InfluxDB 服务的整套集成、Windows 运行、生产网络分区/负载和部署验证。消息和查询相关测试使用真实 SDK 配合受控 TCP、HTTP、Ack 或 etcd 接口夹具，不把它们称为真实生产服务验证。没有发布 Leo，也没有更新 presence-monitor 的固定依赖或 vendor。

本轮规定的本机、Linux、MySQL 和 etcd 验证均已完成，没有仍因环境限制而未完成的必需检查。后续联调可在目标服务环境运行对应包测试；发布/下游升级由后续明确任务执行。

临时 MySQL/etcd 容器、新下载的 etcd 镜像、临时凭据、一次性脚本和反事实源码副本已清理；未修改宿主全局工具环境。

## 记录与来源

- [实施计划](../../.ai/plans/audit-fix-leo-2026-10-20.md)
- [覆盖清单](../../.ai/audits/audit-fix-leo-2026-10-20/coverage.md) / [文件库存](../../.ai/audits/audit-fix-leo-2026-10-20/inventory.json)
- [问题与反证](../../.ai/audits/audit-fix-leo-2026-10-20/findings.md) / [驳回账本](../../.ai/audits/audit-fix-leo-2026-10-20/rejected.md)
- [决策](../../.ai/audits/audit-fix-leo-2026-10-20/decisions.md) / [轮次](../../.ai/audits/audit-fix-leo-2026-10-20/rounds.md)
- [uniqid 接口契约](../../components/uniqid/README.md) / [reconnect 生命周期](../../pkg/reconnect/README.md)

依赖兼容性依据：[Iris v12.2.11 go.mod](https://github.com/kataras/iris/blob/v12.2.11/go.mod)、[golog v0.1.12 level.go](https://github.com/kataras/golog/blob/v0.1.12/level.go)。查询错误契约依据：[InfluxDB v1 API](https://archive.docs.influxdata.com/influxdb/v1.0/tools/api/)。etcd 测试镜像来源：[官方 v3.6.7 发布说明](https://github.com/etcd-io/etcd/releases/tag/v3.6.7)。
