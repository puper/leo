# QA 记录：reconnectable 退役与 iris 启动检查窗口调整

## 背景

- 用户决策：`pkg/reconnectable` 组件已不再使用，允许直接删除。
- 风险点：`components/iris/web/builder.go` 使用固定 `50ms` 启动检查窗口，存在启动误判成功的风险。

## 变更设计

- 删除 `pkg/reconnectable` 目录下实现与测试文件，避免保留无主代码。
- 在 `components/iris/web/config.Config` 中新增 `StartCheckTimeout` 配置项。
- `Builder` 内部将启动检查窗口改为：
  - 若 `StartCheckTimeout > 0`，使用配置值。
  - 否则使用默认值 `300ms`（高于历史 `50ms`）。

## 任务清单

- [x] 文档同步：索引/架构中移除 `reconnectable` 活跃组件描述。
- [x] 代码执行：删除 `pkg/reconnectable`。
- [x] 代码执行：提升并参数化 `iris` 启动检查窗口。
- [ ] 回归验证：执行 `go test ./...`。

## 备注

- 当前环境缺少 `go` 命令，`go test ./...` 无法在本机执行，需在具备 Go 环境的 CI 或开发机复核。
