package main

import (
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/repo"
	"github.com/the-maldridge/nbuild/pkg/source"
	"github.com/the-maldridge/nbuild/pkg/storage"
	_ "github.com/the-maldridge/nbuild/pkg/storage/bc"
	"github.com/the-maldridge/nbuild/pkg/types"
)

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "nbuild",
		Level: hclog.LevelFromString("DEBUG"),
	})
	appLogger.Info("nbuild is initializing")

	switch os.Args[1] {
	case "import":
		srctree := graph.New(appLogger, types.SpecTuple{"x86_64", "x86_64"})
		if err := srctree.LoadVirtual(); err != nil {
			return
		}
		appLogger.Info("Importer performing initial pass")
		if err := srctree.ImportAll(); err != nil {
			return
		}
	case "repodata":
		rss := repo.NewIndexService(appLogger)
		appLogger.Info("repodata load", "error", rss.LoadIndex("x86_64", "main", "http://mirrors.servercentral.com/voidlinux/current/x86_64-repodata"))
		appLogger.Info("repodata load", "error", rss.LoadIndex("x86_64", "nonfree", "http://mirrors.servercentral.com/voidlinux/current/nonfree/x86_64-repodata"))
		p, _ := rss.GetPackage("x86_64", "NetAuth")
		appLogger.Info("Example package", "pkg", p)
		p, _ = rss.GetPackage("x86_64", "scream-raw-ivshmem")
		appLogger.Info("Example package", "pkg", p)
	case "git":
		repo := source.New(appLogger)
		repo.Path = "void-packages"
		repo.Bootstrap()
		repo.Fetch()
		// Some random commit
		repo.Checkout("61ba6baece2f5a065cc821f986cba3a4abd7c6e6")
	case "multigraph":
		mgr := graph.NewManager(appLogger, []types.SpecTuple{{"x86_64", "x86_64"}}}) //, {"x86_64", "armv7l"}})
		mgr.SetIndexURLs(map[string]map[string]string{
			"x86_64": {
				"main": "http://mirrors.servercentral.com/voidlinux/current/x86_64-repodata",
				"nonfree": "http://mirrors.servercentral.com/voidlinux/current/nonfree/x86_64-repodata",
			},
			"armv7l": {
				"main": "http://mirrors.servercentral.com/voidlinux/current/armv7l-repodata",
				"nonfree": "http://mirrors.servercentral.com/voidlinux/current/nonfree/armv7l-repodata",
			},
		})

		storage.SetLogger(appLogger)
		storage.DoCallbacks()
		store, err := storage.Initialize("bitcask")
		if err != nil {
			appLogger.Error("Couldn't initialize storage", "error", err)
			return
		}
		mgr.EnablePersistence(store)
		appLogger.Info("Bootstrapping multigraph", "return", mgr.Bootstrap())
		mgr.UpdateCheckout()
		mgr.SyncTo("e7ca6798247fb7a2d6373dbc48697041df4ebd67")
		mgr.Clean()
		spec := types.SpecTuple{"x86_64","x86_64"}
		dirty := mgr.GetDirty(spec)
		for _, p := range dirty {
			appLogger.Info("Dirty Package", "spec", spec, "package", p)
		}
		appLogger.Info("Total Dirty Packages", "count", len(dirty))
		dispatchable := mgr.GetDispatchable()
		for spec, p := range dispatchable {
			appLogger.Info("Dispatchable Package", "spec", spec, "package", p)
		}
		store.Close()
	}
}
