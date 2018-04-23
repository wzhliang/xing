package xing

import "testing"

func _assert(t *testing.T, cond bool, msg string) {
	if !cond {
		t.Error(msg)
	}
}

func Test_Xing_00(t *testing.T) {
	T := func(cond bool, msg string) {
		_assert(t, cond, msg)
	}
	T(topicLength("hello.world") == 2, "...")
	T(topicLength("hello.world.and.you") == 4, "...")
}
