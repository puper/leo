# 反证与边界

以下六项不作为新增确认缺陷/修法；没有新证据不重开。

|ID / 指纹|候选|反证或范围依据|重新评估条件|
|---|---|---|---|
|R01 / db/pool-config/shadowing|局部 stdDb 变量遮蔽使连接配置无效|每个取得的实际 SQL pool 都立即应用配置；真实 MySQL 路径也覆盖了连接池行为|出现具体配置未生效的复现|
|R02 / db/read/global-rand|包级 rand.Intn 本身不支持并发|Go 包级随机函数支持并发；现有并发读 race 通过，未新增全局锁|实际 race 栈指向非包级随机源|
|R03 / engine/close/first-error|只返回首错等于后续组件未关闭|实现遍历并关闭全部组件后才返回首错，原回归覆盖该行为|出现后续组件未调用 Close 的证据|
|R04 / uniqid/get/ignored-error|Get 的错误仍被忽略导致 nil panic|当前基线已经检查 Get 错误；旧文档 Skip 测试描述过时，现改为实际错误夹具|新的调用路径绕过错误检查|
|R05 / resty/clone/isolated-timeout|浅 Clone 可以在保留配置时得到独立 timeout|当前 SDK Clone 共享 http.Client，仍会改动原 timeout；因此拒绝此修法|SDK 提供可验证的独立 HTTP 客户端替换/复制契约|
|R06 / scope/downstream-migrator|在 presence-monitor 另写迁移器或补丁 vendor|用户已明确改为直接修复 Leo；准备中的下游测试改动已撤回|用户明确要求发布/下游升级或替代实现|

不自动扩展的接口边界：FileInfo.Path 为本地路径而逻辑名称按根目录解释；同步 Close 要求回调能结束；直接 NewLog 仍是长期 logger 工厂。未出现当前调用方违反这些边界的可达故障，不为此新增路径转换、回调适配器或额外公开接口。
