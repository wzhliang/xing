package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

var success = 0

func _assert(err error) {
	if err != nil {
		log.Error().Msgf("Client: %v", err)
	}
}

func _assertReturn(req, resp string) {
	if req != resp {
		log.Error().Msgf("Client: %s != %s", req, resp)
	}
	success++
}

func call(cli hello.GreeterClient, req, resp string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	ret, err := cli.Hello(ctx, &hello.HelloRequest{
		Name: req,
	})
	_assert(err)
	if err == nil {
		_assertReturn(resp, ret.Greeting)
	}
	cancel()
}

func main() {
	numCPUs := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPUs + 1)
	url := "amqp://guest:guest@localhost:5672/"
	mq := os.Getenv("RABBITMQ")
	if mq == "" {
		mq = url
	}
	producer, err := xing.NewClient("orchestration.controller", mq,
		xing.SetIdentifier(&xing.RandomIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
	)
	if err != nil {
		log.Error().Msg("failed to create new client")
		return
	}

	cli := hello.NewGreeterClient("host.server", producer)
	n, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Error().Str("#", os.Args[1]).Msg("Wrong argument")
	}

	var wg sync.WaitGroup
	log.Info().Int("n", n).Msg("looping...")
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			call(cli, "鸠摩智", "yo")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			call(cli, "王语嫣", "美女好")
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			call(cli, "段誉", "陛下好")
		}()
	}
	fmt.Printf("\n\n\n\nWaiting for goroutines to stop...\n\n\n\n")
	wg.Wait()
	fmt.Printf("success=%d\n", success)
	if success != n*3 {
		fmt.Printf("test failed")
	}
	producer.Close()
}
