# AGENTS.md

## 项目概览

Go + Gin 服务模板，模块路径 `server`，入口 `main.go`，CLI 使用 Cobra。默认无数据库启动，保留 JWT、日志，以及可选 MySQL 迁移和示例创建/详情接口。

在 Leo 仓库中位于 `examples/project-template`，通过根 go.work 联合开发；复制到独立目录后使用本模块固定的 Leo 发布版。不要给可复制模板写入开发者本机绝对路径或本地 replace。

## 开发流程

- 修改前阅读相关代码、调用方、测试和配置。
- 修改前创建或更新 `.ai/plans/<task-name>.md`，记录目标与非目标、当前理解、实施计划、验证计划、风险与影响、执行调整。
- 仅进行当前任务必要修改；不要引入无实际用途的抽象、依赖或兼容分支。
- 文档与业务注释使用中文，代码通过 gofmt。
- 不提交真实配置、凭据、生成的二进制或本地运行数据。

## 验证

```bash
make build
make test
make lint
```

`make lint` 执行 go vet。数据库集成测试需要隔离 MySQL 数据库及 `TEST_MYSQL_DSN`，详见 README；不得针对生产数据执行测试。

## 变更守卫

- API 变更同步 Swagger 注释，执行 `make swagger` 并提交 `internal/swagger/docs/`。
- 仅启用 MySQL 示例时使用 `migrates/sqls/server/*.up.sql`；新项目选择其他数据库时同步替换连接、模型和迁移，不改已执行迁移。
- 缺省 db 表示未启用；显式配置错误不能降级为无数据库成功启动。
- 配置变更同步 `config/config.example.toml` 与 README。
- 保留真实错误语义，不用空值或成功响应掩盖失败。
- 最终说明修改、计划文件路径、实际验证结果和未完成事项。

## 文档

- [README](README.md)：初始化、运行、认证、迁移、测试及 Docker。
- [架构](docs/design/architecture.md)：模块职责与请求链路。
- [组件示例](docs/components.md)：开发步骤和实际组件应用。
- [约定](docs/conventions/index.md)：实现与协作约定。
