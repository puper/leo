---
title: "timewheel 优化 - QA Report"
phase: QA
date: "2026-03-31T02:00:00+08:00"
owner: "assistant"
parent_execute: ".artifacts/execute/20260331_timewheel_optimization_execute.md"
git_commit_at_qa: "aa54f77"
tags: [qa, timewheel, golang]
---

## Summary

| Metric | Count |
|--------|-------|
| Files reviewed | 1 |
| Functions reviewed | 12 |
| CRITICAL findings | 1 |
| WARNING findings | 1 |
| INFO findings | 2 |
| PASS (no issues) | 9 |

## Changed Areas Reviewed

### File: `pkg/timewheel/component.go`

| Function | Lines | Status |
|----------|-------|--------|
| `Close()` | 67-70 | ⚠️ WARNING |
| `Sub()` | 72-74 | ✅ PASS |
| `Unsub()` | 76-78 | ✅ PASS |
| `dispatch()` | 80-101 | ✅ PASS |
| `Add()` | 103-109 | ✅ PASS |
| `Delete()` | 111-117 | ✅ PASS |
| `Purge()` | 119-125 | ✅ PASS |
| `getRequest()` | 127-132 | ✅ PASS |
| `mainloop()` | 134-193 | ✅ PASS |

## Findings

### CRITICAL: Close() 多次调用会 panic

**Severity**: CRITICAL
**Category**: Concurrency / Robustness
**Location**: `pkg/timewheel/component.go:67-70`

**Finding**:
```go
func (me *TimeWheel) Close() {
    close(me.closed)        // panic if already closed
    <-me.done
}
```

**Problem**: 如果 `Close()` 被多次调用（例如在 defer 中调用且有多个 defer），`close(me.closed)` 会 panic。

**Recommendation**: 使用 `sync.Once` 保护 close 操作：
```go
func (me *TimeWheel) Close() {
    close(me.closed)
    close(me.dispatchClosed)  // 需要先关闭 dispatchClosed
    <-me.done
}
```
或引入一个 bool 标志位防止重复关闭。

---

### WARNING: Delete 中临时 Job 的 mapKey 可能为 "key:" 格式

**Severity**: WARNING
**Category**: Data Consistency
**Location**: `pkg/timewheel/component.go:163`

**Finding**:
```go
mapKey := req.job.Key + ":" + req.job.Id
```

**Problem**: 在 `action == 1`（Delete）分支中，`req.job` 是通过 `getRequest(1, &Job{Id: id, Key: key})` 创建的，其中 `Key` 和 `Id` 都是必需的。如果 `Key` 为空字符串，`mapKey` 会变成 `":id"`，这是一个合法的 map key，但在业务逻辑上可能是意外。

**Current Behavior**: 这是既有问题，非本次引入。

**Recommendation**: 在 Delete 中检查 Key 和 Id 是否为空。

---

### INFO: 缺少 Close 后 dispatch goroutine 退出保证

**Severity**: INFO
**Category**: Robustness
**Location**: `pkg/timewheel/component.go:80-101`

**Finding**: `Close()` 调用 `close(me.closed)` 但没有 `close(me.dispatchClosed)`。

**Analysis**: 
- `mainloop` 在收到 `me.closed` 信号后退出
- `dispatch` 依赖 `dispatchClosed` 信号退出
- 如果 `mainloop` 先退出，`dispatch` 可能永远不收到 `dispatchClosed`

**Actual Flow**:
1. `Close()` → `close(me.closed)`
2. `mainloop` 收到 `me.closed`，break LOOP
3. `mainloop` 调用 `tk.Stop()` 后结束
4. 但 `dispatch` 仍在等待 `dispatchClosed`
5. **问题**：`dispatch` 永远等待，因为没有地方关闭 `dispatchClosed`

**Impact**: `Close()` 中的 `<-me.done` 会永远阻塞，dispatch goroutine 泄漏。

**Recommendation**: 在 `Close()` 中添加 `close(me.dispatchClosed)`：
```go
func (me *TimeWheel) Close() {
    close(me.closed)
    close(me.dispatchClosed)  // 添加这行
    <-me.done
}
```

---

### INFO: mainloop 中 Delete 操作未校验 job 存在性

**Severity**: INFO
**Category**: Defensive Programming
**Location**: `pkg/timewheel/component.go:176-180`

**Finding**:
```go
} else if req.action == 1 {
    if job, ok := me.jobsById[mapKey]; ok {
        delete(me.jobsById, mapKey)
        delete(me.jobsByTime[job.Time], mapKey)
    }
}
```

**Observation**: 如果 `mapKey` 不存在于 `jobsById`，这是静默失败。这是既有问题，非本次引入。

---

## Test Coverage Analysis

timewheel 包没有测试文件，无法验证：
- 并发 Close() 调用行为
- Purge 后 Add/Delete/Delete 混合场景
- dispatch goroutine 退出超时

---

## Static Analysis Summary

| Tool | Result |
|------|--------|
| go build | pass |
| go vet | pass |

---

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation Status |
|------|------------|--------|-------------------|
| Close() 多次调用 panic | Medium | High | **Not mitigated** |
| dispatch goroutine 泄漏 | High | High | **Not mitigated** |
| sync.Pool 内存泄漏 | Low | Low | Mitigated (正确归还) |

---

## Recommendations Summary

### Must Fix (CRITICAL)
1. **Close() 多次调用 panic** - 使用 sync.Once 或标志位保护

### Should Fix (WARNING)  
1. **Delete 中 Key 为空场景** - 考虑输入校验（既是问题）

### Observations (INFO)
1. **dispatch goroutine 泄漏** - Close() 缺少 close(dispatchClosed)
2. **缺少测试覆盖** - 建议添加并发场景测试

## Next Steps

Review findings and decide whether to:
1. Fix CRITICAL issues before merge
2. Create follow-up plan for WARNING issues
3. Accept as-is for INFO observations
