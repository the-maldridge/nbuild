package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/config"
	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/http"
	"github.com/the-maldridge/nbuild/pkg/reciever"
	"github.com/the-maldridge/nbuild/pkg/scheduler"
	_ "github.com/the-maldridge/nbuild/pkg/scheduler/local"
	_ "github.com/the-maldridge/nbuild/pkg/scheduler/nomad"
	"github.com/the-maldridge/nbuild/pkg/storage"
	_ "github.com/the-maldridge/nbuild/pkg/storage/bc"
)

type servelet func(hclog.Logger, chan error, *config.Config, *http.Server)
type shutdownHandler func()

var (
	components = map[string]servelet{
		"graph":     doGraph,
		"scheduler": doScheduler,
		"reciever":  doReciever,
	}

	shutdownHandlers []shutdownHandler
)

func doGraph(appLogger hclog.Logger, errCh chan error, cfg *config.Config, srv *http.Server) {
	storage.SetLogger(appLogger)
	storage.DoCallbacks()
	store, err := storage.Initialize("bitcask")
	if err != nil {
		appLogger.Error("Couldn't initialize storage", "error", err)
		errCh <- err
		return
	}

	shutdownHandlers = append(shutdownHandlers, func() { store.Close() })

	mgr := graph.NewManager(
		graph.WithLogger(appLogger),
		graph.WithSpecs(cfg.Specs),
		graph.WithStorage(store),
		graph.WithIndexURLs(cfg.RepoDataURLs),
	)
	mgr.Bootstrap()
	mgr.Clean()

	srv.Mount("/api/graph", mgr.HTTPEntry())
}

func doScheduler(appLogger hclog.Logger, errCh chan error, cfg *config.Config, srv *http.Server) {
	scheduler.SetLogger(appLogger)
	scheduler.DoCallbacks()
	cap, err := scheduler.ConstructCapacityProvider(cfg.CapacityProvider)
	if err != nil {
		appLogger.Error("Couldn't initialize capacity provider", "error", err)
		errCh <- err
		return
	}
	cap.SetSlots(cfg.BuildSlots)
	scheduler, err := scheduler.NewScheduler(appLogger, cap, "http://localhost:8080/api/graph")
	if err != nil {
		appLogger.Error("Error initializing scheduler", "error", err)
		errCh <- err
		return
	}
	shutdownHandlers = append(shutdownHandlers, scheduler.Stop)
	srv.Mount("/api/scheduler", scheduler.HTTPEntry())
	scheduler.Run()
}

func doReciever(appLogger hclog.Logger, errCh chan error, cfg *config.Config, srv *http.Server) {
	reciever := reciever.NewReciever(appLogger)
	reciever.SetPath(cfg.RepoPath)
	srv.Mount("/api/reciever", reciever.HTTPEntry())
}

func shutdown() {
	for _, f := range shutdownHandlers {
		f()
	}
}

func main() {
	cfg := config.NewConfig()

	if path, ok := os.LookupEnv("NBUILD_CONFIG"); ok {
		if err := cfg.LoadFromFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration data: %s", err)
			os.Exit(2)
		}
	}

	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "nbuild",
		Level: hclog.LevelFromString("DEBUG"),
	})
	appLogger.Info("nbuild is initializing")

	srv, err := http.New(appLogger)
	if err != nil {
		appLogger.Error("Error initializing webserver", "error", err)
		return
	}

	enabledComponents, ok := os.LookupEnv("NBUILD_COMPONENTS")
	if !ok {
		appLogger.Error("NBUILD_COMPONENTS must contain at least one component")
		return
	}

	errCh := make(chan error, 5)
	go func() {
		for {
			err, ok := <-errCh
			if !ok {
				appLogger.Debug("errCh closed, exiting early error handler")
				return
			}
			appLogger.Error("Initialization Error", "error", err)
			shutdown()
			os.Exit(2)
		}
	}()
	for _, c := range strings.Split(strings.ToLower(enabledComponents), ",") {
		cf, ok := components[c]
		if !ok {
			appLogger.Error("Unknown component", "id", c)
			shutdown()
			return
		}
		go cf(appLogger, errCh, cfg, srv)
	}
	appLogger.Debug("Worker fork done")

	bind, ok := os.LookupEnv("NBUILD_BIND")
	if !ok {
		appLogger.Error("NBUILD_BIND must be set to a valid bind address")
		return
	}
	go srv.Serve(bind)

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt)

	<-stop

	appLogger.Info("Shutting down")
	shutdown()
	appLogger.Info("Goodbye!")
}
