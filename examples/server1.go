package main

import (
	"fmt"
	"os"

	"context"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func errAssert(err error) {
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
	if req.Name == "鸠摩智" {
		(*rsp).Greeting = "yo"
	} else {
		(*rsp).Greeting = "who are you?"
	}
	return nil
}

// Nihao ...
func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	log.Info().Str("name", req.Name).Msg("Nihao")
	return nil
}

func main() {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	name := fmt.Sprintf("game.agent.%s", "xyz")
	svc, err := xing.NewService(name,
		mq,
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	errAssert(err)
	if err != nil {
		return
	}

	hello.RegisterGreeterHandler(svc, &Greeter{})

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	svc.Run()
}
