package scheduler

import (
	"fmt"
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
	if err := s.Reconstruct(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %s", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
