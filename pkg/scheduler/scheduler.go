package scheduler

import (
	"errors"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
)

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

// Reconstructs the queue from dispatchable.
func (s *Scheduler) Reconstruct() bool {
	dispatchable, ok := s.apiClient.GetDispatchable()
	if !ok {
		return false
	}

	s.queueMutex.Lock()
	defer s.queueMutex.Unlock()
	s.queue = make([]Build, 0)

	current, err := s.capacityProvider.ListBuilds()
	if err != nil {
		return false
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
				if b.Equal(&curBuild) {
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
	return true
}

// Update graph and then queue.
func (s *Scheduler) Update() bool {
	ok := true
	for _, tuple := range s.tuples {
		cleanOk := s.apiClient.Clean(tuple.Target)
		ok = ok && cleanOk
	}
	s.l.Info("Cleaned all targets in graph")
	ok = ok && s.Reconstruct()
	return ok
}

// Start the scheduler.
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
