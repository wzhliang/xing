package xing

import (
	"bytes"
	"testing"
)

func _assert(cond bool, t *testing.T) {
	if !cond {
		t.Error("wtf")

	}
}

func Test_00(t *testing.T) {
	s := PlainSerializer{}
	_assert(s.ContentType() == "text/plain", t)

	data, err := s.Marshal("N/A", "hello")
	_assert(err == nil, t)
	_assert(0 == bytes.Compare(data, []byte("hello")), t)
}
