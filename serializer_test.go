package xing

import (
	"bytes"
	"testing"
)

func Test_00(t *testing.T) {
	T := func(cond bool, msg string) {
		_assert(t, cond, msg)
	}

	s := PlainSerializer{}
	T(s.ContentType() == "text/plain", "...")

	data, err := s.Marshal("hello")
	T(err == nil, "...")
	T(0 == bytes.Compare(data, []byte("hello")), "...")
}
