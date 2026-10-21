# 快速上手

返回 [文档索引](index.md)。准备新服务时直接参考 [应用骨架](../examples/project-template/README.md)；下面展示 Leo 引擎的最小用法。

## 最小示例

将以下代码保存为临时目录中的 `main.go`，在本仓库根目录运行 `go run -mod=readonly /实际临时目录/main.go`。示例只向标准输出写日志，不需要数据库或消息服务。

```go
package main

import (
	"errors"
	"fmt"
	"os"

	leolog "github.com/puper/leo/v2/components/zaplog/log"
	logconfig "github.com/puper/leo/v2/components/zaplog/log/config"
	"github.com/puper/leo/v2/engine"
	"github.com/spf13/viper"
)

func run() (err error) {
	app := engine.New(viper.New())
	defer func() { err = errors.Join(err, app.Close()) }()
	app.Register("log", leolog.Builder(&logconfig.Config{
		Logs: map[string]*logconfig.LogConfig{
			"default": {Level: "info", Format: "json"},
		},
	}))
	if err = app.Build(); err != nil {
		return err
	}
	app.Get("log").(*leolog.Log).Get().Info("leo ready")
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

这里提前注册延迟清理，使 `Build()` 中途失败时也能释放已经构建的组件。常驻服务在构建成功后调用 `app.Wait()`，收到退出信号后由引擎关闭组件并返回关闭错误。

## 配置与依赖

`engine.Config` 是 `viper.Viper` 的别名。引擎保存并通过 `GetConfig()` 返回配置对象，不自动读取文件、校验配置或把配置传给 Builder。这些工作由应用启动代码完成；完整例子见模板的 [配置加载](../examples/project-template/internal/config/config.go) 和 [Bootstrap](../examples/project-template/internal/bootstrap/bootstrap.go)。

各 Builder 接收对应的配置结构，多数位于 `components/<组件>/config`。`time.Duration` 字段在 Go 中使用 `5 * time.Second` 等有单位的值；`db.ConnMaxLifeTime` 是整数秒，模板 Gin 的超时配置也是整数秒。

调用 `Register(name, builder, dependencies...)` 时，依赖名称必须与已注册组件一致。例如 `Register("service", builder, "db")` 表示先构建数据库，关闭时先关闭业务组件。Builder 或配置回调可用 `Get("db")` 取得已声明的依赖。

## 下一步

- 阅读 [生命周期契约](architecture.md)，了解 `Get` 的 panic、构建失败和关闭边界。
- 按 [组件指南](components.md) 选择实际需要的组件。
- 使用 [应用骨架的组件集成示例](../examples/project-template/docs/components.md) 为新服务增加功能。
