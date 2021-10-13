package graph

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// HTTPEntry provides the mountpoint for this service into the shared
// webserver routing tree.
func (m *Manager) HTTPEntry() chi.Router {
	r := chi.NewRouter()

	r.Get("/atom/{host}/{target}", m.httpDumpAtom)
	r.Get("/pkgs/{host}/{target}/{pkg}", m.httpDumpPkg)
	r.Get("/dirty/{host}/{target}", m.httpDumpDirty)
	r.Get("/dispatchable", m.httpDumpDispatch)

	return r
}

func (m *Manager) httpDumpAtom(w http.ResponseWriter, r *http.Request) {
	graph, ok := m.graphs[types.SpecTuple{chi.URLParam(r, "host"), chi.URLParam(r, "target")}.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	graph.PkgsMutex.Lock()
	defer graph.PkgsMutex.Unlock()

	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc.Encode(graph.atom)
}

func (m *Manager) httpDumpPkg(w http.ResponseWriter, r *http.Request) {
	graph, ok := m.graphs[types.SpecTuple{chi.URLParam(r, "host"), chi.URLParam(r, "target")}.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	graph.PkgsMutex.Lock()
	defer graph.PkgsMutex.Unlock()
	pkg, ok := graph.atom.Pkgs[chi.URLParam(r, "pkg")]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc.Encode(pkg)

}

func (m *Manager) httpDumpDirty(w http.ResponseWriter, r *http.Request) {
	spec := types.SpecTuple{chi.URLParam(r, "host"), chi.URLParam(r, "target")}
	graph, ok := m.graphs[spec.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	out := struct {
		Rev  string
		Pkgs []*types.Package
	}{
		Rev:  graph.atom.Rev,
		Pkgs: m.GetDirty(spec),
	}

	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc.Encode(out)
}

func (m *Manager) httpDumpDispatch(w http.ResponseWriter, r *http.Request) {
	// Its necessary to re-shape what we get from the API due to
	// the limitations of the JSON format.  Specifically the map
	// keys MUST be strings.
	dispatchable := make(map[string][]*types.Package)
	for spec, list := range m.GetDispatchable() {
		dispatchable[spec.String()] = list
	}

	out := struct {
		Pkgs     map[string][]*types.Package
		Revision string
	}{
		Pkgs:     dispatchable,
		Revision: m.rev,
	}

	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	enc.Encode(out)
}
