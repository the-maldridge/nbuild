package main

import (
	"encoding/json"
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/graph"
)

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "nbuild",
		Level: hclog.LevelFromString("DEBUG"),
	})
	appLogger.Info("nbuild is initializing")

	srctree := graph.New(appLogger)

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
}
