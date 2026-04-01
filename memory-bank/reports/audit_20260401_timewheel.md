# TimeWheel 组件代码审计报告

**审计日期**: 2026-04-01  
**审计范围**: `/Users/puper/Documents/projects/leo/pkg/timewheel/component.go`  
**审计方法**: 多智能体对抗性审查 + 单元测试验证

---

## 审计摘要

| 问题类型 | 数量 | 风险等级 |
|----------|------|----------|
| 逻辑 Bug | 1 | CRITICAL |
| 设计缺陷 | 2 | MEDIUM |
| 误报 | 3 | - |

---

## 一、确认的 Bug

### BUG-1: Purge 操作导致 Nil Pointer Panic (CRITICAL)

**位置**: `component.go:167`

**问题描述**:  
当执行 `Purge()` 操作时，传入的 `job` 参数为 `nil`。但 `mainloop` 中的请求处理逻辑在处理 Purge (action=2) 时，仍然尝试访问 `req.job.Key` 和 `req.job.Id`，导致 nil pointer dereference panic。

**触发条件**:  
```go
tw := New(1000, 1000)
tw.Purge() // 会触发 panic
```

**修复方案**:  
将 `mapKey` 的计算移到 action 判断内部，仅在 action 为 0 或 1 时计算（Delete 和 Add 操作需要 mapKey，Purge 不需要）。

**验证状态**: 已修复并通过测试。

---

## 二、设计缺陷 (MEDIUM)

### ISSUE-1: 时间回拨导致任务延迟

**位置**: `component.go:155`

**问题描述**:  
当系统时间被回拨（NTP 同步或手动调整）时，如果 `lastTime > now.Unix()`，则时间轮循环 `for jobTime := lastTime + 1; jobTime <= now.Unix(); jobTime++` 不会执行任何操作。虽然已添加到 `expiredJobTimes` 的任务会在下次 tick 时处理，但正常排定的任务会被延迟执行。

**影响**:  
任务最多延迟 600ms（一个 tick 周期）才被执行。

**风险等级**: MEDIUM

**建议**:  
在设计文档中明确时间回拨的影响，或考虑使用单调时钟替代 wall clock。

---

### ISSUE-2: Close 时存在理论竞态风险

**位置**: `component.go:68-74, 84-105`

**问题描述**:  
`Close()` 关闭 `closed` 和 `dispatchClosed` channels，然后等待 `done`。`dispatch` goroutine 在收到 `dispatchClosed` 信号后进入 drain 模式，清空 `dispatchJobs`。如果 `dispatchJobs` buffer 较小且 dispatch 回调处理速度慢，mainloop 在发送 job 到 `dispatchJobs` 时可能阻塞。

**风险等级**: MEDIUM（理论上，在 buffer 充足时风险较低）

**建议**:  
考虑为 `dispatchJobs` 使用更大的 buffer，或在 Close 时增加超时机制。

---

## 三、误报 (已排除)

以下问题经对抗性审查确认为误报：

1. **mainloop 竞态条件**: `select` 语句保证每次只能执行一个 case，Add 操作和遍历操作不会并发。
2. **请求池泄漏**: `request` 结构体只有 `action` 和 `job` 两个字段，且每次复用前都会重置。
3. **nil map 访问**: Go 的 map delete 操作是安全的，且没有外部 goroutine 并发访问 `jobsByTime`。

---

## 四、测试覆盖

已生成单元测试文件 `component_test.go`，覆盖以下场景：

| 测试用例 | 描述 | 状态 |
|----------|------|------|
| TestAddAndDispatch | 验证任务添加和回调触发 | PASS |
| TestDelete | 验证任务删除 | PASS |
| TestPurge | 验证 Purge 不触发回调（修复后） | PASS |
| TestConcurrentAddDelete | 验证并发 Add/Delete 安全性 | PASS |
| TestSubscribeUnsubscribe | 验证订阅/取消订阅 | PASS |
| TestClose | 验证 Close 后回调完成 | PASS |
| TestCloseIdempotent | 验证 Close 幂等性 | PASS |

---

## 五、修复清单

- [x] BUG-1: Purge nil pointer panic 已修复
- [ ] ISSUE-1: 时间回拨问题 - 待设计决策
- [ ] ISSUE-2: Close 竞态风险 - 低优先级

---

## 六、审计结论

TimeWheel 组件在大部分场景下能正常工作，但存在一个 **CRITICAL** 级别的 bug（Purge panic）必须立即修复。该 bug 在审计测试中被发现并已修复。

另外两个设计缺陷风险等级为 MEDIUM，属于可接受的范围或需要业务场景确认后再做决策。
