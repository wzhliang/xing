package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

const (
	resultTTL = time.Duration(xing.ResultQueueTTL) * time.Millisecond
	queueTTL  = time.Duration(xing.QueueTTL) * time.Millisecond
)

var wg sync.WaitGroup
var cancel context.CancelFunc

func Test_Xing_01(t *testing.T) {
	t.Log("TESTING SIMPLE HAPPY CASE...")
	time.Sleep(2 * time.Second)
	client1(t)
}

func Test_Xing_NoCall(t *testing.T) {
	t.Log("TESTING NO CALL + LONG WAIT + CALL...")
	time.Sleep(2 * time.Second)
	clientNoCall(t, resultTTL)
}

func Test_Xing_NoCallServer(t *testing.T) {
	t.Log("TESTING NO CALL + LOOOONG WAIT + CALL...")
	time.Sleep(2 * time.Second)
	clientNoCall(t, queueTTL)
}

func Test_Xing_OneCall(t *testing.T) {
	t.Log("TESTING ONE CALL + LONG WAIT + CALL...")
	time.Sleep(2 * time.Second)
	clientOneCall(t, resultTTL)
}

func Test_Xing_OneCallServer(t *testing.T) {
	t.Log("TESTING ONE CALL + LOOOONG WAIT + CALL...")
	time.Sleep(2 * time.Second)
	clientOneCall(t, queueTTL)
}

func Test_ShutDown(t *testing.T) {
	cancel()
	wg.Wait()
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

func clientNoCall(t *testing.T, sleep time.Duration) {
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

	time.Sleep(sleep + 10*time.Second)

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

func clientOneCall(t *testing.T, sleep time.Duration) {
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

	t.Log("sleeping.......\n")
	time.Sleep(sleep + 10*time.Second)

	ctx, cancel = context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()
	res, err = cli.Hello(ctx, &hello.HelloRequest{
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

func server(ctx context.Context, t *testing.T) {
	defer wg.Done()
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

	err = svc.RunWithContext(ctx)
}

func init() {
	wg = sync.WaitGroup{}
	var ctx context.Context
	ctx, cancel = context.WithCancel(context.Background())
	wg.Add(1)
	go server(ctx, nil)
}
