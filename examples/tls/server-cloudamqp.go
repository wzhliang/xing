package main

import (
	"crypto/tls"
	"fmt"

	"context"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *hello.HelloRequest, rsp *hello.HelloResponse) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	(*rsp).Greeting = "yo"
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	return nil
}

func main() {
	cfg := new(tls.Config)
	svc, err := xing.NewService("host.server",
		"amqps://rxdpdbmw:UCPohyxi24dOymvJtc8m2bvkucyUdRH7@sidewinder.rmq.cloudamqp.com/rxdpdbmw",
		xing.SetSerializer(&xing.JSONSerializer{}),
		xing.SetTLSConfig(cfg),
	)
	_assert(err)

	forever := make(chan bool)

	hello.RegisterGreeterHandler(svc, &Greeter{})

	go func() {
		svc.Run()
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
