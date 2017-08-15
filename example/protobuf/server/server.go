package main

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/wzhliang/xing"
	"github.com/wzhliang/xing/example/protobuf/user"
)

func _assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

type MyUser struct {
}

func (u *MyUser) ContentType() string {
	return "data/protobuf"
}

func (u *MyUser) Marshal(event string, data interface{}) ([]byte, error) {
	usr := data.(user.User)
	return proto.Marshal(&usr)
}

func (u *MyUser) Unmarshal(event string, data []byte) (interface{}, error) {
	usr := user.User{}
	err := proto.Unmarshal(data, &usr)
	log.Printf("usr: %v\n", usr)
	if err != nil {
		return nil, err
	}
	return usr, nil
}

func (u *MyUser) DefaultValue() interface{} {
	return ""
}

func main() {
	svc, err := xing.NewService("host.agent", "amqp://guest:guest@localhost:5672/",
		xing.SetSerializer(&MyUser{}))
	_assert(err)

	handler := func() xing.MessageHandler {
		return func(msgType string, v interface{}, ctx amqp.Delivery) {
			m := v.(user.User)
			fmt.Printf("message [%s] \n", msgType)
			fmt.Printf("\tid: %d\n", m.Id)
			fmt.Printf("\tname: %s\n", m.Name)
			fmt.Printf("\temail: %s\n", m.Email)
			fmt.Printf("\tcountry: %s\n", m.Country)
			m.Id++
			svc.Respond(ctx, "resp", m)
		}
	}

	forever := make(chan bool)

	go func() {
		svc.Loop(handler())
	}()
	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}
