package http

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/hashicorp/go-hclog"
)

// New initializes the server with its default routers.
func New(l hclog.Logger) (*Server, error) {
	s := Server{
		l: l.Named("http"),
		r: chi.NewRouter(),
		n: &http.Server{},
	}

	s.r.Use(middleware.Logger)
	s.r.Use(middleware.Heartbeat("/healthz"))

	s.r.Get("/", s.rootIndex)

	return &s, nil
}

// Serve binds, initializes the mux, and serves forever.
func (s *Server) Serve(bind string) error {
	s.l.Info("HTTP is starting")
	s.n.Addr = bind
	s.n.Handler = s.r
	return s.n.ListenAndServe()
}

func (s *Server) rootIndex(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "nbuild is running, check other handlers for more information")
}

// Mount attaches a set of routes to the subpath specified by the path
// argument.
func (s *Server) Mount(path string, router chi.Router) {
	s.r.Mount(path, router)
}
