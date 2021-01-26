package main

import (
	"encoding/json"
	"os"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/srcpkgs"
)

func main() {
	appLogger := hclog.New(&hclog.LoggerOptions{
		Name:  "nbuild",
		Level: hclog.LevelFromString("TRACE"),
	})

	srctree := srcpkgs.NewTree(appLogger)

	if err := srctree.LoadVirtual("etc/defaults.virtual"); err != nil {
		return
	}

	if err := srctree.Import("srcpkgs", 100); err != nil {
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
