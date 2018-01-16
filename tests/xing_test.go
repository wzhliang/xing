package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func Test_Xing_01(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go server(t, ctx)
	time.Sleep(10 * time.Second)
	client1(t)
	cancel()
}

func client1(t *testing.T) {
	url := "amqp://guest:guest@localhost:5672/"
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = url
	}
	producer, err := xing.NewClient("orchestration.controller", mq,
		xing.SetIdentifier(&xing.RandomIdentifier{}),
	)
	if err != nil {
		t.Error("failed to create new client")
		return
	}

	cli := hello.NewGreeterClient("host.server", producer)

	//	time.Sleep(ResultQueueTTL*time.Seconds + 10*time.Seconds)
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()
	res, err := cli.Hello(ctx, &hello.HelloRequest{
		Name: "王语嫣",
	})
	if err != nil {
		t.Error("failed")
	}
	if res.Greeting != "美女好" {
		t.Error("unexpected result")
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

func server(t *testing.T, ctx context.Context) {
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = "amqp://guest:guest@localhost:5672/"
	}
	svc, err := xing.NewService("host.server",
		mq,
		xing.SetBrokerTimeout(15000, 5),
	)
	if err != nil {
		fmt.Printf("unable to create server %v", err)
		return
	}

	hello.RegisterGreeterHandler(svc, &Greeter{})

	fmt.Printf(" [*] Waiting for messages\n")
	err = svc.RunWithContext(ctx)
	log.Printf("Run returned: %v", err)
}
