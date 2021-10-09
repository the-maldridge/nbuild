package graph

import (
	"encoding/json"
	"path"
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/source"
	"github.com/the-maldridge/nbuild/pkg/storage"
	"github.com/the-maldridge/nbuild/pkg/repo"
	"github.com/the-maldridge/nbuild/pkg/types"
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

	x.idx = repo.NewIndexService(x.l)
	return &x
}

// SetIndexURLs feeds in the paths to the URLs for each index related
// to the archs that are being built by each spec.  The keys in this
// map should be the targets from the SpecTuples.
func (m *Manager) SetIndexURLs(urls map[string]map[string]string) {
	for arch, indexes := range urls {
		for repo, index := range indexes {
			m.idx.LoadIndex(arch, repo, index)
		}
	}
}

// EnablePersistence provides a way to allow the graph manager to
// persist storage atoms for each graph.  If not enabled, graphs will
// not be persisted or loaded.
func (m *Manager) EnablePersistence(s storage.Storage) {
	m.storage = s
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

	var err error
	m.rev, err = m.cm.At()
	if err != nil {
		m.l.Error("Error retrieving git hash", "error", err)
		return err
	}
	m.loadGraphs()

	var wg sync.WaitGroup
	for spec, graph := range m.graphs {
		wg.Add(1)
		go func(spec string, graph *PkgGraph) {
			if graph.atom.Rev == m.rev {
				wg.Done()
				return
			}
			m.l.Info("Importing graph", "spec", spec)
			if err := graph.ImportAll(); err != nil {
				m.l.Warn("Error importing all packages", "error", err)
			}
			graph.atom.Rev = m.rev
			wg.Done()
		}(spec, graph)
	}
	wg.Wait()
	m.persistGraphs()
	return nil
}

// UpdateCheckout fetches new references from git
func (m *Manager) UpdateCheckout() error {
	return m.cm.Fetch()
}

// SyncTo causes the graphs to all sync to a specific point in
// history.
func (m *Manager) SyncTo(hash string) error {
	changed, err := m.cm.Checkout(hash)
	if err != nil {
		m.l.Error("Error updating checkout", "error", err)
		return err
	}
	m.rev = hash
	var wg sync.WaitGroup
	for spec, graph := range m.graphs {
		wg.Add(1)
		go func(spec string, graph *PkgGraph) {
			m.l.Debug("Syncing graph", "spec", spec)
			if err := graph.ImportChanged(changed); err != nil {
				m.l.Error("Error syncing changes", "error", err, "spec", spec)
			}
			graph.atom.Rev = m.rev
			wg.Done()
		}(spec, graph)
	}
	wg.Wait()
	m.persistGraphs()
	m.l.Info("Synced", "changed", changed)
	return nil
}

// Clean attempts to reconcile the graph with the repo index service
// to determine what packages are present vs what is missing.
func (m *Manager) Clean() {
	for spec, graph := range m.graphs {
		m.l.Debug("Attempting to clean graph", "spec", spec)
		for _, pkg := range graph.GetDirty() {
			p, err := m.idx.GetPackage(SpecTupleFromString(spec).Target, pkg.Name)
			if err != nil {
				m.l.Debug("Package errors while cleaning", "spec", spec, "package", pkg, "error", err)
				continue
			}
			// This looks so dumb, but is because the
			// version field in the index includes the
			// package name.
			if p.Version == pkg.Name + "-" + pkg.Version {
				m.l.Trace("Cleaning Package", "spec", spec, "package", pkg.Name, "version", pkg.Version)
				graph.CleanPkg(pkg.Name)
			} else {
				m.l.Trace("Package remains dirty", "package", pkg.Name, "have", p.Version, "want", pkg)
			}
		}
		m.l.Debug("Remaining dirty packages", "count", len(m.GetDirty(SpecTupleFromString(spec))))
	}
	m.persistGraphs()
}

// GetDirty returns a list of packages that are dirty in the graph.
func (m *Manager) GetDirty(spec SpecTuple) []*types.Package {
	graph, ok := m.graphs[spec.String()]
	if !ok {
		return nil
	}
	return graph.GetDirty()
}

func (m *Manager) loadGraphs() {
	if m.storage == nil {
		m.l.Warn("Storage is unavailable, graphs will not be imported")
		return
	}

	for spec, graph := range m.graphs {
		m.l.Debug("Attempting to load graph", "spec", spec)
		graph.PkgsMutex.Lock()
		graph.AuxMutex.Lock()
		defer graph.AuxMutex.Unlock()
		defer graph.PkgsMutex.Unlock()
		graphbytes, err := m.storage.Get([]byte(path.Join("graph", spec)))
		if err != nil {
			m.l.Warn("Error loading graph", "error", err)
			continue
		}
		if err := json.Unmarshal(graphbytes, &graph.atom); err != nil {
			m.l.Warn("Error loading graph", "error", err)
			continue
		}
		m.l.Debug("Loaded Graph", "spec", spec, "count", len(graph.atom.Pkgs), "rev", graph.atom.Rev)
	}
}

func (m *Manager) persistGraphs() {
	if m.storage == nil {
		return
	}

	for spec, graph := range m.graphs {
		graph.PkgsMutex.Lock()
		graph.AuxMutex.Lock()
		defer graph.AuxMutex.Unlock()
		defer graph.PkgsMutex.Unlock()
		graphbytes, err := json.Marshal(graph.atom)
		if err != nil {
			m.l.Warn("Error serializing graph", "error", err)
			continue
		}
		if err := m.storage.Put([]byte(path.Join("graph", spec)), graphbytes); err != nil {
			m.l.Warn("Error writing graph", "error", err)
			continue
		}
	}
}
