package types

// A SrcPkg is a package as read in from the filesystem.  These have
// to later be mapped onto the graph of Package structs as defined
// below.
type SrcPkg struct {
	Name        string
	Dirty       bool
	Failed      bool
	Version     string
	HostDepends map[string]struct{}
	MakeDepends map[string]struct{}
	Depends     map[string]struct{}
}

// Package represents a single package in the srcpkgs collection.
type Package struct {
	Name        string
	Version     string `plist:"pkgver"`
	HostDepends []*Package
	MakeDepends []*Package
	Depends     []*Package
}
