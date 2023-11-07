package graph

import (
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/storage"
	"github.com/the-maldridge/nbuild/pkg/types"
)

// WithLogger sets up the logging instance for the graph manager.
func WithLogger(l hclog.Logger) Option {
	return func(m *Manager) {
		m.l = l.Named("graph")
	}
}

// WithSpecs provides a list of SpecTuples that this manager should
// supervise.
func WithSpecs(specs []types.SpecTuple) Option {
	return func(m *Manager) {
		m.specs = specs
		for _, spec := range specs {
			m.graphs[spec.String()] = New(m.l.Named("graph"), spec)
		}
	}
}

// WithIndexURLs sets up the paths for the URLs for each index of each
// arch in each spec.  The keys of the map should be the targets from
// the SpecTuples.
func WithIndexURLs(urls map[string]map[string]string) Option {
	return func(m *Manager) {
		for arch, indexes := range urls {
			for repo, index := range indexes {
				m.idx.LoadIndex(arch, repo, index)
			}
		}
	}
}

// WithStorage enables persistance of the graphs to a durable
// datastore.
func WithStorage(s storage.Storage) Option {
	return func(m *Manager) {
		m.storage = s
	}
}

// WithBasePath points the graph manager at the location on disk that
// the checkout of void-packages will be maintained in.
func WithBasePath(b string) Option {
	return func(m *Manager) {
		m.basepath = b
	}
}
