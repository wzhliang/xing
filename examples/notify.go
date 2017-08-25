package main

import (
	hello "./hello"
	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	producer, err := xing.NewClient("ingress.controller", "amqp://guest:guest@localhost:5672/",
		xing.SetIdentifier(&xing.NodeIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	_assert(err)
	err = producer.Notify("ingress.agent", "Nihao", &hello.HelloRequest{Name: "Jack"})
	_assert(err)
	err = producer.Notify("confcenter.nogo", "Nihao", &hello.HelloRequest{Name: "XXXX"})
	_assert(err)
	err = producer.Notify("confcenter.api", "Nihao", &hello.HelloRequest{Name: "Tom"})
	_assert(err)
	producer.Close()
}
