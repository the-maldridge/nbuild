package main

import (
	"encoding/json"
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
	"github.com/the-maldridge/nbuild/pkg/repo"
)

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "nbuild",
		Level: hclog.LevelFromString("DEBUG"),
	})
	appLogger.Info("nbuild is initializing")

	srctree := graph.New(appLogger)

	switch os.Args[1] {
	case "import":
		if err := srctree.LoadVirtual(); err != nil {
			return
		}

		appLogger.Info("Importer performing initial pass")
		if err := srctree.Import(); err != nil {
			return
		}

		appLogger.Info("Import Complete, Resolving Graph")
		srctree.ResolveGraph()
		appLogger.Info("Resolution complete")

		f, _ := os.Create("state.json")
		defer f.Close()

		enc := json.NewEncoder(f)

		if err := enc.Encode(srctree); err != nil {
			appLogger.Error("Error marshalling", "error", err)
			return
		}
	case "repodata":
		rss := repo.NewIndexService(appLogger)
		appLogger.Info("repodata load", "error", rss.LoadIndex("http://mirrors.servercentral.com/voidlinux/current/x86_64-repodata"))
		appLogger.Info("repodata contains some packages", "count", rss.PkgCount())
		p, _ := rss.GetPackage("NetAuth")
		appLogger.Info("Example package", "pkg", p)
	}
}
