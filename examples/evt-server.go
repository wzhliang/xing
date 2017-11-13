package main

import (
	"os"

	"context"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

var counter int

func _assert(err error) {
	if err != nil {
		log.Error().Err(err).Msg("...")
		panic(err)
	}
}

// Greeter ...
type Greeter struct{}

// Hello ...
func (g *Greeter) Hello(ctx context.Context, req *hello.HelloRequest, rsp *hello.HelloResponse) error {
	log.Info().Str("name", req.Name).Msg("Hello")
	rsp.Greeting = "Ciao"
	counter++
	log.Info().Int("counter", counter).Msg("Hello")
	return nil
}

// Nihao ...
func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	log.Info().Str("name", req.Name).Msg("Nihao")
	counter++
	log.Info().Int("counter", counter).Msg("Hello")
	return nil
}

func main() {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	consumer, err := xing.NewEventHandler(
		"ingress.agent.xyz", mq,
		xing.SetInterets("ingress.#"),
		xing.SetInterets("confcenter.api.#"),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	_assert(err)
	counter = 0

	hello.RegisterGreeterHandler(consumer, &Greeter{})

	log.Info().Msg(" [*] Waiting for messages. To exit press CTRL+C")
	consumer.Run()
}
