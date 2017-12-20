package main

import (
	"os"
	"time"

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

// Key ...
type Key string

var key Key = "transport"

// Hello ...
func (g *Greeter) Hello(ctx context.Context, req *hello.HelloRequest, rsp *hello.HelloResponse) error {
	log.Info().Str("name", req.Name).Msg("Hello")
	rsp.Greeting = "Ciao"
	log.Info().Str("context", ctx.Value(key).(string)).Msg("XXX")
	counter++
	log.Info().Int("counter", counter).Msg("Hello")
	return nil
}

// Nihao ...
func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	log.Info().Str("name", req.Name).Msg("Nihao")
	log.Info().Str("context", ctx.Value(key).(string)).Msg("XXX")
	counter++
	log.Info().Int("counter", counter).Msg("Hello")
	return nil
}

func main() {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	consumer, err := xing.NewStreamHandler(
		"logging.controller", mq,
	)
	_assert(err)
	counter = 0

	hello.RegisterGreeterHandler(consumer, &Greeter{})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Second)
		cancel()
	}()
	log.Info().Msg(" [*] Should stop in 100 seconds")
	ctx = context.WithValue(ctx, key, "RMQ")
	ret := consumer.RunWithContext(ctx)
	log.Info().Err(ret).Msg("Run() returned")
}
