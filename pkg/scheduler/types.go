package scheduler

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/types"
)

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
	SetSlots(map[string]int)
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
