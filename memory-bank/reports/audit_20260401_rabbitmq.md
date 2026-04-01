---
topic: "rabbitmq subscription component audit"
date: 2026-04-01
verdicts: [Logic, Security, Performance]
---

## Findings

### [Issue 1] Start() 可能永久阻塞 (Severity: CRITICAL)
- **Description**: 如果在连接建立前 context 被取消，`<-me.initCh` 会永久阻塞
- **Impact**: 无法优雅关闭订阅组件，导致 goroutine 泄漏
- **Proof**: component.go:101 `return <-me.initCh` 且 run() 可以通过 signalCh 提前返回而不写入 initCh
- **Reproduction**: 
  1. New() → Start()
  2. mainloop() 启动 run()
  3. amqp.Dial 阻塞期间调用 Close()
  4. ctx 被取消，run() 通过 signalCh 返回
  5. initCh 永不被写入，Start() 阻塞

### [Issue 2] Ack() 失败导致缓存移除，引发重复投递循环 (Severity: HIGH)
- **Description**: 当 Delivery.Ack() 因网络故障返回错误时，缓存被删除，但服务器可能已收到 ack
- **Impact**: 消息被重新投递并重复处理
- **Proof**: component.go:252-254
  ```go
  if err := me.Delivery.Ack(multiple); err != nil {
      me.cache.Remove(me.DeliveryTag)  // 错误！应该保留缓存
      return err
  }
  ```

### [Issue 3] Close() 在重连期间阻塞 (Severity: MEDIUM)
- **Description**: Close() 需等待完整的 reconnectDelay 才能退出，最长可达 1 分钟
- **Impact**: 无法快速关闭订阅
- **Proof**: component.go:145 `case <-time.After(reconnectDelay)`

### [Issue 4] 缺少 QoS/Prefetch 配置 (Severity: HIGH)
- **Description**: 没有设置 prefetch count，服务器可能推送大量消息导致内存溢出
- **Impact**: 高吞吐场景下内存急剧增长
- **Proof**: config.go 无 PrefetchCount 字段，component.go 无 QoS 调用

### [Issue 5] 缓存无限增长 (Severity: HIGH)
- **Description**: gcache size=0 永不驱逐，长时间运行会导致内存泄漏
- **Impact**: 每秒 1000 msg × 3600 秒 = 3,600,000 条目/小时
- **Proof**: component.go:25 `cache: gcache.New(0).Build()`

### [Issue 6] 消息通道满时静默丢弃 (Severity: MEDIUM)
- **Description**: msgCh 满时消息直接 Ack 丢弃，无日志或监控
- **Impact**: 消费速度跟不上时消息丢失，无法追踪
- **Proof**: component.go:221-228

## Unit Test Evidence

### Test 1: TestStartCloseRace
- **Target File**: `components/rabbitmq/subscription/component_test.go`
- **Logic**: 验证 Start() 在 context 取消后能正确返回

### Test 2: TestAckFailureHandling  
- **Target File**: `components/rabbitmq/subscription/component_test.go`
- **Logic**: 验证 Ack 失败时不移除缓存

### Test 3: TestMessageDropOnFullChannel
- **Target File**: `components/rabbitmq/subscription/component_test.go`
- **Logic**: 验证 msgCh 满时的背压行为
