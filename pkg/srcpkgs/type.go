package srcpkgs

import (
	"github.com/hashicorp/go-hclog"
)

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
	l hclog.Logger

	Pkgs    map[string]*Package
	Virtual map[string]string

	seen map[string]struct{}
}
