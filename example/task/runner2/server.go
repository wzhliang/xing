package main

import (
	"fmt"
	"time"

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
	svc, err := xing.NewTaskRunner("host.agent.dev-2", "amqp://guest:guest@localhost:5672/")
	_assert(err)

	handler := func() xing.MessageHandler {
		return func(msgType string, v interface{}, ctx amqp.Delivery) {
			m := v.(string)
			fmt.Printf("message [%s]: %s\n", msgType, m)
			time.Sleep(2 * time.Second)
			svc.Respond(ctx, "plain", "job done")
		}
	}

	forever := make(chan bool)

	go func() {
		svc.Loop(handler())
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
