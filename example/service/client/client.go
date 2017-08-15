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
	producer, err := xing.NewProducer("confcenter.agent", url,
		xing.SetIdentifier(&xing.NodeIdentifier{}))
	_assert(err)
	_, _, err = producer.Call("confcenter.apiserver", "restart_pod", os.Args[1])
	_assert(err)
	// err = producer.Call("confcenter.apiserver", "foo", os.Args[1])
	// _assert(err)
	// err = producer.Call("confcenter.apiserver", "bar", os.Args[1])
	// _assert(err)
}
