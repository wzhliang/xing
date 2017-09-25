# About
Designed to be pronounced like "cross-ing", this is currently a narrow-minded RPC library that helps building a particular style of microservices. 

Essentially, it is RabbitMQ + protobuf with some pre-defined rules on message topics.


# Credit
This started from an opioninated usage of the amqp library, as such, it took reference from the following fine projects, especially `cony`:

* [cony](https://github.com/assembla/cony)
    * wrapper around ampq in declarative style
* [coworkers](https://github.com/tjmehta/coworkers)
    * A RabbitMQ Microservice Framework in Node.js
* [relay](https://github.com/armon/relay)
    * Golang framework for simple message passing using an AMQP broker


Later on, somebody suggested that `xing` should provide higher level functionalities like what `gRPC` does. Initially I was against the idea because 1) I don't really like code generators 2) I don't really like hiding the fact that a seemingly simple function call is actually a remote call. But I added the functionality anyway, again, taking references from the following fine projects, especially `go-micro`:

* [go-micro](https://github.com/micro/go-micro)
    * a feature rich, flexible microservice framework that has very good abstraction. the companion [protobuf compiler plugin](https://github.com/wzhliang/protobuf/) for `xing` was a fork from go-micro; the WIP sercice registration builds directly on top of the `registry` package.
* [go-kit](https://github.com/go-kit/kit)
    * A standard library for microservices.
* [rpcx](https://github.com/smallnest/rpcx)
    * A RPC service framework based on net/rpc like alibaba Dubbo and weibo Motan. One of best performance RPC frameworks.
