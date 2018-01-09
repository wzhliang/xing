package main

import (
	"os"

	"context"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

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
	if req.Name == "鸠摩智" {
		(*rsp).Greeting = "yo"
	} else if req.Name == "王语嫣" {
		(*rsp).Greeting = "美女好"
	} else {
		(*rsp).Greeting = "陛下好"
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
	svc, err := xing.NewService("host.server",
		mq,
		xing.SetSerializer(&xing.JSONSerializer{}),
		xing.SetBrokerTimeout(15000, 5),
	)
	_assert(err)

	hello.RegisterGreeterHandler(svc, &Greeter{})

	log.Info().Msg(" [*] Waiting for messages. To exit press CTRL+C")
	err = svc.Run()
	log.Printf("Run returned: %v", err)
}
