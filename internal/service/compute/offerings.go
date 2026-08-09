package compute

import (
	"fmt"

	"github.com/virtfoundry/core/internal/platform"
	"github.com/virtfoundry/core/internal/platform/store"
	"github.com/virtfoundry/core/internal/service/shared"
)

// CreateServiceOfferingInput registers a platform compute size.
type CreateServiceOfferingInput struct {
	Name         string
	DisplayName  string
	CPU          int
	MemoryMi     int64
	DedicatedCPU bool
}

// UpdateServiceOfferingInput patches an existing offering.
type UpdateServiceOfferingInput struct {
	DisplayName  string
	CPU          int
	MemoryMi     int64
	State        string
	DedicatedCPU *bool
}

func (s *Service) ListServiceOfferings(activeOnly bool) []*platform.ServiceOffering {
	return s.store.ListServiceOfferings(activeOnly)
}

func (s *Service) CreateServiceOffering(in CreateServiceOfferingInput) (*platform.ServiceOffering, error) {
	name := shared.SanitizeSlug(in.Name)
	if name == "" {
		return nil, fmt.Errorf("invalid offering name")
	}
	if in.CPU <= 0 {
		return nil, fmt.Errorf("cpu must be greater than 0")
	}
	if in.MemoryMi <= 0 {
		return nil, fmt.Errorf("memory must be greater than 0")
	}
	if _, ok := s.store.GetServiceOfferingByName(name); ok {
		return nil, fmt.Errorf("offering name already exists")
	}
	displayName := in.DisplayName
	if displayName == "" {
		displayName = name
	}
	o := &platform.ServiceOffering{
		ID: store.NewID(), Name: name, DisplayName: displayName,
		CPU: in.CPU, MemoryMi: in.MemoryMi, DedicatedCPU: in.DedicatedCPU, State: "Active",
		CreatedAt: store.Now(),
	}
	s.store.SaveServiceOffering(o)
	return o, nil
}

func (s *Service) UpdateServiceOffering(id string, in UpdateServiceOfferingInput) (*platform.ServiceOffering, error) {
	o, ok := s.store.GetServiceOffering(id)
	if !ok {
		return nil, fmt.Errorf("service offering not found")
	}
	if in.DisplayName != "" {
		o.DisplayName = in.DisplayName
	}
	if in.CPU > 0 {
		o.CPU = in.CPU
	}
	if in.MemoryMi > 0 {
		o.MemoryMi = in.MemoryMi
	}
	if in.State != "" {
		o.State = in.State
	}
	if in.DedicatedCPU != nil {
		o.DedicatedCPU = *in.DedicatedCPU
	}
	s.store.SaveServiceOffering(o)
	return o, nil
}

func (s *Service) DeleteServiceOffering(id string) error {
	o, ok := s.store.GetServiceOffering(id)
	if !ok {
		return fmt.Errorf("service offering not found")
	}
	o.State = "Inactive"
	s.store.SaveServiceOffering(o)
	return nil
}
