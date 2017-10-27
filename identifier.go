package xing

import (
	"os"

	uuid "github.com/satori/go.uuid"
)

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
	return os.Getenv("MY_POD_NAME")
}

// NodeIdentifier ...
type NodeIdentifier struct {
}

// InstanceID ...
func (p *NodeIdentifier) InstanceID() string {
	return os.Getenv("MY_NODE_NAME")
}

// RandomIdentifier ...
type RandomIdentifier struct {
}

// InstanceID ...
func (p *RandomIdentifier) InstanceID() string {
	return uuid.NewV4().String()[:6]
}
