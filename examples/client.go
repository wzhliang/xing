package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

var success = 0
var m sync.Mutex

func _assert(err error) {
	if err != nil {
		log.Error().Msgf("Client: %v", err)
	}
}

func _assertReturn(req, resp string) {
	if req != resp {
		log.Error().Msgf("Client: %s != %s", req, resp)
	}
	success++
}

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = url
	}
	producer, err := xing.NewClient("orchestration.controller", mq,
		xing.SetIdentifier(&xing.RandomIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	if err != nil {
		log.Error().Msg("failed to create new client")
		return
	}

	cli := hello.NewGreeterClient("host.server", producer)
	n, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Error().Str("#", os.Args[1]).Msg("Wrong argument")
	}

	for i := 0; i < n; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
		ret, err := cli.Hello(ctx, &hello.HelloRequest{
			Name: "鸠摩智",
		})
		_assert(err)
		if err == nil {
			_assertReturn("yo", ret.Greeting)
		}
		cancel()

		ctx, cancel = context.WithTimeout(context.Background(), 5000*time.Millisecond)
		ret, err = cli.Hello(ctx, &hello.HelloRequest{
			Name: "王语嫣",
		})
		_assert(err)
		if err == nil {
			_assertReturn("美女好", ret.Greeting)
		}
		cancel()

		ctx, cancel = context.WithTimeout(context.Background(), 5000*time.Millisecond)
		ret, err = cli.Hello(ctx, &hello.HelloRequest{
			Name: "段誉",
		})
		_assert(err)
		if err == nil {
			_assertReturn("陛下好", ret.Greeting)
		}
		cancel()
		time.Sleep(1 * time.Millisecond)
	}
	fmt.Printf("success=%d\n", success)
	if success != n*3 {
		fmt.Printf("test failed")
	}

	producer.Close()
}
