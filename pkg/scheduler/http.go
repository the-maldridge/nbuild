package scheduler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// HTTPEntry provides the mountpoint for this service into the shared
// webserver routing tree.
func (s *Scheduler) HTTPEntry() chi.Router {
	r := chi.NewRouter()

	r.Get("/done", s.httpDone)
	return r
}

func (s *Scheduler) httpDone(w http.ResponseWriter, r *http.Request) {
	ok := s.Reconstruct()
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
