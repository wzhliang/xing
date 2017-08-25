package xing

import "testing"

func _assert(t *testing.T, cond bool, msg string) {
	if !cond {
		t.Error(msg)
	}
}
