package main

import (
	"fmt"

	"golang.org/x/net/context"

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
	fmt.Printf(" [Hello] name: %s\n", req.Name)
	rsp.Greeting = "Ciao"
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [Nihao] name: %s\n", req.Name)
	return nil
}

func main() {
	consumer, err := xing.NewEventHandler(
		"ingress.agent",
		"amqp://guest:guest@localhost:5672/",
		xing.SetIdentifier(&xing.RandomIdentifier{}),
		xing.SetInterets("confcenter.api.#"),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	_assert(err)

	hello.RegisterGreeterHandler(consumer, &Greeter{})

	forever := make(chan bool)

	go func() {
		consumer.Run()
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
