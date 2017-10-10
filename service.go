package xing

import (
	"fmt"
	"time"

	"github.com/micro/go-micro/registry"
	etcd "github.com/micro/go-plugins/registry/etcdv3"
	"github.com/rs/zerolog/log"
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

type commonRegistrator struct {
	reg registry.Registry
}

// NewConsulRegistrator ...
func NewConsulRegistrator(opts ...registry.Option) Registrator {
	reg := registry.NewRegistry(opts...)
	cr := &commonRegistrator{reg: reg}
	return cr
}

// NewEtcdRegistrator ...
func NewEtcdRegistrator(opts ...registry.Option) Registrator {
	reg := etcd.NewRegistry(opts...)
	cr := &commonRegistrator{reg: reg}
	return cr
}

// Register ...
func (cr *commonRegistrator) Register(s *Service, ttl time.Duration) error {
	nodes := make([]*registry.Node, 0, 0)
	nodes = append(nodes, &registry.Node{
		Id:      s.Instance,
		Address: s.Address,
		Port:    s.Port,
	})
	log.Info().Str("name", s.Name).Str("addr", s.Address).Int("port", s.Port).
		Msg("Registering service")
	err := cr.reg.Register(&registry.Service{
		Name:     s.Name,
		Version:  "0.1",
		Metadata: s.Tags,
		Nodes:    nodes,
	}, registry.RegisterTTL(ttl))
	if err != nil {
		log.Error().Str("name", s.Name).Err(err).Msg("Failed to register service")
	}
	return err
}

// Deregister ...
func (cr *commonRegistrator) Deregister(s *Service) error {
	log.Info().Str("name", s.Name).Str("addr", s.Address).Int("port", s.Port).
		Msg("Deregistering service")
	return cr.reg.Deregister(toMicroService(s))
}

// GetService ...
func (cr *commonRegistrator) GetService(name string, selector Selector) (*Service, error) {
	svcs, err := cr.reg.GetService(name)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get service")
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
