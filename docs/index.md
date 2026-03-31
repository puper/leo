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

- 2026-03-31: 启动一轮稳定性修复，覆盖 `reconnectable` 启动阻塞、`influxdb` 反序列化目标错误、`iris` 启停行为、`engine` 缺失依赖构建器检测、`localfile` 路径归属校验与 `db` 迁移加载健壮性。
