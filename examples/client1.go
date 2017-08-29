package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	url := "amqp://guest:guest@localhost:5672/"
	producer, err := xing.NewClient("orchestration.controller", url,
		xing.SetIdentifier(&xing.NoneIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	name := fmt.Sprintf("host.agent.%s", os.Args[1])
	cli := hello.NewGreeterClient(name, producer)
	ret, err := cli.Hello(nil, &hello.HelloRequest{
		Name: "鸠摩智",
	})
	_assert(err)
	fmt.Printf("returned: %v\n", ret)
	_, err = cli.Nihao(nil, &hello.HelloRequest{
		Name: "王语嫣",
	})
	_assert(err)
}
