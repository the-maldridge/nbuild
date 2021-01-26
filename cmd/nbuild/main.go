package main

import (
	"encoding/gob"
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

	if err := srctree.Import("srcpkgs"); err != nil {
		return
	}

	f, _ := os.Create("state.gob")
	defer f.Close()

	enc := gob.NewEncoder(f)

	if err := enc.Encode(srctree.Pkgs); err != nil {
		appLogger.Error("Error marshalling", "error", err)
		return
	}
}
