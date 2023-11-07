package scheduler

import (
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
)

// WithLogger sets the parent logger.
func WithLogger(l hclog.Logger) Option {
	return func(s *Scheduler) error {
		s.l = l.Named("scheduler")
		return nil
	}
}

// WithCapacityProvider provides some capacity to the system.
func WithCapacityProvider(c CapacityProvider) Option {
	return func(s *Scheduler) error {
		s.capacityProvider = c
		return nil
	}
}

// WithGraphURL provides the scheduler with the API endpoint that a
// graph server can be found at.
func WithGraphURL(url string) Option {
	return func(s *Scheduler) error {
		graph, err := graph.NewAPIClient(s.l, url)
		if err != nil {
			return err
		}
		s.apiClient = graph
		return nil
	}
}
