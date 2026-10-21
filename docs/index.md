# Leo 文档

Leo 是 Go 组件库，模块路径为 `github.com/puper/leo`。`engine` 负责按依赖构建组件并关闭资源，`components` 封装常用服务客户端和服务端，`pkg` 提供可独立使用的并发和重连工具。

## 阅读入口

| 文档 | 内容 |
|---|---|
| [快速上手](getting-started.md) | 最小可运行示例、配置与启动方式 |
| [架构与生命周期](architecture.md) | 依赖注册、构建失败、组件获取和关闭契约 |
| [组件指南](components.md) | 组件入口、配置、资源所有权及使用边界 |
| [开发与验证](development.md) | Go Modules、workspace、测试与集成验证 |
| [新项目骨架](../examples/project-template/README.md) | 可复制的 Gin 服务、目录组织、业务模块开发和组件集成 |

## 仓库结构

```text
engine/                    组件容器与依赖图
components/                数据库、网络、日志、存储和分布式 ID 组件
pkg/                       keyed mutex、重连、定时任务
examples/project-template/ 独立 Go module 的应用骨架
docs/                      当前使用与开发文档
go.mod、go.sum              Leo 库的依赖
go.work、go.work.sum        库与应用骨架的本地联合开发
```

依赖通过 Go Modules 获取，不维护 `vendor`。库声明 Go 1.25，联合开发工作区和模板要求 Go 1.25.5 或更高版本。具体依赖版本以 [go.mod](../go.mod) 和 [模板 go.mod](../examples/project-template/go.mod) 为准。

AI 计划与审查记录写入被 Git 忽略的 `.ai/`，对外使用说明维护在本目录；旧审计报告不是当前接口文档。
