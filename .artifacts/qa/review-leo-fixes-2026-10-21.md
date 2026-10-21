# Leo 修复准确性与完整性复核（2026-10-21）

## 结论

迁移历史读取错误的修复准确，其他改动的主要根因处理也有当前代码与现有测试支持。但本轮确认 2 项修复遗漏：P1 1 项、P2 1 项，均已用临时补充测试复现。因此上一轮“全部修复验证、无待修复项”的完整性结论需要收窄，不能据此认定已经覆盖所有关键边界。

两项均属于现有问题没有补齐，不将其表述为已证实的新引入回归。本轮只审查；新增本计划和报告，未修改生产代码、测试、依赖、旧 QA/审计账本，也未提交、发布或更新下游。原有 67 项 .opencode 删除仍在。

## 范围与方法

- 基线 `d602fea`，审查对象为当前工作区上一轮的 24 项修复及生产代码、测试、文档和 golog vendor 差异。
- 核对现有调用方、配置、SDK 源码、失败和取消流程；重点反证并发重连、租约与节点复用、消息确认、文件边界及服务启动/关闭。
- 仓库仍有 81 个 Go 文件，其中 30 个测试文件；没有前端 UI。本轮不重新声明对无改动模块、所有 vendor 或生产部署完成了全项目审计。
- 已审阅 24 项修复记录：22 项未发现新的确认遗漏；F13、F16 各发现一项未覆盖边界。新发现计数为 P0=0、P1=1、P2=1、P3=0，置信度均为 A（运行复现和源码因果一致）。本轮修复数为 0。
- 复现通过 Go overlay 加入临时测试，没有将测试写进源码目录；完整测试正文见文末。

## R1 / P1：跨实例快速复用节点号仍会生成重复 ID

- 指纹：`uniqid / unique-id / sequential-lease-handoff / new-node-sequence-reset`。
- 位置：`components/uniqid/component.go:273-283`，尤其是新建 `snowflake.NewNode` 后立即设置 `leaseValid=true`。
- 触发：实例 A 发号并成功 Close/撤约；实例 B 随后认领相同节点号，首次发号仍落在同一毫秒。
- 实际结果：两个实例返回相同 ID。旧实例已经完全停止发号，租约没有重叠；这不依赖失效实例继续发号或机器时钟回拨。
- 预期：前后两任合法持有者生成的 ID 也必须唯一。
- 根因：保留 Node 只覆盖“同一实例、同一个节点号”的重新申请。新实例的 Node 为空，序列重新从 0 开始；没有节点号交接的时间隔离或可跨实例验证的发号上界。vendored snowflake 的 Generate 将毫秒、节点号和序列拼接，相同三元组必然重复。
- 运行证据：`TestReviewSequentialLeaseHandoffKeepsIDsUnique` 使用同一租约夹具，每次发放不同 lease ID，只允许前一个租约撤销成功后申请下一租约；实际 Go 时间未替换。普通运行和 race 运行均在实例 0/1 间复现重复 ID。
- 反证：现有 `TestEtcdLeaseAllocationRenewalAndRelease` 同时运行两个不同节点号的实例，未测试旧节点释放后由新实例快速接管。虽然某些部署的 etcd 往返耗时会超过 1ms，但代码和客户端契约没有最短耗时保证，不能把网络延迟用作唯一性保护。
- 建议：为新建节点建立跨实例复用保护，明确接管后允许发号的时间边界，或使用可验证的发号上界；补充顺序交接回归。同一实例保留计数器的现有处理应继续保留。
- 验证层级：真实组件 + SDK 接口故障夹具，非真实 etcd 交接压测。

## R2 / P2：QueryRecords 忽略畸形 tags，仍将解析失败变成成功

- 指纹：`influxdb / query-protocol / QueryRecords / invalid-tags-as-empty`。
- 位置：`components/influxdb/query.go:174-189`（新校验未覆盖 tags），以及 `:116-118` 的 `.Map()` 转换。
- 触发：HTTP 200 返回合法 JSON，但 series.tags 的类型是数组而非对象；columns/values 均合法。
- 实际结果：`Query` 返回 tags 类型解析错误；同一响应经 `QueryRecords` 返回 nil error，结果为 `[{Host: Value:7}]`，标签静默丢失。
- 预期：已存在但类型错误的 tags 应显式报错，不能当成没有标签。缺失或合法的可选 tags 不应被误报。
- 根因：validateQueryResponse 只检查 results、series、columns 和行长度。随后 gjson.Map 把数组当空 map，最终反序列化面对的已是“成功转换后”的残缺记录。
- 运行证据：`TestReviewQueryRecordsRejectsMalformedTags` 使用本地 HTTP 服务与真实解析代码，确认 Query 拒绝同一输入、QueryRecords 却接受。补充测试在 race 模式下失败。
- 反证：现有测试覆盖无效 JSON、error 字段、缺少 results 和行列长度，但正常 tags 只覆盖正确的字符串对象；没有 malformed tags。此项处在 F13 的错误响应验证范围内，不要求服务端正常情况下会主动返回畸形数据。
- 建议：对实际消费的 tags 补齐对象形状及值类型验证，保留正常可选字段语义；加入 Query/QueryRecords 对相同非法响应一致报错的回归。

## 原修复逐项复核

|原记录|本轮判断与正向证据|
|---|---|
|F01 迁移读取错误|准确；返回 rows.Err，真实 GORM/SQL 故障夹具覆盖首行、中途、查询、扫描及正常历史，现有测试通过|
|F02 golog 兼容|准确；模块和 vendor 版本一致，14 个 vendored 文件逐字节匹配已下载的 v0.1.12 模块；构建通过|
|F03 文件错误与目录信息|准确；不存在文件直接返回错误，IsDir 来自实际元数据，相关回归通过|
|F04 原子创建|准确；O_EXCL 避免检查/创建竞争，覆盖写路径仍显式使用 O_TRUNC；回归通过|
|F05 根目录边界|实现与声明一致；实际 I/O 使用 os.Root；外链拒绝、内部相对链接与删除配置根的回归通过|
|F06 任务键冲突|准确；二元结构体键保留两个独立字段，碰撞回归通过|
|F07 回调新增任务死锁|根因处理合理；派发和请求接收处于同一 select；无缓冲回调与关闭回归通过|
|F08 退避溢出|根因处理合理；在 duration 转换前限幅；本机回归通过，本轮未重跑 amd64|
|F09 重连与并发|原问题路径未发现新确认遗漏；连接操作由同一循环承担，状态和代次同锁发布，现有并发/健康重连/关闭测试通过|
|F10 失败构造清理|SDK、SQL pool 与受控连接测试通过，主错误仍可 errors.Is；真实 MySQL 部分初始化本轮未运行|
|F11 gRPC 关闭期限|配置期限生效并返回 DeadlineExceeded，活动 RPC 回归通过；超时分支异步强制 Stop，不声称所有用户 handler 已退出|
|F12 SetTimeout 配置保留|正常请求与 hook 回归通过；原地修改属于公开语义变化，见兼容性边界|
|F13 查询协议错误|已有覆盖通过，但不完整；R2 证明 tags 类型错误仍伪成功|
|F14 channel 确认缓存|缓存由每个 channel 创建，旧 Message 仅保留旧缓存；真实 Ack 接口夹具通过，未运行真实 broker 重连|
|F15 RabbitMQ 生命周期|首错回收、握手取消、关闭返回值及 ClearMsgs 路径与修复目标一致；现有测试通过|
|F16 租约与发号|失效 panic、有效期、快照、watch revision、关闭等现有回归通过，但节点交接唯一性不完整，见 R1|
|F17 日志文件释放|管理器拥有并关闭 lumberjack writer，不关闭 stdout；文件描述符回归通过|
|F18 信号注册释放|新增 defer signal.Stop 与注册配对；引擎测试通过，没有执行进程信号端到端验证|
|F19 Iris Runner 调用|配置的 HTTP Server 与已有 listener 通过同一 Runner 接入；编译和实际 HTTP 回归通过|
|F20 Influx HTTP 池释放|关闭自有 HTTP idle connections；成功及两种 Builder 失败路径的连接状态回归通过|
|F21 Influx 重复关闭|SDK Close 自带 flush，sync.Once 避免重复关闭；回归通过|
|F22 NATS 日志泄密|只记录错误，不再格式化连接对象；合成凭据回归通过|
|F23 Iris 启动失败清理|Build 先于端口绑定、Host 注册后发布、超时关闭 server/listener；现有 TCP/HTTP 回归通过|
|F24 文档链接|新目标存在，原审查报告导航可达；旧报告完整性结论由本报告补充修正|

## 兼容性与边界

- uniqid 失去租约时 panic 是用户已确认的选择，不作为缺陷重新提出。底层 Node 私有化也阻止绕过门控。
- restyclient.SetTimeout 从返回独立客户端改为修改原对象。当前 Leo 和 presence-monitor 源码中未找到依赖旧行为的业务调用，但不能推断其他外部调用方兼容；下游发布前需要核对。现有 QA 已披露这一变化。
- reconnect 关闭后不可再次 Start，Raw 是 wrapper 快照；localfile 拒绝绝对符号链接。这些已记录的语义不作为新的回归报告。
- TimeWheel.Close 等待回调完成，不能从同一派发回调同步等待自身关闭；现有文档已将同步 Close 的回调结束要求列为边界。未无依据扩大为新的修复。
- 本轮未确认新的已引入回归；此表与构建通过不代表所有外部消费者都已经验证。

## 验证结果

环境：Go1.27.2 / Darwin arm64，`GOTOOLCHAIN=local`、`GOPROXY=off`、`GOFLAGS=-mod=vendor`，使用项目缓存。初次 build 输出一次宿主模块 stat-cache 写入被沙箱拒绝的提示，但最终退出码为 0；随后补充验证显式设置项目 GOMODCACHE，未安装或修改宿主全局依赖。

### 已运行且通过

- `go build ./...`（退出码 0）。
- `go vet ./...`（退出码 0）。
- `go test -race -count=1 -timeout=90s ./...`（现有全量测试通过；两个真实服务用例跳过）。
- `go test -race -count=1 -timeout=30s ./pkg/reconnect -run 'TestConcurrentClientRefreshAndRaw|TestComponent_Do'`（实际匹配并发刷新测试，通过）。
- `git diff --check`。
- golog vendor 14 个文件与本地下载模块逐字节比对，差异为 0；原有 .opencode 删除仍为 67 项。

### 已运行但失败

实际复现命令：

```sh
go test -race -overlay=/private/tmp/leo-review-20261021/overlay.json -count=1 -timeout=10s ./components/uniqid ./components/influxdb -run '^TestReview' -v
```

两项分别因“交接后重复 ID”和“畸形 tags 被成功接受”失败，退出码 1。没有 race 检测器数据竞争报告；失败来自行为断言。临时 overlay 只增加测试，不替换生产代码。

### 未运行 / 环境限制

- 真实 MySQL 和 etcd 集成用例通过 `go test -count=1 ./components/db ./components/uniqid -run 'TestPartialInitializationClosesOpenedPools|TestEtcdLeaseAllocationRenewalAndRelease' -v` 核对，本轮因没有注入隔离服务端点而明确 SKIP；不能将命令退出码 0 视为集成通过。
- 本轮未启动外部服务，也未执行真实 RabbitMQ/NATS/InfluxDB 整套联调、Linux/Windows 运行或生产网络/负载测试。上一轮服务测试记录仅作历史证据。
- 没有权限阻塞当前只读复核。补齐本轮服务覆盖需准备隔离服务，并设置对应测试环境后运行：`go test -race -count=1 -timeout=90s ./components/db ./components/uniqid`；勿把真实凭据写入报告或计划。

## 后续工作

1. 先解决 R1 的节点交接唯一性，并将顺序交接测试纳入正式回归。
2. 补齐 R2 的 tags 验证及对应协议错误回归。
3. 执行两个回归和全量 race/build/vet，另在真实 etcd 上验证节点交接。当前只读审查已经完成，两项实现修复尚未执行。

本轮计划：[review-leo-fixes-2026-10-21.md](../../.ai/plans/review-leo-fixes-2026-10-21.md)。原记录：[上一轮 QA](../../docs/qa/qa-2026-10-20-leo-audit-fixes.md)。

## 补充复现正文

以下代码是本轮实际运行的测试。恢复复现时，将两个代码块分别保存到临时目录，在 overlay 的 Replace 中映射到对应包下的新 `_test.go` 路径；不要覆盖现有文件。uniqid 测试复用已有 lifecycle_test.go 的 testComponent/testLease 夹具。临时文件在本轮收尾删除，正文保留供正式回归使用。

### uniqid/review_handoff_test.go

```go
package uniqid

import (
    "context"
    "errors"
    "sync"
    "testing"

    clientv3 "go.etcd.io/etcd/client/v3"
)

// 同一个租约夹具确保每次只存在一个租约，撤销成功后才能申请下一租约。
func TestReviewSequentialLeaseHandoffKeepsIDsUnique(t *testing.T) {
    var mu sync.Mutex
    var active, next clientv3.LeaseID
    shared := &testLease{}
    shared.grantFn = func(context.Context, int64) (*clientv3.LeaseGrantResponse, error) {
        mu.Lock()
        defer mu.Unlock()
        if active != 0 { return nil, errors.New("previous lease is still active") }
        next++
        active = next
        return &clientv3.LeaseGrantResponse{ID: active, TTL: 10}, nil
    }
    shared.revokeFn = func(_ context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
        mu.Lock()
        defer mu.Unlock()
        if active != id { return nil, errors.New("wrong lease revoked") }
        active = 0
        return &clientv3.LeaseRevokeResponse{}, nil
    }
    seen := make(map[int64]int)
    for i := 0; i < 1000; i++ {
        c, _, _ := testComponent(t)
        c.etcdCli.Lease = shared
        if err := c.Start(); err != nil { t.Fatal(err) }
        id := c.Generate().Int64()
        if err := c.Close(); err != nil { t.Fatal(err) }
        if prior, exists := seen[id]; exists {
            t.Fatalf("duplicate ID across completed lease handoff: previous instance %d, current instance %d, id %d", prior, i, id)
        }
        seen[id] = i
    }
}
```

### influxdb/review_tags_test.go

```go
package influxdb

import (
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestReviewQueryRecordsRejectsMalformedTags(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"results":[{"statement_id":0,"series":[{"name":"m","tags":["host","a"],"columns":["value"],"values":[[7]]}]}]}`))
    }))
    defer srv.Close()
    api := NewV1QueryApi(srv.Client(), srv.URL, "", "review")
    if _, err := api.Query("db", "select"); err == nil { t.Fatal("invalid fixture: Query accepted malformed tags") }
    var records []struct { Host string `json:"host"`; Value int `json:"value"` }
    if err := api.QueryRecords("db", "select", &records); err == nil {
        t.Fatalf("QueryRecords silently accepted invalid tags: records=%+v", records)
    }
}
```
