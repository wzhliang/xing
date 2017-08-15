package main

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"
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
	log.Printf("usr: %v\n", usr)
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
	url := "amqp://guest:guest@localhost:5672/"
	producer, err := xing.NewProducer("orchestration.controller", url,
		xing.SetIdentifier(&xing.NodeIdentifier{}),
		xing.SetSerializer(&MyUser{}))
	_assert(err)
	typ, ret, err := producer.Call("host.agent", "restart_pod", user.User{
		Id:      1,
		Name:    "Simon",
		Email:   "simon@cool.com",
		Country: "CN",
	})
	_assert(err)
	fmt.Printf("returned: %s, %v\n", typ, ret)
}
