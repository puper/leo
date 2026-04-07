# Reconnect 稳定性修复记录（2026-04-01）

## 背景

`pkg/reconnect` 在高并发和异常重连场景下存在以下问题：

1. `connectLoop` 与 `reconnect` 都会调用 `Connect`，重连成功后下一轮又立即 `Connect`，可能导致重复建连。
2. `WaitReconnect()` 仅等待 `connected=true`，缺少关闭/停止退出条件，失败或关闭场景可能永久阻塞。
3. `tryRefresh()` 会重置 `ctx/cancel`，与主循环并发读取存在竞态风险。

## 修复设计

### 1) 统一连接状态机，消除双重 Connect

- 将重连退避逻辑内聚到 `connectLoop`：
  - 首次连接：不延迟直接尝试。
  - 重连阶段：先按 backoff 等待，再执行一次 `Connect`。
- `reconnect()` 不再承担连接动作，避免“重连成功后额外再连一次”。

### 2) WaitReconnect 增加可退出条件

- 引入运行状态位（关闭中/已停止），并在状态变化时 `Broadcast`。
- `WaitReconnect()` 等待条件调整为：`connected == true` 或组件进入终止态。

### 3) 保护上下文生命周期操作

- 为 `ctx/cancel` 增加独立锁与访问辅助函数，避免 `tryRefresh` 与主循环并发读写产生竞态。
- `Close()` 进入关闭态后禁止 `tryRefresh()` 重置上下文，防止关闭过程被“反向拉起”。

## 回归测试计划

- 新增测试：验证一次断连恢复只发生一次重连 `Connect`（无重复连接）。
- 新增测试：验证 `WaitReconnect()` 在 `Close()` 后可及时返回。
- 保留现有重连、退避、健康检查、Client 封装用例。

## 任务状态

- [x] 设计文档更新
- [x] 代码实现
- [x] 单元测试补充
- [ ] `go test ./...` 全量回归（受本地 Go 环境限制）
