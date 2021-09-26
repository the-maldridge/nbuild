package graph

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// PkgGraph contains a tree of packages
type PkgGraph struct {
	// Lock for source pkgs map
	SrcPkgsMutex *sync.Mutex

	l hclog.Logger

	basePath string
	parallelism int

	SrcPkgs map[string]*types.SrcPkg
	pkgs    map[string]*types.Package
	Virtual map[string]string

	seen map[string]struct{}

	// bad returned some errors, so we keep an eye on what the
	// error was and continue.
	bad map[string]string
}
