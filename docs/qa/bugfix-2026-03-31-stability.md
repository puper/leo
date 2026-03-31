# 稳定性缺陷修复记录（2026-03-31）

## 背景

本轮修复聚焦于组件生命周期、启动可观测性、数据反序列化以及文件路径安全校验等高风险问题。

## 问题清单与修复策略

- `pkg/reconnectable` 启动阻塞  
  现象：`Start()` 等待 `initCh`，但无发送路径，调用方会卡住。  
  修复：在主循环启动后发送一次初始化完成信号，避免死锁。

- `components/influxdb` 记录查询结果反序列化异常  
  现象：`QueryRecords` 使用 `json.Unmarshal(b, &reply)`，导致结果不能正确写回调用方对象。  
  修复：改为 `json.Unmarshal(b, reply)`，并补齐 `json.Marshal` 错误处理。

- `components/iris/web` 启停行为不完整  
  现象：服务启动错误在 goroutine 中被忽略；`ShutdownTimeout` 配置未生效。  
  修复：启动前先执行 `net.Listen` 以提前暴露端口错误；关闭时使用 `context.WithTimeout`。

- `engine` 依赖构建器缺失时静默跳过  
  现象：拓扑节点存在但缺失 builder 时，`Build()` 不报错，导致问题延后到运行期。  
  修复：`Build()` 遇到缺失或空 builder 立即返回错误。

- `components/storage/localfile` 路径归属校验不严谨  
  现象：路径归属判断使用错误基准，存在误判风险。  
  修复：统一基于绝对 `RootDir` 与路径分隔符判断归属，禁止越界路径。

- `components/db/migratefs` 迁移加载健壮性不足  
  现象：`sqls` 目录读取错误被吞；迁移 SQL 在闭包中存在循环变量捕获风险。  
  修复：仅在目录不存在时静默返回；其他错误透传。执行 SQL 前固定每次循环的 SQL 文本副本。

## 回归检查建议

- 启动包含 `reconnectable` 的组件，确认 `Start()` 立即返回并可正常关闭。
- 调用 `V1QueryApi.QueryRecords`，确认目标切片/结构体被正确填充。
- 配置一个已占用端口启动 `iris` 组件，确认构建阶段直接报错。
- 在 `engine` 中声明未注册依赖，确认 `Build()` 返回明确错误。
- 使用 `../` 路径访问 `localfile`，确认被拒绝。
- 准备多份迁移 SQL，确认 up/down 分别执行对应内容。
