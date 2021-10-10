package graph

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/repo"
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

	atom types.Atom
}

// Manager is a collection of graphs that all interact with the same
// git checkout.
type Manager struct {
	l        hclog.Logger
	cm       CheckoutManager
	graphs   map[string]*PkgGraph
	specs    []types.SpecTuple
	idx      *repo.IndexService
	basepath string
	rev      string

	storage storage.Storage
}

// CheckoutManager handles a git checkout
type CheckoutManager interface {
	SetBasepath(string)

	Bootstrap() error
	Fetch() error
	Checkout(string) ([]string, error)
	At() (string, error)
}
