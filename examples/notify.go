package main

import (
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
		xing.SetIdentifier(&xing.RandomIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	_assert(err)
	err = producer.Notify("ingress.agent", "Greeter::Nihao", &hello.HelloRequest{Name: "Jack"})
	_assert(err)
	err = producer.Notify("confcenter.nogo", "Greeter::Nihao", &hello.HelloRequest{Name: "XXXX"})
	_assert(err)
	err = producer.Notify("confcenter.api", "Greeter::Nihao", &hello.HelloRequest{Name: "Tom"})
	_assert(err)
	producer.Close()
}
