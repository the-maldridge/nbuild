package graph

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/source"
)

// NewManager creates a collection of graphs under a single manager
// and returns the manager.  Graphs do not have state on return.
func NewManager(l hclog.Logger, specs []SpecTuple) *Manager {
	x := Manager{
		l:        l.Named("graph"),
		cm:       source.New(l),
		basepath: "void-packages",
		graphs:   make(map[string]*PkgGraph),
		specs:    specs,
	}
	for _, spec := range specs {
		x.graphs[spec.String()] = New(l.Named("graph"), spec)
	}
	return &x
}

// OverrideBasepath allows you to change the location that the
// checkout will be maintained in.
func (m *Manager) OverrideBasepath(b string) {
	m.basepath = b
	m.cm.SetBasepath(b)
}

// Bootstrap performs the initial download of the void-packages repo,
// and performs an import of all configured archs.
func (m *Manager) Bootstrap() error {
	m.cm.SetBasepath(m.basepath)
	if err := m.cm.Bootstrap(); err != nil {
		m.l.Error("Error bootstrapping", "error", err)
		return err
	}
	m.cm.Checkout("0273e73426895f4833ff636b8b72de90414db639") // Testing to always start at a known place

	var wg sync.WaitGroup
	for spec, graph := range m.graphs {
		wg.Add(1)
		go func(spec string, graph *PkgGraph) {
			m.l.Info("Importing graph", "spec", spec)
			if err := graph.ImportAll(); err != nil {
				m.l.Warn("Error importing all packages", "error", err)
			}
			wg.Done()
		}(spec, graph)
	}
	wg.Wait()
	return nil
}

// SyncTo causes the graphs to all sync to a specific point in
// history.
func (m *Manager) SyncTo(hash string) error {
	changed, err := m.cm.Checkout(hash)
	if err != nil {
		m.l.Error("Error updating checkout", "error", err)
		return err
	}
	var wg sync.WaitGroup
	for spec, graph := range m.graphs {
		wg.Add(1)
		go func(spec string, graph *PkgGraph) {
			m.l.Debug("Syncing graph", "spec", spec)
			if err := graph.ImportChanged(changed); err != nil {
				m.l.Error("Error syncing changes", "error", err, "spec", spec)
			}
			wg.Done()
		}(spec, graph)
	}
	wg.Wait()
	m.l.Info("Synced", "changed", changed)
	return nil
}
