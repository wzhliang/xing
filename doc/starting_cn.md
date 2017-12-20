# 准备
1. 需要安装protobuf compiler - protoc 和一个针对本项目的compiler plugin
1. 从下面两个地方之一下载并安装protoc:
    * [github.com/google/protobuf/releases](https://github.com/google/protobuf/releases)
    * [gitee mirror (incomplete)](https://gitee.com/wisecloud/protobuf/attach_files)
1. 安装compiler plugin
    1. `go get github.com/google/protobuf`
    1. `go get github.com/wzhliang/protobuf`
    1. `cd $GOPATH/src/github.com/wzhliang/protobuf && make`


# protobuf
* RPC和event都是通过protobuf实现的。需要先定义protobuf文件。参见`examples/hello`
* 用`protoc --go_out=plugins=xing:. hello.proto`命令编译Go语言实现
    * 用你自己的proto文件替换`hello.proto`
    * **不要**改动自动生成的文件！
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
* `svc := NewEventHandler("domain.service.instance", SetInterests()...)`
    * 可以指定关心的信息来源
* `svc.Run()`
* `cli := NewClient()`
* `cli.Notify()`
* 参见 `examples/{evt-server, notify}.go`

## 自定义context
缺省情况下，所有handler的调用都会使用`context.Background()`作为上下文。如果需要指定上下文，需要用`RunWithContext()`这个函数
这样做有两个用途：
0. 使用`context.WithValue()`给handler传自定义的参数。
1. 使用`context.WithCancel()`控制server的生命周期。


# Topic
topic是控制信息流动的机制。在xing里面，topic被定义为

    domain.service.instance.x.x

也就是五段。最后两段是内部使用的，用户可以不关心。前三段的基本规则如下：

1. 同一个微服务，`domain.service`应该相同。
1. stream handler缺省会关心本服务的所有信息。但同时可以调用`SetInterests()`来接收
   其它的信息
1. event handerl缺省对任何topic都没有interest。

一个消息是“单播”，还是“广播”的方式发送给一个consumer(service, event handler, 和stream handler)是跟第一次声明该consumer时所指定的名字有关的。
如果名字是两段，那么接受的方式就是共享的，所有实例会以先到先得的方式接受请求。如果是三段，那么每个实例会各自收到一份请求的副本。


# 服务注册
服务注册和健康检查通过心跳行为实现。用户应该自行实现`HealthChecker()`并提供给服务。
同时用户应该制定服务注册中心类型，现在仅仅支持consul。在服务正确初始化之后，可以调用

    Register(address, port, tags, ttl)

来注册该服务。系统会规律的调用health checker并决定是否注册。

参见 `examples/register.go`


# Debug
* 缺省xing仅仅打印WARN, ERROR级别的日志，如果需要打开全日志，需要定义一个环境变量
* `export XING_TRACE_ON=1`
