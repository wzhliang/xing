package main

import (
	"fmt"
	"os"

	"context"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

var counter int

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *hello.HelloRequest, rsp *hello.HelloResponse) error {
	fmt.Printf(" [Hello] name: %s\n", req.Name)
	rsp.Greeting = "Ciao"
	counter++
	fmt.Printf(" counter: %d", counter)
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [Nihao] name: %s\n", req.Name)
	counter++
	fmt.Printf(" counter: %d", counter)
	return nil
}

func main() {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	consumer, err := xing.NewEventHandler(
		"ingress.agent", mq,
		xing.SetIdentifier(&xing.RandomIdentifier{}),
		xing.SetInterets("ingress.#"),
		xing.SetInterets("confcenter.api.#"),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	_assert(err)
	counter = 0

	hello.RegisterGreeterHandler(consumer, &Greeter{})

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	consumer.Run()
}
