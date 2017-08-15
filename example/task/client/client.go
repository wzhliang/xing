package main

import (
	"fmt"
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
	id1, err := producer.RunTask("host.agent.dev-1", "restart_pod", os.Args[1]+"-1")
	_assert(err)
	id2, err := producer.RunTask("host.agent.dev-2", "restart_pod", os.Args[1]+"-2")
	_assert(err)
	id3, err := producer.RunTask("host.agent.dev-3", "restart_pod", os.Args[1]+"-3")
	_assert(err)
	fmt.Printf("Waiting for result...\n")
	for _, t := range []string{id3, id2, id1} {
		typ, result, err := producer.WaitForTask(t)
		_assert(err)
		fmt.Printf("  result: %s\n", result)
	}
}
