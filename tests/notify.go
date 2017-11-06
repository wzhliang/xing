package main

import (
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
	producer, err := xing.NewClient("ingress.controller", "amqp://guest:guest@localhost:5672/",
		xing.SetIdentifier(&xing.NodeIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	_assert(err)
	for {

		err = producer.Notify("ingress.agent", "Greeter::Nihao", &hello.HelloRequest{Name: "Jack"})
		_assert(err)
		time.Sleep(10 * time.Second)
	}
}