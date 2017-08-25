package main

import (
	"os"

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
		xing.SetIdentifier(&xing.NodeIdentifier{}))
	_assert(err)
	err = producer.NotifyAll("new_ingress_rule", os.Args[1])
	_assert(err)
	err = producer.NotifySome("ingress.agent", "new_ingress_rule", os.Args[1]+"XXX")
	_assert(err)
	producer.Close()
}
