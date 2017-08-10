package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func handler(msgType string, v interface{}) {
	m := v.(string)
	fmt.Printf("message [%s]: %s\n", msgType, m)
}

func main() {
	consumer, err := xing.NewEventHandler(
		"ingress.agent",
		"amqp://guest:guest@localhost:5672/",
		xing.SetInterets("confcenter.#"),
	)
	_assert(err)

	forever := make(chan bool)

	go func() {
		consumer.Loop(handler)
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
