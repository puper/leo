# 语言约定 (Language Conventions)

## 默认语言

本项目所有 AI 回复、代码注释、文档均使用**简体中文 (Simplified Chinese)**。

## 代码注释规范

- **关键逻辑**: 必须包含中文注释解释"为什么"而非"是什么"
- **公共 API**: 使用中文 DocString 说明用途和注意事项
- **复杂算法**: 添加步骤说明的中文注释

## 示例

```go
// New 创建 Db 实例
// 依赖配置中的 Servers 列表建立主从连接
func New(cfg *config.Config) (*Db, error) {
    // 使用随机数实现读负载均衡
    if len(me.slave) == 0 {
        return me.master
    }
    return me.slave[rander.Intn(len(me.slave))]
}
```

## 文档风格

- 标题使用中文
- 技术术语首次出现时附英文原词
- 保持简洁，避免冗余解释
