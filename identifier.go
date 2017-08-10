package xing

import "os"

// Identifier ...
type Identifier interface {
	InstanceID() string
}

// NoneIdentifier ...
type NoneIdentifier struct {
}

// InstanceID ...
func (p *NoneIdentifier) InstanceID() string {
	return "x"
}

// PodIdentifier ...
type PodIdentifier struct {
}

// InstanceID ...
func (p *PodIdentifier) InstanceID() string {
	return os.Getenv("MY_POD_ID")
}

// NodeIdentifier ...
type NodeIdentifier struct {
}

// InstanceID ...
func (p *NodeIdentifier) InstanceID() string {
	return os.Getenv("MY_NODE_ID")
}
