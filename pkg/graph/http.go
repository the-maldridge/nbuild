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

	r.Post("/pkgs/{host}/{target}/{pkg}/fail", m.httpFailPkg)
	r.Post("/pkgs/{host}/{target}/{pkg}/unfail", m.httpUnfailPkg)
	r.Post("/clean/{target}", m.httpCleanTarget)
	r.Post("/syncto/{sha}", m.httpSyncToRev)

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
	dispatchable := make(map[string][]string)
	for spec, list := range m.GetDispatchable() {
		dispatch := make(map[string]struct{}, len(list))
		// dedup the list (subpkgs add the parent multiple
		// times)
		for _, pkg := range list {
			dispatch[pkg.Name] = struct{}{}
		}

		ret := make([]string, len(dispatch))
		i := 0
		for key := range dispatch {
			ret[i] = key
			i++
		}
		dispatchable[spec.String()] = ret
	}

	out := struct {
		Pkgs     map[string][]string
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

func (m *Manager) httpFailPkg(w http.ResponseWriter, r *http.Request) {
	graph, ok := m.graphs[types.SpecTuple{chi.URLParam(r, "host"), chi.URLParam(r, "target")}.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := graph.FailPkg(chi.URLParam(r, "pkg")); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (m *Manager) httpUnfailPkg(w http.ResponseWriter, r *http.Request) {
	graph, ok := m.graphs[types.SpecTuple{chi.URLParam(r, "host"), chi.URLParam(r, "target")}.String()]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err := graph.UnfailPkg(chi.URLParam(r, "pkg")); err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (m *Manager) httpCleanTarget(w http.ResponseWriter, r *http.Request) {
	tgt := chi.URLParam(r, "target")

	enc := json.NewEncoder(w)
	if err := m.idx.ReloadArch(tgt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		out := struct {
			Error string
		}{
			Error: err.Error(),
		}
		enc.Encode(out)
		return
	}

	for spec, graph := range m.graphs {
		if types.SpecTupleFromString(spec).Target != tgt {
			continue
		}
		m.CleanSpec(types.SpecTupleFromString(spec), graph)
	}
}

func (m *Manager) httpSyncToRev(w http.ResponseWriter, r *http.Request) {
	if err := m.UpdateCheckout(); err != nil {
		m.l.Warn("Error updating", "error", err)
		return
	}

	if err := m.SyncTo(chi.URLParam(r, "sha")); err != nil {
		jsonError(w, err, http.StatusInternalServerError)
		return
	}

	m.Clean()
	w.WriteHeader(http.StatusNoContent)
}

func jsonError(w http.ResponseWriter, err error, code int) {
	enc := json.NewEncoder(w)
	w.WriteHeader(code)
	out := struct {
		Error string
	}{
		Error: err.Error(),
	}
	enc.Encode(out)
}
