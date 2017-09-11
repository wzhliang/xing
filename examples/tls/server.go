package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"

	"context"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/examples/hello"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type Greeter struct{}

func (g *Greeter) Hello(ctx context.Context, req *hello.HelloRequest, rsp *hello.HelloResponse) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	(*rsp).Greeting = "yo"
	return nil
}

func (g *Greeter) Nihao(ctx context.Context, req *hello.HelloRequest, v *hello.Void) error {
	fmt.Printf(" [*] name: %s\n", req.Name)
	return nil
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

	svc, err := xing.NewService("host.server",
		"amqps://guest:guest@foo.wise2c.com:12000/",
		xing.SetSerializer(&xing.JSONSerializer{}),
		xing.SetTLSConfig(cfg),
	)
	_assert(err)

	forever := make(chan bool)

	hello.RegisterGreeterHandler(svc, &Greeter{})

	go func() {
		svc.Run()
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
