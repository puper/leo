---
title: "mutexmanager 优化实现计划"
link: "mutexmanager-optimization-plan"
type: implementation_plan
ontological_relations:
  - relates_to: [[mutexmanager-implementation]]
tags: [plan, mutexmanager, optimization]
uuid: "mutexmanager-optimization-20260331"
created_at: "2026-03-31T00:00:00+08:00"
git_commit_at_plan: "d4e0bb52"
---

## Goal

优化 mutexmanager 组件，提升并发性能并增加功能：
- 修复并发安全隐患
- 添加读锁（RLock）支持
- 提升高并发场景性能
- 增加 TryLock 能力

## Scope & Assumptions

- IN scope: mutexmanager 组件优化
- OUT of scope: 其他组件修改、benchmark 测试、性能对比
- Assumptions: 使用 Go 1.21+，sync.Map 已稳定

## Deliverables

- 优化后的 `pkg/mutexmanager/component.go`
- 单元测试 `pkg/mutexmanager/component_test.go`

## Readiness

- mutexmanager 实现已存在于 `pkg/mutexmanager/component.go`
- 无外部依赖需要准备

## Milestones

- M1: 修复并发安全隐患，重构锁逻辑
- M2: 添加 RLock/RUnlock 读写锁支持
- M3: 添加 TryLock 和 LockTimeout 功能
- M4: 编写单元测试验证

## Work Breakdown (Tasks)

### T001: 修复 Lock/Unlock 并发安全隐患

**Summary**: 重构 Lock/Unlock 方法，消除锁释放后的竞态窗口

**Changes**:
1. 修改 `Lock()` 方法：在持有 `me.mutex` 期间完成目标 mutex 的获取
2. 修改 `Unlock()` 方法：使用原子操作或额外标记确保安全
3. 添加 `locks` 溢出的防御性检查

**Acceptance**: 多个 goroutine 并发 Lock/Unlock 同一 key 不会导致 panic 或死锁
**Evidence Contract**: `go test -race ./pkg/mutexmanager/...`
**Files**: pkg/mutexmanager/component.go
**Milestone**: M1
**Estimate**: 1 小时
**Dependencies**: 无

---

### T002: 添加 RLock/RUnlock 支持

**Summary**: 为 MutexManager 添加读锁支持，提升读多写少场景性能

**Changes**:
1. 添加 `RLock(key string)` 方法
2. 添加 `RUnlock(key string)` 方法
3. 使用 `sync.RWMutex` 替代 `sync.Mutex` 作为内部锁

**Acceptance**: RLock 允许多个并发读，WriteLock 独占
**Evidence Contract**: `go test -v -run TestRLock ./pkg/mutexmanager/...`
**Files**: pkg/mutexmanager/component.go
**Milestone**: M2
**Estimate**: 1 小时
**Dependencies**: T001

---

### T003: 添加 TryLock 功能

**Summary**: 实现非阻塞获取锁的能力

**Changes**:
1. 添加 `TryLock(key string) bool` 方法 - 尝试获取写锁
2. 添加 `TryRLock(key string) bool` 方法 - 尝试获取读锁

**Acceptance**: TryLock 在锁不可用时立即返回 false，不阻塞
**Evidence Contract**: `go test -v -run TestTryLock ./pkg/mutexmanager/...`
**Files**: pkg/mutexmanager/component.go
**Milestone**: M3
**Estimate**: 30 分钟
**Dependencies**: T002

---

### T004: 添加 LockTimeout 功能

**Summary**: 实现带超时机制的锁获取

**Changes**:
1. 添加 `LockTimeout(key string, timeout time.Duration) bool` 方法
2. 添加 `RLockTimeout(key string, timeout time.Duration) bool` 方法

**Acceptance**: LockTimeout 在超时后返回 false，timeout 参数生效
**Evidence Contract**: `go test -v -run TestLockTimeout ./pkg/mutexmanager/...`
**Files**: pkg/mutexmanager/component.go
**Milestone**: M3
**Estimate**: 30 分钟
**Dependencies**: T003

---

### T005: 编写单元测试

**Summary**: 编写完整的单元测试覆盖所有功能

**Changes**:
1. 编写 `TestLockUnlock` - 基本 Lock/Unlock 测试
2. 编写 `TestRLock` - 读写锁并发测试
3. 编写 `TestTryLock` - TryLock 功能测试
4. 编写 `TestLockTimeout` - 超时功能测试
5. 使用 `-race` 标志确保无数据竞争

**Acceptance**: 所有测试通过，无 race 检测警告
**Evidence Contract**: `go test -race ./pkg/mutexmanager/...`
**Files**: pkg/mutexmanager/component_test.go (新建)
**Milestone**: M4
**Estimate**: 2 小时
**Dependencies**: T004

---

## Risks & Mitigations

- **风险**: 重构可能引入新的并发问题
  - **缓解**: 每次重构后运行 `go test -race`
- **风险**: sync.RWMutex 在读多写少时可能优于 sync.Map 分段锁
  - **缓解**: 保持当前 map 结构，仅在 mutex 层面使用 RWMutex

## Test Strategy

- 使用表驱动测试（table-driven tests）
- 每个新增方法对应一个测试函数
- 使用 goroutine 并发测试 RLock
- `-race` 标志强制开启竞态检测

## References

- 当前实现: `pkg/mutexmanager/component.go:1-62`
- Go sync.RWMutex 文档: https://pkg.go.dev/sync#RWMutex
- Go sync.Map 文档: https://pkg.go.dev/sync#Map

## Final Gate

- **Output summary**: plan path, milestone count, tasks ready
- **Next step**: 使用 execute-phase 执行计划
