package xing

import "context"

// CallOption ...
type CallOption struct {
}

// Request ...
type Request struct {
	ctx     context.Context
	target  string
	method  string
	payload interface{}
}
