package main

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func _assert(err error) {
	if err != nil {
		log.Errorf("Client: %v", err)
	}
}

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	producer, err := xing.NewClient("orchestration.controller", url,
		xing.SetIdentifier(&xing.NoneIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()

	cli := hello.NewGreeterClient("host.server", producer)
	ret, err := cli.Hello(ctx, &hello.HelloRequest{
		Name: "鸠摩智",
	})
	_assert(err)
	if err == nil {
		fmt.Printf("returned: %s\n", ret.Greeting)
	}
	_, err = cli.Nihao(ctx, &hello.HelloRequest{
		Name: "王语嫣",
	})
	_assert(err)
	cli = hello.NewGreeterClient("host.server", producer)
	ret, err = cli.Hello(ctx, &hello.HelloRequest{
		Name: "鸠摩智",
	})
	_assert(err)
	if err == nil {
		fmt.Printf("returned: %s\n", ret.Greeting)
	}
	producer.Close()
}
