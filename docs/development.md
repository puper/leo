# 开发与验证

返回 [文档索引](index.md)。使用 Go 1.25.5 或更高版本进行库和模板联合开发，依赖由 Go Modules 获取，不需要生成 vendor。

## 两个模块

根模块是 `github.com/puper/leo/v2`，`examples/project-template` 是应用骨架。根 `go.work` 使用本地 Leo，并将模板的 `github.com/puper/leo/v2 v2.0.0` 依赖图指向根目录，使尚未发布的新版本也可联合构建。模板的 Gin/JWT/Swagger 依赖保留在自身 `go.mod` 中。

更新模板的 Leo 版本时，同步调整 `go.work` 中版本限定的 replace，并分别验证工作区和独立模块模式。

v1 调用方迁移步骤、公开接口变化与发布验收见 [v2 迁移说明](v2-migration.md)。既有 v1 标签不移动，v2 不提供旧导入路径的兼容层。

`go test ./...` 不会跨越嵌套 module，必须分别验证：

```bash
# 在仓库根目录
go test -mod=readonly ./...
go test -race -mod=readonly ./...
go vet -mod=readonly ./...
go build -mod=readonly ./...

# 应用骨架
cd examples/project-template
make build test lint
go test -race -mod=readonly ./...
```

需要确认发布给外部调用方的库依赖没有被工作区掩盖时，在根目录执行 `GOWORK=off go test -mod=readonly ./...`。模板复制到独立目录后也应执行自己的构建、测试和静态检查。

`-mod=readonly` 禁止构建时自动修改依赖声明。不要把 `go mod tidy` 或全量升级当作每次修改的常规步骤；需要调整依赖时，明确版本与调用方影响并检查 `go.mod`、`go.sum` 差异。Iris 与 golog 存在版本耦合，更新必须包含 Iris 包的编译验证。

## 测试条件

多数测试使用 mock、临时目录或本机回环服务。没有额外环境变量时，下面的真实服务测试会跳过：

| 模块 | 环境变量 | 实际场景 |
|---|---|---|
| Leo `components/db` | `LEO_TEST_MYSQL_DSN` | 部分连接初始化失败后的连接池释放 |
| Leo `components/uniqid` | `LEO_TEST_ETCD_ENDPOINT` | 真实租约分配、续租、释放与节点交接 |
| 模板 example service | `TEST_MYSQL_DSN` | 迁移和业务创建/查询、请求失败 |

仅把这些变量指向隔离的测试服务；模板的 MySQL 测试会执行迁移并写入记录。设置好对应变量后，可运行：

```bash
go test -count=1 -v ./components/db -run TestPartialInitializationClosesOpenedPools
go test -count=1 -v ./components/uniqid -run TestEtcdLeaseAllocationRenewalAndRelease
cd examples/project-template
go test -count=1 -v ./internal/modules/example/service -run TestMySQLIntegration
```

普通测试通过不等于这些集成场景通过。日志文件描述符检查仅在 darwin/linux 构建，其他操作系统需单独验证。

## 协作约定

修改前阅读实际接口、调用方、配置和测试，并更新 `.ai/plans/<任务名>.md`。仅修改当前任务所需内容，Go 文件使用 gofmt；新增行为覆盖正常、边界和错误路径。文档与注释同步更新，不将默认值、吞错或成功响应用作故障处理。

`.ai/` 被忽略后，新计划不进入 Git；历史已跟踪文件的删除仍需正常提交，忽略规则不会替代删除提交。真实配置、密钥和一次性验证产物不进入仓库。

gRPC 协议源码为 [any.proto](../components/grpc/protos/any.proto)，生成命令见 [protos README](../components/grpc/protos/README.md)。模板 Swagger 已提交生成结果，修改模板 API 注释后执行 `make swagger`；工具固定版本，通过 `go run` 使用，不需要全局安装。
