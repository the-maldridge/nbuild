package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/config"
	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/http"
	"github.com/the-maldridge/nbuild/pkg/scheduler"
	_ "github.com/the-maldridge/nbuild/pkg/scheduler/local"
	_ "github.com/the-maldridge/nbuild/pkg/scheduler/nomad"
	"github.com/the-maldridge/nbuild/pkg/storage"

	_ "github.com/the-maldridge/nbuild/pkg/storage/bc"
)

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

	storage.SetLogger(appLogger)
	storage.DoCallbacks()
	store, err := storage.Initialize("bitcask")
	if err != nil {
		appLogger.Error("Couldn't initialize storage", "error", err)
		return
	}

	mgr := graph.NewManager(appLogger, cfg.Specs)
	mgr.EnablePersistence(store)
	mgr.Bootstrap()
	mgr.SetIndexURLs(cfg.RepoDataURLs)
	mgr.Clean()

	scheduler.SetLogger(appLogger)
	scheduler.DoCallbacks()
	cap, err := scheduler.ConstructCapacityProvider(cfg.CapacityProvider)
	if err != nil {
		appLogger.Error("Couldn't initialize capacity provider", "error", err)
		return
	}
	cap.SetSlots(cfg.BuildSlots)
	scheduler, err := scheduler.NewScheduler(appLogger, cap, "localhost:8080")
	if err != nil {
		appLogger.Error("Couldn't initialize scheduler", "error", err)
		return
	}
	go scheduler.Run()

	srv.Mount("/api/scheduler", scheduler.HTTPEntry())
	srv.Mount("/api/graph", mgr.HTTPEntry())
	go srv.Serve(":8080")

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt)

	<-stop

	appLogger.Info("Shutting down")
	store.Close()
	appLogger.Info("Goodbye!")
}
