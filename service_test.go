package xing

import (
	"strconv"
	"testing"
	"time"
)

// Make sure you have consul server running.
// e.g. docker run -d -p 8500:8500 consul

func Test_Service_00(t *testing.T) {
	name := "api.auth"
	reg := NewConsulRegistrator()
	register := func(inst int) {
		s := &Service{
			Name:     name,
			Instance: strconv.Itoa(inst),
			Port:     9000,
			Tags:     nil,
		}
		reg.Register(s, 10*time.Second)
	}

	for i := 0; i < 10; i++ {
		register(i)
	}
}

func Test_Service_01(t *testing.T) {
	name := "api.auth"
	reg := NewConsulRegistrator()
	s, err := reg.GetService(name, NewInstanceSelector("3"))
	if err != nil {
		t.Error("failed.\n")
	}
	if s.Instance != "3" {
		t.Error("wrong instance found\n")
	}

	_, err = reg.GetService(name, NewInstanceSelector("90"))
	if err == nil {
		t.Error("failed\n")
	}
}

func Test_Service_02(t *testing.T) {
	time.Sleep(10 * time.Second)
	name := "api.auth"
	reg := NewConsulRegistrator()
	_, err := reg.GetService(name, NewInstanceSelector("3"))
	if err == nil {
		t.Error("Failed. Should have expired\n")
	}
}
