package dispatchable

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// DispatchFinder walks the graph and works out what can actually be
// built right now.
type DispatchFinder struct {
	l      hclog.Logger
	AtomMu *sync.Mutex

	atoms map[types.SpecTuple]types.Atom
}

// Option allows various config values to be passed in using a slice
// of option.
type Option func(*DispatchFinder)
