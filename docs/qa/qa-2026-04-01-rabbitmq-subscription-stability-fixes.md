# RabbitMQ Subscription 稳定性修复记录（2026-04-01）

## 背景

`components/rabbitmq/subscription` 在启动与重连路径存在以下稳定性问题：

1. `Start()` 复用了 `CloseTimeout` 作为启动超时，且零值会立即超时。
2. 初始化阶段错误通过 `initCh` 反复发送，`Start()` 返回后可能无人接收，导致发送阻塞。
3. `deliveries` 通道关闭时未退出，可能进入空转循环占用 CPU。
4. `ReconnectDelay` 与 `CloseTimeout` 的零值缺少兜底，易出现重连热循环或关闭等待不稳定。

## 修复设计

- 新增 `StartTimeout` 配置项，启动等待与关闭等待解耦。
- 初始化通知改为“仅首个结果触发一次”，并使用非阻塞发送避免 goroutine 卡死。
- `deliveries` 关闭时立即返回错误，交由外层重连逻辑处理。
- 为 `StartTimeout`、`CloseTimeout`、`ReconnectDelay` 增加默认值：
  - `StartTimeout`: `10s`
  - `CloseTimeout`: `10s`
  - `ReconnectDelay`: `1s`

## 任务状态

- [x] 文档更新
- [x] 代码实现
- [x] 单元测试补充
- [ ] 全量回归（受本地环境限制）
