package main

import (
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func _assert(err error) {
	if err != nil {
		log.Error().Err(err).Msg("Client")
	}
}

func main() {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	producer, err := xing.NewClient("logging.agent", mq,
		xing.SetIdentifier(&xing.RandomIdentifier{}),
	)
	_assert(err)
	if err != nil {
		return
	}
	n, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Error().Str("#", os.Args[1]).Msg("Wrong argument")
	}
	for i := 0; i < n; i++ {
		err = producer.Notify("logging.controller", "Greeter::Nihao", &hello.HelloRequest{Name: "Nima"})
		_assert(err)
		time.Sleep(1 * time.Second)
	}
	producer.Close()
}
