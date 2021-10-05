package graph

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/storage"
	"github.com/the-maldridge/nbuild/pkg/types"
)

// PkgGraph contains a tree of packages
type PkgGraph struct {
	// Lock for source pkgs map
	PkgsMutex *sync.Mutex

	// Lock for auxiliary maps
	AuxMutex *sync.Mutex

	l hclog.Logger

	basePath    string
	parallelism int

	atom Atom
}

// Atom is a storage struct that contains all the serializable data
// for a single arch graph.
type Atom struct {
	Pkgs    map[string]*types.Package
	Virtual map[string]string

	// bad returned some errors, so we keep an eye on what the
	// error was and continue.
	Bad map[string]string

	// These keep track of what the archs this graph is rendered
	// from are.
	Spec SpecTuple

	// Rev stores the git revision of the PkgGraph for later so
	// that we can tell if the graph needs to be reloaded.
	Rev string
}

// Manager is a collection of graphs that all interact with the same
// git checkout.
type Manager struct {
	l        hclog.Logger
	cm       CheckoutManager
	graphs   map[string]*PkgGraph
	specs    []SpecTuple
	basepath string
	rev      string

	storage storage.Storage
}

// A SpecTuple is a listing of the host and target arch.
type SpecTuple struct {
	Host   string
	Target string
}

// CheckoutManager handles a git checkout
type CheckoutManager interface {
	SetBasepath(string)

	Bootstrap() error
	Fetch() error
	Checkout(string) ([]string, error)
	At() (string, error)
}
