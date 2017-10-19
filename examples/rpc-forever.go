package main

import (
	"context"
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

	cli := hello.NewGreeterClient("host.server", producer)
	for {
		cli = hello.NewGreeterClient("host.server", producer)
		ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		_, err = cli.Hello(ctx, &hello.HelloRequest{
			Name: "鸠摩智",
		})
		_assert(err)
		cancel()
		time.Sleep(1 * time.Second)
	}
}
