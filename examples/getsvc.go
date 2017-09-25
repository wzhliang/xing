package main

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/wzhliang/xing"
)

func err_assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	cr := xing.NewEtcdRegistrator()
	if cr == nil {
		log.Fatalln("Unable to get registrator")
	}

	forever := make(chan bool)
	go func() {
		for {
			svc, err := cr.GetService("host.agent", xing.NewInstanceSelector(os.Args[1]))
			if err == nil {
				log.Infof("service: %v", svc)
			} else {
				log.Errorf("Failed to get service: %v", err)
			}
			time.Sleep(10 * time.Second)
		}
	}()
	<-forever
}
