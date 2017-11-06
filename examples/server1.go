package main

import (
	"fmt"
	"os"

	"context"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func err_assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *hello.HelloRequest, rsp *hello.HelloResponse) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	if req.Name == "鸠摩智" {
		(*rsp).Greeting = "yo"
	} else {
		(*rsp).Greeting = "who are you?"
	}
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	return nil
}

func main() {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	name := fmt.Sprintf("game.agent.%s", "xyz")
	svc, err := xing.NewService(name,
		mq,
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	err_assert(err)
	if err != nil {
		return
	}

	hello.RegisterGreeterHandler(svc, &Greeter{})

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	svc.Run()
}
