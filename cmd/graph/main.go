package main

import (
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/http"
	"github.com/the-maldridge/nbuild/pkg/storage"
	"github.com/the-maldridge/nbuild/pkg/types"

	_ "github.com/the-maldridge/nbuild/pkg/storage/bc"
)

func main() {
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

	specs := []types.SpecTuple{{"x86_64", "x86_64"}}
	mgr := graph.NewManager(appLogger, specs)
	mgr.EnablePersistence(store)
	mgr.Bootstrap()

	srv.Mount("/api/graph", mgr.HTTPEntry())
	srv.Serve(":8080")
}
