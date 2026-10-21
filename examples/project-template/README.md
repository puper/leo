# Leo 新项目骨架

基于 Go、Gin、Cobra、Viper、Zap 和 leo 组件容器的后端参考模板。默认无数据库启动，提供 JWT 鉴权、统一错误响应和健康检查。仓库保留可选的 GORM/MySQL 创建/查询示例与迁移，供需要持久化的新项目参考。没有登录、用户管理、告警或文件服务业务。

本目录是独立的 Go module（`server`），可整体复制为新项目。目录组织见 [架构](docs/design/architecture.md)，从零添加业务模块和注册 Leo 组件见 [开发与组件示例](docs/components.md)。

## 在 Leo 仓库中开发

从 Leo 根目录进入 `examples/project-template` 后执行本文命令。根 `go.work` 让模板使用当前 Leo 源码和依赖图，因此可以同时调试库和应用；无需修改模板 `go.mod` 为本地路径。库与模板是两个 module，根目录的 `go test ./...` 不包含模板，需单独运行 `make build test lint`。

复制到 Leo 仓库之外后，模板使用自身 `go.mod` 固定的 `github.com/puper/leo/v2 v2.0.0`，不再继承工作区；`GOWORK=off make build test lint` 可验证这种模式。库的未发布修改不会自动进入新项目，应在发布并验证后更新新项目的 Leo 版本。

## 环境与首次运行

- Go 1.25.5 或更高版本。
- 默认不需要数据库。只有启用 MySQL 示例时才需要 MySQL 8。
- Docker 可选；构建和单元测试不需要数据库或 Docker。

```bash
go mod download
make build
make test
make lint
cp config/config.example.toml config/config.toml
```

编辑 `config/config.toml`：

- `app.auth.jwtSecret`：填入独立生成的 URL-safe Base64 密钥，解码后至少 32 字节。可用 `openssl rand -base64 32 | tr '+/' '-_'` 在本地生成；不要提交或记录密钥。
- `app.nodeID`：0–1023 的整数；所有同时运行的服务进程必须使用不同值，避免 Snowflake ID 冲突。
- `app.env`：`dev` 或 `prod`；`app.timezone` 默认 UTC。
- `web.corsOrigins`：默认空列表，不启用跨域；需要跨域时显式填写前端 origin（如 `http://localhost:3000`），不接受通配符。

```bash
make run
# 或者
./server serve --config ./config/config.toml
curl --fail http://localhost:8888/readiness
```

默认配置只需填写独立 JWT 密钥即可启动，不会连接数据库、执行迁移或注册数据库示例接口。真实配置已被 Git 和 Docker 构建上下文忽略。无效配置或端口占用会以非零状态退出，配置修改后需要重启。

## API 与认证

| 方法 | 路径 | 用途与启用条件 |
|---|---|---|
| GET | `/readiness` | 默认检查服务可用；启用数据库后还检查主库，失败返回 503 |
| POST | `/api/example/create` | 仅 MySQL 示例启用时注册；JSON 为 `{"name":"示例"}` |
| POST | `/api/example/detail` | 仅 MySQL 示例启用时注册；JSON 为 `{"id":"返回的ID"}` |
| GET | `/swagger/index.html` | 仅 dev 环境提供；文档标注可选接口的启用条件 |

示例接口需要 `Authorization: Bearer <token>`；URL 参数、Cookie 和裸 token 均不接受。接入项目在验证真实用户身份后调用 `middleware.GenerateTokenWithUserID` 签发令牌。模板没有默认用户或绕过登录的 token 接口。认证信息可通过 Gin context 的 `userId`、`username` 读取。

响应格式为 `{"code":200000,"message":"ok","data":...}`。HTTP 状态码分别表达成功、参数错误、未授权、未找到和内部错误，JSON `code` 为对应 HTTP 状态码乘 1000。ID 在 JSON 中使用字符串，避免 JavaScript 整数精度丢失。

示例表是共享演示数据，不包含数据归属或租户权限模型；用于真实业务时补充相应授权规则。

## 选择新项目的数据库方案

先确定新项目是否需要持久化，不必照搬模板的 MySQL 技术选型。

| 新项目需求 | 参考方式 |
|---|---|
| 不使用数据库 | 保持配置中没有 `db` 段，直接开发无状态业务；可进一步裁剪数据库示例源码 |
| 使用 MySQL | 显式启用下述 MySQL 配置，参考现有 GORM 模型、Service 和迁移 |
| 使用 PostgreSQL 或其他数据库 | 参考业务分层与错误处理，在新项目中替换连接组件、配置及 SQL；模板未内置这些驱动 |

### 启用 MySQL 示例

1. 创建独立的 MySQL 8 数据库及具有建表、读写权限的用户。
2. 在 `config/config.toml` 中取消示例文件末尾**整个**数据库配置块的注释，包括 `[db.servers.server]` 和 `[db.servers.server.migrate]`。
3. 将 `master` 填为实际 DSN，格式为 `USER:PASSWORD@tcp(HOST:3306)/DATABASE?charset=utf8mb4&parseTime=True`。
4. 重启服务。连接和迁移成功后，才注册 `/api/example/create`、`/api/example/detail`。

省略 `db` 表示不启用数据库；配置了 `db` 却缺字段、连接失败或迁移失败都会阻止启动，不能用空 DSN 代表禁用，也不会自动切换到无数据库模式。

### 无数据库项目的源码裁剪

默认无数据库是运行行为，源码仍包含 MySQL 示例及其编译依赖。如果新项目确定不需要数据库，可以配套移除：

- `internal/modules/example/`、`internal/ifaces/db/`、`migrates/`，以及示例的单元/数据库集成测试。
- `internal/bootstrap/bootstrap.go` 中 MySQL/示例注册与数据库 readiness 检查。
- `internal/app/app.go` 的 `GetDb`，`internal/components/names.go` 的 `Db`、`Example`，以及 `internal/config/config.go` 的 `Db` 字段、导入和校验。
- 配置中的 MySQL 注释块和对应文档。示例没有其他调用方时再移除相关组件，不影响通用响应、鉴权、日志与 Web 测试。

随后执行 `go mod tidy`、`make swagger`、`make build`、`make test`、`make lint`。依赖以实际引用为准：leo 的容器和日志仍在使用，不要直接删除整个 leo 依赖，也不需要改成内存存储来伪装数据库行为。

### 在新项目中改接 PostgreSQL 等数据库

当前 `github.com/puper/leo/v2 v2.0.0` 的数据库组件内部固定调用 MySQL 驱动。**仅把 `driver = "mysql"` 改为 `"postgres"` 不会支持 PostgreSQL**，模板会拒绝这个配置。

新项目采用 PostgreSQL 时，建议只接入实际选定的驱动，不必为所有数据库建立兼容层：

1. 根据新项目的 Go/GORM 版本选择并固定 `gorm.io/driver/postgres`，用 `gorm.Open(postgres.Open(dsn), ...)` 建立连接，替换 MySQL 专用初始化。具体连接方式参见 [GORM 官方 PostgreSQL 文档](https://gorm.io/docs/connecting_to_the_database.html#PostgreSQL)。本仓库未新增该依赖，也未验证 PostgreSQL 运行。
2. 同步修改 `internal/config/config.go` 和配置示例，定义 PostgreSQL 连接参数、连接池及校验；DSN、TLS 和时区设置按实际部署要求配置，不复用 MySQL 参数。
3. 调整 `internal/ifaces/db/iface.go`、`internal/app/app.go`、示例 Service 和测试中的数据库类型。若直接提供 `*gorm.DB`，将 MySQL 组件的 `ReadModel`、`WriteModel` 调用替换为项目实际采用的 GORM 查询方式。
4. 在 `internal/bootstrap/bootstrap.go` 注册新的连接和迁移入口，让 readiness 检查实际依赖；连接或迁移失败必须阻止启动。
5. 将 `migrates/` 改为目标数据库的迁移，不能直接执行当前带有 `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` 的 MySQL SQL。重新检查主键生成、字段类型、索引、事务与回滚语义，并明确目标 [schema/search_path](https://www.postgresql.org/docs/current/ddl-schemas.html)。
6. 用隔离的目标数据库替换 MySQL 集成测试，覆盖迁移重放、真实读写、错误与取消路径，再整理依赖、生成文档并执行完整验证。

其他数据库采用相同的接入检查过程；数据库选型、迁移 SQL 和驱动支持都由新项目决定。

## MySQL 示例的数据库迁移

迁移文件位于 `migrates/sqls/<连接名>/<序号>_<说明>.up.sql`，由 Go embed 打包到二进制。当前连接名为 `server`，`001_create_example.up.sql` 创建示例表；`schema_migrations` 记录已执行版本。

- 修改表结构时新增迁移，不编辑已经执行的迁移。
- 模板只包含建表迁移，不提供自动删表或回滚入口。
- MySQL DDL 不保证事务回滚；上线前审查迁移，首次迁移由单个服务进程执行，避免多副本并发建表。

## 开发与验证

```bash
make build                 # Go modules 构建，无需 vendor
make test                  # 单元测试；未配置测试数据库时跳过 MySQL 集成测试
make lint                  # go vet，无需安装全局 lint 工具
make swagger               # 通过固定版本 swag 重新生成 API 文档
```

Swagger 生成结果在 `internal/swagger/docs/` 中随源码提交，首次构建无需生成。修改接口注释后执行 `make swagger` 并提交生成文件。生成工具使用固定版本 `swag v1.16.6`，不安装到全局 PATH；首次使用需要下载工具依赖。

要运行数据库集成测试，将 `TEST_MYSQL_DSN` 设置为**隔离测试数据库**的 DSN，然后执行：

```bash
go test -count=1 -v ./internal/modules/example/service -run TestMySQLIntegration
```

该测试会运行迁移并创建示例记录，覆盖迁移重复执行、真实创建/查询、参数校验、鉴权、取消请求和数据库失败；不会删除已有数据。不要指向生产数据库。

## Docker

此 Dockerfile 用于独立应用，构建上下文为本模板目录，依赖取自模板 `go.mod/go.sum` 的发布版本，不读取 Leo 根目录的工作区。对当前 Leo 源码的联合验证使用前述 workspace 命令；发布后再更新应用依赖并构建部署镜像。

```bash
make docker-build
make docker-run
```

镜像以非 root 用户运行，运行层使用精简 Debian、二进制和 CA 证书，不包含 Go 工具链；时区数据嵌入二进制。`make docker-run` 将本地 `config/config.toml` 只读挂载到容器，文件须允许容器 UID 65532 读取；镜像内没有预设密钥或数据库配置。

启用数据库时，容器中的数据库地址必须能从容器访问；`127.0.0.1` 指向容器自身。Docker Desktop 访问宿主机数据库可使用 `host.docker.internal`，同一 Docker 网络内使用数据库容器名称。

## 创建新项目

1. 将本模板源码复制到 Leo 仓库以外的独立目录；不要复制本地 `server` 二进制、`.ai/`、`config/config.toml`、日志，也不要复制 Leo 根目录的 `go.work`。建立新项目自己的 Git 仓库。
2. 将 `go.mod` 的 `module server`、Go 源码中的 `server/` 导入路径、Makefile 与 Dockerfile 的版本注入路径统一替换为新模块路径。
3. 按需要修改二进制名、CLI 描述、`main.go` 的 API 标题和版本。
4. 按上文选择无数据库、MySQL 或其他数据库方案，再决定保留、裁剪或替换示例；不要把模板的数据库、ID 策略和业务模型视为新项目的强制约定。
5. 生成新的 JWT 密钥，配置节点号；仅在需要数据库时配置对应连接与迁移，按真实登录流程接入认证。
6. 执行 `go mod tidy`、`make swagger`、`make build`、`make test`、`make lint`。

选择 Leo 发布版本时使用已验证的明确版本号，不把 `@latest` 写成自动构建步骤。本模板默认依赖 v2.0.0，Leo 的导入路径均包含 `/v2`；本仓库里的工作区构建验证的是当前源码，两者需分别验证。

目录职责见 [架构说明](docs/design/architecture.md)，协作约定见 [AGENTS.md](AGENTS.md)。
