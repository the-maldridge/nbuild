package srcpkgs

import (
	"sync"

	"github.com/hashicorp/go-hclog"
)

// A SrcPkg is a package as read in from the filesystem.  These have
// to later be mapped onto the graph of Package structs as defined
// below.
type SrcPkg struct {
	Name        string
	Version     string
	Revision    int
	HostDepends map[string]struct{}
	MakeDepends map[string]struct{}
	Depends     map[string]struct{}
}

// Package represents a single package in the srcpkgs collection.
type Package struct {
	Name        string
	Version     string
	Revision    int
	HostDepends []*Package
	MakeDepends []*Package
	Depends     []*Package
}

// PkgTree contains a tree of packages
type PkgTree struct {
	// Lock for source pkgs map
	SrcPkgsMutex *sync.Mutex

	l hclog.Logger

	SrcPkgs map[string]*SrcPkg
	Pkgs    map[string]*Package
	Virtual map[string]string

	seen map[string]struct{}

	// bad returned some errors, so we keep an eye on what the
	// error was and continue.
	bad map[string]string
}
