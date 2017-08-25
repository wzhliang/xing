# 准备
0. `go get github.com/google/protobuf`
0. `go get github.com/wzhliang/protobuf`
0. `cd $GOPATH/src/github.com/wzhliang/protobuf && make`

# protobuf
* RPC和event都是通过protobuf实现的。需要先定义protobuf文件。参见`examples/hello`
* 用`protoc --go_out=plugins=xing:. hello.proto`命令编译Go语言实现
* 编写protocol真正的实现
* 如果需要无返回的RPC，可以使用特定的`Void`返回message。
    * `.proto` 文件里面需要有定义改message

# Use cases
## Load Balanced RPC
* `svc := NewService("domain.service", ...)`
* `svc.Run()`
* `cli := NewXXXClient()`
* `cli.RPCMethod()`
* 参见 `examples/{client, server}.go`
## Singleton RPC
* `svc := NewService("domain.service.instace", ...)`
* `svc.Run()`
* `cli := NewXXXClient()`
* `cli.RPCMethod()`
* 参见 `examples/{client1, server1}.go`
## Event Handler
* `svc := NewEventHandler("domain.service.instance", SetIntereste()...)`
    * 可以指定关心的信息来源
* `svc.Run()`
* `cli := NewClient()`
* `cli.Notify()`
* 参见 `examples/{evt-server, notify}.go`
