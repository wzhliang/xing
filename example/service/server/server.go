package main

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/wzhliang/xing"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	svc, err := xing.NewService("confcenter.apiserver", "amqp://guest:guest@localhost:5672/")
	_assert(err)

	forever := make(chan bool)

	handler := func() xing.MessageHandler {
		return func(msgType string, v interface{}, ctx amqp.Delivery) {
			m := v.(string)
			fmt.Printf("message [%s]: %s\n", msgType, m)
			svc.Respond(ctx, "plain", "got it boss")
		}
	}

	go func() {
		svc.Loop(handler())
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
