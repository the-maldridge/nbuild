package scheduler

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/types"
)

type NoCapacityError struct{}

func (e NoCapacityError) Error() string {
	return "no capacity"
}

// A Build is all the information required for a build
type Build struct {
	Spec types.SpecTuple
	Pkg  string
	Rev  string
}

// CapacityProviders are a way for packages to be built.
type CapacityProvider interface {
	DispatchBuild(Build) error
	ListBuilds() ([]Build, error)
}

// Scheduler makes builds ready + dispatches them using a CapacityProvider.
type Scheduler struct {
	l hclog.Logger

	queue      []Build
	queueMutex *sync.Mutex
	tuples     []types.SpecTuple

	apiClient        *graph.APIClient
	capacityProvider CapacityProvider
}
