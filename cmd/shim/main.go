package main

import (
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/repo"
	"github.com/the-maldridge/nbuild/pkg/source"
	"github.com/the-maldridge/nbuild/pkg/storage"
	_ "github.com/the-maldridge/nbuild/pkg/storage/bc"
)

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "nbuild",
		Level: hclog.LevelFromString("DEBUG"),
	})
	appLogger.Info("nbuild is initializing")

	switch os.Args[1] {
	case "import":
		srctree := graph.New(appLogger, graph.SpecTuple{"x86_64", "x86_64"})
		if err := srctree.LoadVirtual(); err != nil {
			return
		}
		appLogger.Info("Importer performing initial pass")
		if err := srctree.ImportAll(); err != nil {
			return
		}
	case "repodata":
		rss := repo.NewIndexService(appLogger)
		appLogger.Info("repodata load", "error", rss.LoadIndex("http://mirrors.servercentral.com/voidlinux/current/x86_64-repodata"))
		appLogger.Info("repodata contains some packages", "count", rss.PkgCount())
		p, _ := rss.GetPackage("NetAuth")
		appLogger.Info("Example package", "pkg", p)
	case "git":
		repo := source.New(appLogger)
		repo.Path = "void-packages"
		repo.Bootstrap()
		repo.Fetch()
		// Some random commit
		repo.Checkout("61ba6baece2f5a065cc821f986cba3a4abd7c6e6")
	case "multigraph":
		mgr := graph.NewManager(appLogger, []graph.SpecTuple{{"x86_64", "x86_64"}, {"x86_64", "armv7l"}})

		storage.SetLogger(appLogger)
		storage.DoCallbacks()
		store, err := storage.Initialize("bitcask")
		if err != nil {
			appLogger.Error("Couldn't initialize storage", "error", err)
			return
		}
		mgr.EnablePersistence(store)
		appLogger.Info("Bootstrapping multigraph", "return", mgr.Bootstrap())
		mgr.SyncTo("0ee5b487dca9d6a2476beeb93e9a75d2b5751953")
		store.Close()
	}
}
