package main

import (
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
	if req.Name == "鸠摩智" {
		(*rsp).Greeting = "yo"
	} else if req.Name == "王语嫣" {
		(*rsp).Greeting = "美女好"
	} else {
		(*rsp).Greeting = "陛下好"
	}
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	return nil
}

func main() {
	svc, err := xing.NewService("host.server",
		"amqp://guest:guest@localhost:5672/",
		xing.SetSerializer(&xing.JSONSerializer{}),
		xing.SetBrokerTimeout(5, 2),
	)
	_assert(err)

	hello.RegisterGreeterHandler(svc, &Greeter{})

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	err = svc.Run()
	log.Printf("Run returned: %v", err)
}
