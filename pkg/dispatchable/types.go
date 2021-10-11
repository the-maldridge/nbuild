package dispatchable

import (
	"sync"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// DispatchFinder
type DispatchFinder struct {
	l      hclog.Logger
	AtomMu *sync.Mutex

	atoms map[types.SpecTuple]types.Atom
}
