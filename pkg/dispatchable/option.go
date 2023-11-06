package dispatchable

import (
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// WithLogger sets up the parent logger for the dispatcher.
func WithLogger(l hclog.Logger) Option {
	return func(df *DispatchFinder) {
		df.l = l.Named("Dispatch")
	}
}

// WithAtoms sets up the slice of atoms making a copy of each for the
// dispatcher's use.
func WithAtoms(atoms []types.Atom) Option {
	mAtoms := make(map[types.SpecTuple]types.Atom)
	for _, atom := range atoms {
		mAtoms[atom.Spec] = atom
	}

	return func(df *DispatchFinder) {
		df.atoms = mAtoms
	}
}
