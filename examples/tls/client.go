package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func _assert(err error) {
	if err != nil {
		log.Errorf("Client: %v", err)
	}
}

func main() {
	cfg := new(tls.Config)
	cfg.RootCAs = x509.NewCertPool()

	if ca, err := ioutil.ReadFile("testca/ca_certificate.pem"); err == nil {
		cfg.RootCAs.AppendCertsFromPEM(ca)
	}

	if cert, err := tls.LoadX509KeyPair("testca/client_certificate.pem", "testca/client_key.pem"); err == nil {
		cfg.Certificates = append(cfg.Certificates, cert)
	} else {
		log.Printf("Error: %v", err)
	}
	url := "amqps://guest:guest@foo.wise2c.com:12000/"
	producer, err := xing.NewClient("orchestration.controller", url,
		xing.SetIdentifier(&xing.NoneIdentifier{}),
		xing.SetSerializer(&xing.JSONSerializer{}),
		xing.SetTLSConfig(cfg),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5000*time.Millisecond)
	defer cancel()

	cli := hello.NewGreeterClient("host.server", producer)
	ret, err := cli.Hello(ctx, &hello.HelloRequest{
		Name: "鸠摩智",
	})
	_assert(err)
	if err == nil {
		fmt.Printf("returned: %s\n", ret.Greeting)
	}
	_, err = cli.Nihao(ctx, &hello.HelloRequest{
		Name: "王语嫣",
	})
	_assert(err)
	cli = hello.NewGreeterClient("host.server", producer)
	ret, err = cli.Hello(ctx, &hello.HelloRequest{
		Name: "鸠摩智",
	})
	_assert(err)
	if err == nil {
		fmt.Printf("returned: %s\n", ret.Greeting)
	}
	producer.Close()
}
