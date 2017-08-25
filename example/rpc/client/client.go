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
	url := "amqp://guest:guest@localhost:5672/"
	producer, err := xing.NewClient("orchestration.controller", url,
		xing.SetIdentifier(&xing.NodeIdentifier{}))
	_assert(err)
	err = producer.Call("host.agent", "restart_pod", os.Args[1])
	_assert(err)
}
