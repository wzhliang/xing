# 准备
0. 需要安装protobuf compiler - protoc 和一个针对本项目的compiler plugin
0. 下载安装protoc:
    * [github.com/google/protobuf/releases](https://github.com/google/protobuf/releases)
    * [gitee mirror (incomplete)](https://gitee.com/wisecloud/protobuf/attach_files)
0. `go get github.com/google/protobuf`
    * 为编译protobuf做准备
0. `go get github.com/wzhliang/protobuf`
0. `cd $GOPATH/src/github.com/wzhliang/protobuf && make`


# protobuf
* RPC和event都是通过protobuf实现的。需要先定义protobuf文件。参见`examples/hello`
* 用`protoc --go_out=plugins=xing:. hello.proto`命令编译Go语言实现
    * 用你自己的proto文件替换`hello.proto`
    * **不要**改动自动生成的文件！
* 编写protocol真正的实现
* 如果需要无返回的RPC，可以使用特定的`Void`返回message。
    * `.proto` 文件里面需要有定义改message


# Topic
topic是控制信息流动的机制。在xing里面，topic被定义为

    domain.service.instance.x.x

也就是五段。最后两段是内部使用的，用户可以不关心。基本规则如小儿：

0. 同一个微服务，`domain.service`应该相同。
0. event handler缺省会关心本domain的所有信息。但同时可以调用`SetInterests()`来接收
   其它的信息


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
* `svc := NewEventHandler("domain.service.instance", SetInterests()...)`
    * 可以指定关心的信息来源
* `svc.Run()`
* `cli := NewClient()`
* `cli.Notify()`
* 参见 `examples/{evt-server, notify}.go`


# 服务注册
服务注册和健康检查通过心跳行为实现。用户应该自行实现`HealthChecker()`并提供给服务。
同时用户应该制定服务注册中心类型，现在仅仅支持consul。在服务正确初始化之后，可以调用

    Register(address, port, tags, ttl)

来注册该服务。系统会规律的调用health checker并决定是否注册。

参见 `examples/register.go`
