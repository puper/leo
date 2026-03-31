---
title: "timewheel 性能优化计划"
link: "timewheel-optimization-plan"
type: implementation_plan
ontological_relations:
  - relates_to: "timewheel-analysis"
tags: [plan, timewheel, optimization, golang]
uuid: "20260331-timewheel-optimization"
created_at: "2026-03-31T00:00:00+08:00"
git_commit_at_plan: "d4e0bb5"
---

## Goal

优化 `pkg/timewheel/component.go` 的时间轮实现，修复内存泄漏风险、消除锁竞争、补充缺失功能。

## Scope & Assumptions

**IN scope:**
- `pkg/timewheel/component.go` 源码优化
- 补充 `Purge` 功能
- 修复 dispatch goroutine 泄漏
- 降低锁竞争

**OUT of scope:**
- 不修改 vendor 中的 gcache 版本
- 不新增单元测试框架
- 不涉及部署和配置变更

**Assumptions:**
- Go 1.21+
- 使用 Go 原生 sync 包

## Deliverables

- 优化后的 `pkg/timewheel/component.go`
- 新增 `Purge()` 方法
- 修复 Close 时 dispatchJob 排空问题

## Readiness

- `pkg/timewheel/component.go` 已存在
- 现有功能需保持向后兼容

## Milestones

- **M1**: 修复 Close 泄漏 + 添加 Purge
- **M2**: 降低锁竞争（sync.Map 或分片锁）
- **M3**: 优化内存分配（对象池/预分配）

## Work Breakdown

### T001: 修复 dispatch goroutine 泄漏

**Summary**: 修改 dispatch 和 Close 逻辑，确保关闭时 dispatchJobs channel 被 close，dispatch goroutine 优雅退出

**Changes**:
1. 在 `TimeWheel` 结构体添加 `dispatchClosed chan struct{}`
2. `Close()` 时先 close `dispatchClosed`，再等待 dispatch 退出
3. `dispatch()` 中 select 监听 `dispatchClosed`，收到后排空 dispatchJobs 再 return

**Acceptance**: 并发调用 Add 和 Close 不死锁，dispatch goroutine 正确退出
**Evidence Contract**: `go build ./pkg/timewheel/...`
**Files**: pkg/timewheel/component.go
**Milestone**: M1
**Estimate**: 30min

---

### T002: 添加 Purge 功能

**Summary**: 补充 action=2 的 Purge 请求处理，清空所有待执行任务

**Changes**:
1. `Delete` 方法添加 `action` 字段支持
2. `mainloop` 中添加 `action == 2` 的 Purge 处理分支：
   ```go
   } else if req.action == 2 {
       me.jobsByTime = map[int64]map[string]*Job{}
       me.jobsById = map[string]*Job{}
   }
   ```
3. `request` 结构体添加 `Purge()` 方法便捷函数

**Acceptance**: 调用 Purge 后 jobsByTime 和 jobsById 为空
**Evidence Contract**: `go build ./pkg/timewheel/...`
**Files**: pkg/timewheel/component.go
**Milestone**: M1
**Estimate**: 15min

---

### T003: 使用 sync.Map 降低锁竞争

**Summary**: 将 `callbacks` map 替换为 `sync.Map`，消除读锁开销

**Changes**:
1. `TimeWheel.callbacks` 从 `map[string]Callback` 改为 `sync.Map`
2. `Sub()`: 使用 `callbacks.Store(key, f)` 替代赋值
3. `Unsub()`: 使用 `callbacks.Delete(key)` 替代 delete
4. `dispatch()`: 使用 `callbacks.Load(key)` 替代读锁访问

**Acceptance**: 回调注册/注销时无需加锁
**Evidence Contract**: `go build ./pkg/timewheel/...`
**Files**: pkg/timewheel/component.go
**Milestone**: M2
**Estimate**: 30min

---

### T004: 使用对象池优化内存分配

**Summary**: 用 `sync.Pool` 复用 request 对象，减少 GC 压力

**Changes**:
1. 添加包级 `sync.Pool`:
   ```go
   var requestPool = sync.Pool{
       New: func() any {
           return &request{}
       },
   }
   ```
2. `Add()` 和 `Delete()` 从 pool 获取 request 对象
3. mainloop 处理完后归还 pool

**Acceptance**: 大量 Add/Delete 场景下 GC 次数减少
**Evidence Contract**: `go build ./pkg/timewheel/...`
**Files**: pkg/timewheel/component.go
**Milestone**: M3
**Estimate**: 45min

---

### T005: 优化 expiredJobTimes 分配（低优先级）

**Summary**: 消除每次 tick 创建新 map 的开销

**Changes**:
1. 将 `expiredJobTimes` 声明在循环外，用 `clear()` 清空而非重建

**Acceptance**: tick 处理时减少一次 map 分配
**Evidence Contract**: `go build ./pkg/timewheel/...`
**Files**: pkg/timewheel/component.go
**Milestone**: M3
**Estimate**: 10min

**Note**: 该 map 最大只存跨秒的时间戳（通常 0-1 个 key），优化收益极小，可跳过

---

### T006: 移除（已实现）

**说明**: 第 130/139 行已有 `delete(me.jobsByTime, jobTime)`，到期时间槽会被正确清理。无需额外处理。

## Risks & Mitigations

| 风险 | 描述 | 缓解 |
|------|------|------|
| sync.Map 性能退化 | 少量 key 时比原生 map 慢 | 只对 callbacks 使用，key 数量少 |
| sync.Pool 内存泄漏 | 未使用对象可能被永久保留 | 控制 pool size |

## Test Strategy

无新增测试。若需验证，运行 `go build ./...` 确保编译通过。

## References

- `pkg/timewheel/component.go:76-89` - dispatch 函数
- `pkg/timewheel/component.go:116-143` - mainloop tick 处理
- `vendor/github.com/puper/gcache/timewheel/timewheel.go` - 参考实现

## Final Gate

- **Output summary**: `.artifacts/plan/20260331_timewheel_optimization.md`
- **Milestones**: 3
- **Tasks**: 5（已移除 1 个）
- **Git state**: d4e0bb5

**Next step**: 使用 `/phase-planner` 或直接进入 execute-phase 执行优化
