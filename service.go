package xing

import (
	"fmt"
	"time"

	"github.com/micro/go-micro/registry"
	"github.com/sirupsen/logrus"
)

// Service ...
type Service struct {
	Name     string
	Instance string
	Address  string
	Port     int
	Tags     map[string]string
}

func fromMicroService(s *registry.Service) []*Service {
	svcs := make([]*Service, 0, 0)
	for _, n := range s.Nodes {
		svcs = append(svcs, &Service{
			Name:     s.Name,
			Instance: n.Id,
			Address:  n.Address,
			Port:     n.Port,
			Tags:     s.Metadata,
		})
	}
	return svcs
}

func toMicroService(s *Service) *registry.Service {
	return &registry.Service{
		Name:     s.Name,
		Version:  "0.1",
		Metadata: s.Tags,
		Nodes: []*registry.Node{
			{Id: s.Instance},
			{Address: s.Address},
		},
	}
}

// Registrator ...
type Registrator interface {
	Register(s *Service, ttl time.Duration) error
	Deregister(s *Service) error
	GetService(name string, selector Selector) (*Service, error)
}

// ConsulRegistrator ...
type ConsulRegistrator struct {
	reg registry.Registry
}

// NewConsulRegistrator ...
func NewConsulRegistrator() Registrator {
	// FIXME: add options
	reg := registry.NewRegistry()
	cr := &ConsulRegistrator{reg: reg}
	return cr
}

// Register ...
func (cr *ConsulRegistrator) Register(s *Service, ttl time.Duration) error {
	nodes := make([]*registry.Node, 0, 0)
	nodes = append(nodes, &registry.Node{
		Id:      s.Instance,
		Address: s.Address,
		Port:    s.Port,
	})
	logrus.Infof("Registering %s from %s:%d", s.Name, s.Address, s.Port)
	err := cr.reg.Register(&registry.Service{
		Name:     s.Name,
		Version:  "0.1",
		Metadata: s.Tags,
		Nodes:    nodes,
	}, registry.RegisterTTL(ttl))
	if err != nil {
		logrus.Infof("Failed to to register service %s: %v", s.Name, err)
	}
	return err
}

// Deregister ...
func (cr *ConsulRegistrator) Deregister(s *Service) error {
	logrus.Infof("Deregistering %s from %s:%d", s.Name, s.Address, s.Port)
	return cr.reg.Deregister(toMicroService(s))
}

// GetService ...
func (cr *ConsulRegistrator) GetService(name string, selector Selector) (*Service, error) {
	svcs, err := cr.reg.GetService(name)
	if err != nil {
		logrus.Errorf("Failed to get service %v", err)
		return nil, err
	}
	// XXX: I still don't know why svcs is an array
	for _, svc := range svcs {
		all := fromMicroService(svc)
		s := selector.Select(all)
		if s != nil {
			return s, nil
		}
	}
	return nil, fmt.Errorf("Unable to find service instance matching selector")
}
