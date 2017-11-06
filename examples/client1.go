package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
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

func _assertReturn(req, resp string) {
	if req != resp {
		log.Errorf("Client: %s != %s", req, resp)
	}
}

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = url
	}
	producer, err := xing.NewClient("game.controller", mq,
		xing.SetIdentifier(&xing.NoneIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	if err != nil {
		log.Errorf("failed to create new client")
	}
	target := fmt.Sprintf("game.agent.%s", "xyz")
	cli := hello.NewGreeterClient(target, producer)

	n, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Printf("Wrong argument: %s", os.Args[2])
	}

	for i := 0; i < n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
		ret, err := cli.Hello(ctx, &hello.HelloRequest{
			Name: "鸠摩智",
		})
		_assert(err)
		cancel()
		if err == nil {
			_assertReturn("yo", ret.Greeting)
		}
		ctx, cancel = context.WithTimeout(context.Background(), 5000*time.Millisecond)
		_, err = cli.Nihao(ctx, &hello.HelloRequest{
			Name: "王语嫣",
		})
		_assert(err)
		cancel()

		time.Sleep(1 * time.Second)
	}
	producer.Close()
}
