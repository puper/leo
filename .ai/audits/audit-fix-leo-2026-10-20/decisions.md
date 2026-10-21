# 决策与兼容性

- 已解决：用户明确选择保留 `Generate() ID`，失去租约时 panic。实现消息为 `uniqid: no valid lease`；底层 `Node` 私有化，防止绕过门控。调用方若直接访问原嵌入字段，需要改用 Generate。
- 已采用：`restyclient.SetTimeout` 与其他 SetX 一致地原地修改，保留完整配置。仓库内及 presence-monitor 自有代码没有依赖“返回新空客户端”的调用方；外部需要独立客户端时须显式独立创建。
- 已采用：Iris `StartCheckTimeout` 从“等待后假定成功”落实为等待 Host 注册的期限；构建错误同步返回，关闭拥有的 listener。历史 QA 已把误判成功列为风险。
- 已采用：仅将 golog 从 0.1.15 调整为 0.1.12 以匹配现有 Iris；没有升级其他依赖，没有手工修改第三方实现。
- 范围边界：本轮不提交、推送、打标签、发布，也不自动升级 presence-monitor 的固定依赖或 vendor。

待用户决定 0，授权阻塞 0。真实服务夹具仅使用本轮创建的临时数据库/键空间；不保存连接秘密到本计划或报告。
