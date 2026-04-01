# 并发关闭竞态修复记录（2026-04-01）

## 背景

本次修复覆盖两个并发安全问题：

1. `pkg/mutexmanager` 在 `Unlock` 时可能提前删除仍有待读锁的条目，导致后续 `RUnlock` 走到“key 不存在”分支并 panic。
2. `pkg/timewheel` 的 `Close()` 仅等待 `dispatch` 退出，未与 `mainloop` 建立完整退出握手，关闭期间存在并发时序不确定性。

## 问题分析

### 1) MutexManager 提前删除条目

- 当前行为：`Unlock` 在 `locks == 0` 时直接删除 `mutexes[key]`。
- 风险时序：
  1. writer 持有写锁；
  2. reader 调用 `RLock`，先 `rlocks++`，再阻塞于底层 `RWMutex.RLock()`；
  3. writer `Unlock` 将 `locks` 减到 0 并删除条目；
  4. reader 后续 `RUnlock` 找不到 key，触发 panic。
- 修复原则：删除条目必须同时满足 `locks == 0 && rlocks == 0`。

### 2) TimeWheel 关闭握手不完整

- 当前行为：`Close` 关闭 `closed` 与 `dispatchClosed` 后，只等待 `dispatch` 的 `done`。
- 风险时序：
  1. `dispatch` 在收到 `dispatchClosed` 后，按“当前队列为空”即退出；
  2. `mainloop` 可能仍在 tick 分支继续向 `dispatchJobs` 发送；
  3. 关闭期间可能出现任务丢失、阻塞或 goroutine 泄漏风险。
- 修复原则：
  1. 建立 `mainloop` 与 `dispatch` 的顺序退出关系；
  2. 保证 `dispatch` 的退出条件来自“生产端结束”（关闭通道），而非“瞬时队列为空”。

## 修复方案

- `pkg/mutexmanager/component.go`
  - 将 `Unlock` 删除条件统一为 `locks == 0 && rlocks == 0`。

- `pkg/timewheel/component.go`
  - 新增 `mainloopDone` 信号；
  - `Close()` 等待 `mainloop` 与 `dispatch` 均退出；
  - 由 `mainloop` 退出时关闭 `dispatchJobs`，`dispatch` 改为 `range dispatchJobs` 直到生产端结束。

## 回归策略

- `mutexmanager`
  - 新增用例覆盖“writer 持锁 + reader 先计数后阻塞”的路径，验证不会出现 `r_unlock of unlocked mutex`。

- `timewheel`
  - 新增用例验证 `Close()` 返回时，`mainloop` 与 `dispatch` 均已完成退出。

## 任务状态

- [x] 已完成 `MutexManager.Unlock` 删除条件修复
- [x] 已完成 `TimeWheel.Close` 完整关闭握手修复
- [x] 已完成对应单元测试补充与执行

## 验证结果

- 执行命令：`GOCACHE=/tmp/go-build-cache /usr/local/go/bin/go test ./pkg/mutexmanager ./pkg/timewheel`
- 结果：`pkg/mutexmanager`、`pkg/timewheel` 均通过。
