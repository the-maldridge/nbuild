package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
)

// NewScheduler returns a scheduler instance using the listed capacity
// provider.  The capacity provider will run builds at a maximum
// degree of parallelism that is implementation specific and
// potentially dynamic.
func NewScheduler(l hclog.Logger, c CapacityProvider, url string) *Scheduler {
	x := Scheduler{
		l:                l.Named("scheduler"),
		capacityProvider: c,
		apiClient:        graph.NewAPIClient(l),
		queueMutex:       new(sync.Mutex),
	}
	x.apiClient.Url = url

	return &x
}

// Pops a build off the queue and hands it off to the CapacityProvider.
func (s *Scheduler) send() error {
	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()

	if len(s.queue) == 0 {
		return errors.New("none in queue")
	}
	if err := s.capacityProvider.DispatchBuild(s.queue[0]); err != nil {
		s.l.Trace("Unable to dispatch right now", "build", s.queue[0], "err", err)
		return err
	}
	s.l.Trace("Dispatching", "build", s.queue[0])
	s.queue = s.queue[1:]
	return nil
}

// Reconstruct rebuidls the queue from what is currently known to be
// running and what is currently dispatchable.
func (s *Scheduler) Reconstruct() error {
	dispatchable, err := s.apiClient.GetDispatchable()
	if err != nil { return err }

	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.queue = make([]Build, 0)

	current, err := s.capacityProvider.ListBuilds()
	if err != nil {
		return err
	}
	for tuple, pkgs := range dispatchable.Pkgs {
		for _, pkg := range pkgs {
			b := Build{
				Spec: tuple,
				Pkg:  pkg,
				Rev:  dispatchable.Rev,
			}
			alreadyBuilding := false
			for _, curBuild := range current {
				if b.Equal(curBuild) {
					alreadyBuilding = true
					break
				}
			}
			if !alreadyBuilding {
				s.queue = append(s.queue, b)
			}
		}
		s.tuples = append(s.tuples, tuple)
	}
	s.l.Info("Successfully reconstructed queue")
	return nil
}

// Update graph and then queue.
func (s *Scheduler) Update() error {
	for _, tuple := range s.tuples {
		if err := s.apiClient.Clean(tuple.Target); err != nil {
			return err
		}
	}
	s.l.Info("Cleaned all targets in graph")
	return nil
}

// Run loads up the initial data for the scheduler, and serves
// forever.
func (s *Scheduler) Run() {
	s.Reconstruct() // Get tuples
	s.Update()      // Now get real dispatchable
	for {
		err := s.send()
		if err != nil {
			// Don't try to send too often
			time.Sleep(time.Second)
		}
	}
}
