# gRPC 协议生成

在本目录执行：

```bash
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative any.proto
```

v2.0.0 的生成版本为 protoc 34.1、protoc-gen-go v1.34.2、protoc-gen-go-grpc 1.5.1。工具可通过已有安装或隔离工具目录提供，不需要修改全局 Go 环境。

`go_package` 为 `github.com/puper/leo/v2/components/grpc/protos`；protobuf package、消息字段编号和 gRPC 服务方法均保持不变，主版本迁移只改变 Go 导入路径。
