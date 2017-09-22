package main

import (
	"fmt"
	"os"
	"time"

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
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	return nil
}

func main() {
	name := fmt.Sprintf("host.agent.%s", os.Args[1])
	svc, err := xing.NewService(name,
		"amqp://guest:guest@localhost:5672/",
		xing.SetSerializer(&xing.JSONSerializer{}),
		xing.SetHealthChecker(&xing.RandomHealthChecker{}),
		xing.SetRegistrator(xing.NewConsulRegistrator()),
	)
	err_assert(err)

	forever := make(chan bool)

	hello.RegisterGreeterHandler(svc, &Greeter{})

	svc.Register("1.2.3.4", 9000, map[string]string{"author": "wzhliang"}, 10*time.Second)

	go func() {
		svc.Run()
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
