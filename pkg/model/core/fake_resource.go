package core

import (
	"context"
	"github.com/pkg/errors"
)

var _ Resource = &FakeResource{}

func NewFakeResource(stack Stack, resType string, id string, spec FakeResourceSpec, status *FakeResourceStatus) *FakeResource {
	r := &FakeResource{
		resType: resType,
		id:      id,
		Spec:    spec,
		Status:  status,
	}
	stack.AddResource(r)
	return r
}

func (r *FakeResource) Type() string {
	return r.resType
}

func (r *FakeResource) ID() string {
	return r.id
}

// register dependencies for LoadBalancer.
func (r *FakeResource) registerDependencies(stack Stack) {
	for _, field := range r.Spec.FieldA {
		for _, dep := range field.Dependencies() {
			stack.AddDependency(dep, r)
		}
	}
}

func (r *FakeResource) FieldB() StringToken {
	return NewResourceFieldStringToken(r, "status/fieldB",
		func(ctx context.Context, res Resource, fieldPath string) (s string, err error) {
			r := res.(*FakeResource)
			if r.Status == nil {
				return "", errors.Errorf("FakeResource is not fulfilled yet: %v", r.ID())
			}
			return r.Status.FieldB, nil
		},
	)
}

type FakeResource struct {
	resType string
	id      string

	Spec   FakeResourceSpec    `json:"spec"`
	Status *FakeResourceStatus `json:"status,omitempty"`
}

type FakeResourceSpec struct {
	FieldA []StringToken `json:"fieldA"`
}

type FakeResourceStatus struct {
	FieldB string `json:"fieldB"`
}
