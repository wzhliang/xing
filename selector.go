package xing

// Selector ...
type Selector interface {
	Select(s []*Service) *Service
}

// InstanceSelector ...
type InstanceSelector struct {
	id string
}

// Select ...
func (is *InstanceSelector) Select(services []*Service) *Service {
	for _, s := range services {
		if s.Instance == is.id {
			return s
		}
	}

	return nil
}

// NewInstanceSelector ...
func NewInstanceSelector(id string) Selector {
	return &InstanceSelector{id: id}
}
