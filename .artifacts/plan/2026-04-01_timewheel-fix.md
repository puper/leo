---
title: "timewheel 组件修复计划"
link: "timewheel-fix"
type: implementation_plan
ontological_relations:
  - relates_to: "pkg/timewheel/component.go 审查"
tags: [plan, timewheel, bugfix, golang]
uuid: "timewheel-fix-20260401"
created_at: "2026-04-01T12:00:00+08:00"
parent_research: "审查任务输出"
git_commit_at_plan: "pending"
---

## Goal

修复时间轮组件的两处风险：
1. **Close 竞态死锁**（CRITICAL）- dispatchJobs 可能导致死锁
2. **时间回拨处理**（MEDIUM）- 使用 monotonic tick 计数替代绝对时间戳

## Scope & Assumptions

- **IN scope**: 
  - 重构 `mainloop()` 和 `dispatch()` 的关闭逻辑
  - 使用 tick 计数替代 `time.Now().Unix()` 作为时间度量
  - 确保所有 channel 正确关闭，无死锁风险
- **OUT of scope**: 
  - 不修复测试文件
  - 不添加新功能
  - 不改变 API 接口（Add/Delete/Purge/Close 签名不变）
- **Assumptions**:
  - Go 1.21+
  - 使用 Go 原生 `time.Ticker`（不引入第三方库）

## Deliverables

- 修改后的 `pkg/timewheel/component.go`
- 修复后的单元测试（验证关闭流程无死锁）

## Readiness

- 源代码已读取：`pkg/timewheel/component.go` (198 行)
- 测试文件存在：`pkg/timewheel/component_test.go`

## Milestones

- **M1**: 修复 Close 竞态死锁问题
- **M2**: 使用 tick 计数替代绝对时间
- **M3**: 验证修复无回归

## Work Breakdown (Tasks)

### T001: 引入 tick 计数机制

**Summary**: 在 TimeWheel 结构体中添加 tickCount 字段，替代 lastTime 作为时间度量

**Files**:
- `pkg/timewheel/component.go` (修改)

**Changes**:
1. 添加字段 `tickCount int64` 到 TimeWheel 结构体
2. 删除 `lastTime int64` 字段
3. 将所有 `lastTime` 引用改为 `tickCount`

**Acceptance**: 代码编译通过，`go build ./pkg/timewheel/...` 无错误

**Dependencies**: 无

**Milestone**: M1

**Estimate**: 10 分钟

---

### T002: 重构 mainloop 的 tick 处理逻辑

**Summary**: 将基于绝对时间 `now.Unix()` 的遍历改为基于 tick 计数

**Files**:
- `pkg/timewheel/component.go` (修改)

**Changes**:
1. Line 139: `tk := time.NewTicker(...)` 保持不变，但用于触发 tick
2. Line 140: 删除 `lastTime := time.Now().Unix()`
3. Line 145-165: 重写 tick 处理逻辑：
   - 每次 tick递增 `me.tickCount`
   - 遍历所有 job 时间 <= tickCount 的任务并执行
   - 使用 `for jobTime <= me.tickCount` 而非 `now.Unix()`

**Acceptance**: `go build ./pkg/timewheel/...` 编译通过

**Dependencies**: T001

**Milestone**: M1

**Estimate**: 30 分钟

---

### T003: 修复 Close 竞态 - 重构 dispatch 关闭逻辑

**Summary**: 使用 sync.WaitGroup 确保 mainloop 和 dispatch 正确同步

**Files**:
- `pkg/timewheel/component.go` (修改)

**Changes**:
1. 添加 `wg sync.WaitGroup` 字段到 TimeWheel
2. mainloop 启动时 `wg.Add(1)`，退出时 `wg.Done()`
3. dispatch 启动时 `wg.Add(1)`，退出时 `wg.Done()`
4. Close() 中 `close(me.closed)` 后添加 `me.wg.Wait()` 等待两个 goroutine 退出
5. 删除原有的 `done` channel（改用 WaitGroup）

**Acceptance**: `go build ./pkg/timewheel/...` 编译通过

**Dependencies**: 无（独立于 T001/T002）

**Milestone**: M1

**Estimate**: 20 分钟

---

### T004: 修复 Close 竞态 - 确保 dispatchJobs 正确 drain

**Summary**: 解决 dispatchJobs 可能满导致的发送阻塞

**Files**:
- `pkg/timewheel/component.go` (修改)

**Changes**:
1. dispatch 中收到 `dispatchClosed` 信号后，使用 `for len(me.dispatchJobs) > 0` 循环 drain
2. 不再使用 select+default 组合，改用确定性 drain
3. mainloop 在 `closed` 信号后应**停止向 dispatchJobs 发送**

**关键修复**：mainloop 需要在 break LOOP 前停止发送任务，或使用非阻塞发送：
```go
// 在 mainloop 的 case <-me.closed 分支中
select {
case me.dispatchJobs <- job: // 尝试发送
default: // 如果 buffer 满，跳过（不阻塞）
}
```

**Acceptance**: `go build ./pkg/timewheel/...` 编译通过

**Dependencies**: T003

**Milestone**: M1

**Estimate**: 20 分钟

---

### T005: 修复 Add 请求中的时间回拨处理

**Summary**: Add 请求中的 `lastTime` 引用改为 `tickCount`

**Files**:
- `pkg/timewheel/component.go` (修改)

**Changes**:
1. Line 177-179: 将 `req.job.Time <= lastTime` 改为 `req.job.Time <= me.tickCount`
2. job.Time 字段的含义变为"目标 tick 计数"而非"Unix 时间戳"
3. 调用方需要传入 tick 计数而非 Unix 时间戳

**Breaking Change**: `Job.Time` 的语义从 Unix 时间戳变为 tick 计数

**Acceptance**: `go build ./pkg/timewheel/...` 编译通过

**Dependencies**: T002

**Milestone**: M2

**Estimate**: 10 分钟

---

### T006: 更新 Job 结构体注释

**Summary**: 明确 Job.Time 字段的语义变更

**Files**:
- `pkg/timewheel/component.go` (修改)

**Changes**:
1. Line 27-32: 更新 Job 结构体注释，说明 Time 字段现在是 tick 计数

**Acceptance**: 注释清晰描述语义

**Dependencies**: T005

**Milestone**: M2

**Estimate**: 5 分钟

---

### T007: 编写关闭流程的并发测试

**Summary**: 验证 Close 调用不会死锁

**Files**:
- `pkg/timewheel/component_test.go` (修改)

**Changes**:
1. 添加 TestCloseNoDeadlock 测试：
   - 启动多个 goroutine 并发 Add 任务
   - 在后台持续添加任务
   - 调用 Close
   - 使用 select + time.After 防止无限等待
   - 如果 5 秒内未返回，测试失败

**Acceptance**: `go test ./pkg/timewheel/... -v -run TestCloseNoDeadlock -timeout 10s`

**Dependencies**: T001, T002, T003, T004

**Milestone**: M3

**Estimate**: 20 分钟

---

## Risks & Mitigations

| 风险 | 描述 | 缓解 |
|------|------|------|
| 语义变更 | Job.Time 从 Unix 时间戳变为 tick 计数，可能破坏现有调用方 | 在注释中明确说明，提供迁移指南 |
| 测试覆盖不足 | 只验证了关闭场景，未验证时间回拨场景 | 添加时间回拨模拟测试（可选） |

## Test Strategy

- **主测试**: `TestCloseNoDeadlock` - 验证并发 Add + Close 无死锁
- **回归测试**: 运行现有测试确保无破坏
- **命令**: `go test ./pkg/timewheel/... -v -timeout 30s`

## References

- 审查结果输出（对话）
- `pkg/timewheel/component.go:84-105` (dispatch 函数)
- `pkg/timewheel/component.go:138-197` (mainloop 函数)

## Final Gate

- **Output summary**: 计划路径 `.artifacts/plan/2026-04-01_timewheel-fix.md`
- **Milestone count**: 3
- **Task count**: 7
- **Git state**: 待执行前检查

## 重要说明

### T005 的 Breaking Change

Job.Time 语义变更可能影响调用方。当前调用方传入的是 Unix 时间戳：
```go
&Job{
    Key:  "my-key",
    Id:   "job-1",
    Time: time.Now().Unix(), // 当前用法
}
```

修复后需要传入 tick 计数。这需要：
1. TimeWheel 导出获取当前 tickCount 的方法
2. 或要求调用方自行管理时间转换

### 替代方案（不破坏 API）

如果不想破坏 API，可以保留 `Job.Time` 为 Unix 时间戳，但在内部维护一个映射表 `jobTime -> tickCount`。但这会增加复杂度。

**建议**：采用 Breaking Change 方案，并在文档中说明迁移方法。
