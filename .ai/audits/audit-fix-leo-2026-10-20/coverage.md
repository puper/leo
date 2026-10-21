# 范围与覆盖

基线 `d602fea`。覆盖 18 个单元，当前共 81 个 Go 文件：49 个非生成生产/配置/示例文件、30 个测试文件、2 个生成协议文件。生产与测试按当前任务逐项检查；生成协议文件以接口追踪和编译验证为主。

第三方 vendor 仅审查直接相关契约和构建冲突，不宣称完成所有第三方源码/CVE 审计。已有根目录 Mach-O `demo` 是二进制产物，不作为源码审查目标；其对应 TCP 示例源码已检查并编译。没有前端平台，因此浏览器、设备、方向和输入方式不适用。

|单元|覆盖内容|验证证据|最终状态|
|---|---|---|---|
|engine|拓扑、缺失依赖、回滚、关闭及信号生命周期|引擎测试、race；信号注销静态配对|verified|
|db|迁移、SQL 资源所有权、主从构造|真实 GORM/SQL 故障夹具、MySQL5.7.44、race|verified|
|etcd|Builder 失败资源释放、借用关系|真实 SDK 客户端关闭状态、race|verified|
|grpc/client|连接构造与配置失败|SDK Shutdown 状态、race|verified|
|grpc/server|服务启动、RPC 和关闭期限|实际 TCP RPC、race|verified|
|influxdb|查询协议、精度、构造失败及关闭|HTTP 响应/连接状态、SDK write worker、race|verified|
|iris/web|当前 SDK 签名、Build、启动、HTTP 超时和端口释放|实际 HTTP/TCP、失败注入、race|verified|
|nats|Builder、协议连接和错误日志|本地 NATS 协议服务、合成凭据、race|verified|
|rabbitmq/subscription|channel/ack 缓存、背压、启动、取消和关闭|真实 Ack 接口夹具、AMQP 握手 TCP 故障、race|verified|
|restyclient|客户端配置及设置超时调用关系|实际 HTTP 请求和 hook、race|verified|
|storage/localfile|完整 Storage 操作、路径、链接、并发创建|Darwin/Linux 文件系统、race|verified|
|uniqid|租约、发号、修订、快照、并发、关闭|故障接口夹具、etcd3.6.7 双实例、race|verified|
|zaplog/log|自有 writer、Builder 失败与关闭|Unix 实际文件描述符、race|verified|
|pkg/mutexmanager|读写等待计数和锁管理|原有并发测试、全量 race，无需修改|verified|
|pkg/reconnect|连接代次、重试、并发操作及取消|故障/并发夹具、长退避 amd64 反事实、race|verified|
|pkg/timewheel|任务索引、派发、回调重入、关闭|真实时间队列及零缓冲测试、race|verified|
|构建与配置|go.mod/vendor 一致性、当前 SDK 调用、组件配置|完整 build/vet；Darwin/Linux 编译和测试|verified|
|文档、示例和测试|公开 API 文档、历史 QA 反证、TCP 示例、测试夹具|相关文档逐项复核；示例编译；新增链接检查|verified|

覆盖总数 18，终态 18，pending 0。Windows、真实 RabbitMQ broker、真实 NATS/InfluxDB 服务、生产部署和故障演练未执行，不继承夹具或另一平台的通过结论。
