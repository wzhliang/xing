package main

import (
	"context"
	"fmt"
	"os"

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
	producer, err := xing.NewClient("game.controller", url,
		xing.SetIdentifier(&xing.NoneIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	target := fmt.Sprintf("game.agent.%s", os.Args[1])
	cli := hello.NewGreeterClient(target, producer)
	ret, err := cli.Hello(context.Background(), &hello.HelloRequest{
		Name: "鸠摩智",
	})
	_assert(err)
	_assertReturn("yo", ret.Greeting)
	if err != nil {
		fmt.Printf("returned: %v\n", ret)
	}
	_, err = cli.Nihao(context.Background(), &hello.HelloRequest{
		Name: "王语嫣",
	})
	_assert(err)
	producer.Close()
}
