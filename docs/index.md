# 文档索引 (Documentation Index)

## 导航地图

| 区域 | 路径 | 说明 |
|------|------|------|
| **架构设计** | [docs/design/architecture.md](design/architecture.md) | 核心设计与组件关系 |
| **组件文档** | `components/*/` | 各组件使用指南 |
| **约定规范** | [docs/conventions/](conventions/) | 代码风格、语言约定 |
| **工作流程** | [docs/workflows/](workflows/) | 研究/计划/执行/QA 流程 |
| **缺陷修复记录** | [docs/qa/](qa/) | 已确认问题、修复方案与回归结果 |
| **决策记录** | [docs/decisions/](decisions/) | 重要技术决策 |
| **参考资料** | [docs/refs/](refs/) | 外部链接、证据 |

## 快速链接

- [Engine 核心机制](../engine/engine.go) - 依赖注入与生命周期
- [DB 组件示例](../components/db/) - Builder 模式参考
- [命令行工具](../README.md) - 项目构建命令

## 关键变更日志

- 2026-04-01: 修复 `rabbitmq/subscription` 启动超时与关闭超时耦合问题，补齐初始化通知单次发送与 `deliveries` 关闭退出逻辑，避免阻塞与空转。
- 2026-04-01: 修复 `pkg/reconnect` 连接状态机，消除重连阶段重复 `Connect`；补齐 `WaitReconnect` 关闭退出条件并收敛 `ctx/cancel` 并发访问。
- 2026-04-01: 退役 `pkg/reconnectable` 组件；`iris/web` 的 Builder 启动检查窗口调整为可配置并提高默认等待时长，降低启动误判概率。
- 2026-04-01: 修复 `mutexmanager` 在写锁释放时误删仍有待读锁条目的问题；重构 `timewheel` 关闭握手，确保 `Close()` 等待 `mainloop` 与 `dispatch` 完整退出。
- 2026-03-31: 启动一轮稳定性修复，覆盖 `reconnectable` 启动阻塞、`influxdb` 反序列化目标错误、`iris` 启停行为、`engine` 缺失依赖构建器检测、`localfile` 路径归属校验与 `db` 迁移加载健壮性。
