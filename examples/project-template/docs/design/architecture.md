# 架构设计

## 组件

| 目录 | 职责 |
|---|---|
| `internal/cmd` | Cobra CLI，启动失败返回非零状态 |
| `internal/config` | 启动时加载并校验 Viper 配置 |
| `internal/bootstrap` | 按依赖注册日志、ID、数据库、示例与 Web 组件 |
| `internal/app` | 向业务模块提供日志、数据库与 ID 生成接口 |
| `internal/components` | Gin HTTP 生命周期、Snowflake ID |
| `internal/logging` | 统一错误日志入口 |
| `internal/middleware` | HS256 JWT Bearer 鉴权 |
| `internal/modules/example` | 示例创建/查询、请求响应类型与模型 |
| `internal/pkg` | ServiceCaller HTTP 包装和版本信息 |
| `internal/model/dto`、`internal/derrors` | HTTP 响应与业务错误 |
| `internal/route` | 汇总模块路由注册函数 |
| `internal/swagger` | 生成命令与已提交的 API 文档 |
| `migrates` | 可选 MySQL 示例的嵌入建表迁移 |

## 请求链路

启用 MySQL 示例时：Gin → JWT 鉴权 → ServiceWrapper → ServiceCaller → ExampleService → GORM/MySQL。

默认只启动通用组件和 Web，不注册数据库示例；新模块按自身需求声明依赖。

ServiceCaller 的方法签名为 `Method(context.Context, json.RawMessage, *Reply) error`。Service 负责参数校验并传播数据库错误，包装层统一转换为 HTTP/JSON 响应。

省略 `db` 配置时不连接数据库、不执行迁移，`/readiness` 返回服务就绪。显式配置 `db` 后，MySQL 连接与迁移成功才开始监听，`/readiness` 还会检查主库；Swagger UI 仅开发环境开放，不挂载 pprof 或 statsviz 调试端点。

配置只在启动时加载，修改后重启。日志从 leo 日志组件获取，按名称选择 logger。数据库连接名 `server` 对应迁移目录 `migrates/sqls/server/`。

当前 MySQL 组件及 `GetDb` 接口属于可替换示例，不是跨数据库抽象。PostgreSQL 等接入和无数据库源码裁剪步骤见 [README](../../README.md#选择新项目的数据库方案)。
