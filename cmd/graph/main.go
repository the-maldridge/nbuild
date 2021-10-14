package main

import (
	"os"
	"os/signal"

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
	mgr.SetIndexURLs(map[string]map[string]string{
		"x86_64": {
			"main":    "http://mirrors.servercentral.com/voidlinux/current/x86_64-repodata",
			"nonfree": "http://mirrors.servercentral.com/voidlinux/current/nonfree/x86_64-repodata",
			"local":   "file://test-checkout/hostdir/binpkgs/x86_64-repodata",
		},
	})

	srv.Mount("/api/graph", mgr.HTTPEntry())
	go srv.Serve(":8080")

	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt)

	<-stop

	appLogger.Info("Shutting down")
	store.Close()
	appLogger.Info("Goodbye!")
}
