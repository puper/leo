# 项目约定

- 文档与业务注释使用中文；Go 源码使用 gofmt。
- 修改前在 `.ai/plans/` 记录计划，修改后执行 `make build`、`make test`、`make lint`。
- 错误通过 `error` 向上返回；业务错误使用 `internal/derrors`，不得吞掉数据库错误或返回假成功。
- HTTP 状态码与 JSON 错误码保持一致；ID 通过 JSON 字符串传递。
- 新增接口同步 Swagger 注释与生成文件；沿用 ServiceCaller 实际方法签名。
- 数据库是可选能力；MySQL 示例不能只改 driver 来支持其他数据库。
- 启用持久化时，数据模型变化新增对应数据库的版本迁移，不修改已执行迁移；高风险迁移须单独审查。
- 配置变化同步示例与 README；真实配置、密钥不进入 Git 或镜像。
- 测试要验证实际组件行为，避免只测试复制出来的配置；数据库测试仅使用隔离数据库。
- 依赖通过 go.mod/go.sum 管理，不提交 vendor，不安装全局开发工具。
