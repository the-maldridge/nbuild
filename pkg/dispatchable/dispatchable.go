package dispatchable

import (
	"sync"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// NewDispatchFinder takes in a set of atoms and works out what can be
// built across them.
func NewDispatchFinder(opts ...Option) *DispatchFinder {
	x := new(DispatchFinder)
	x.AtomMu = new(sync.Mutex)
	for _, o := range opts {
		o(x)
	}
	return x
}

// IsDispatchable determines whether a specific package could be dispatched
// right now.
func (d *DispatchFinder) IsDispatchable(spec types.SpecTuple, p *types.Package) bool {
	hAtom := d.atoms[types.SpecTuple{spec.Host, spec.Host}]
	for hdep := range p.HostDepends {
		if _, ok := hAtom.Pkgs[hdep]; !ok {
			d.l.Warn("Host dependency cannot be found in atom", "hdep", hdep, "pkg", p)
			return false
		}
		if hAtom.Pkgs[hdep].Dirty || hAtom.Pkgs[hdep].Failed {
			return false
		}
	}

	tAtom := d.atoms[spec]
	for dep := range p.MakeDepends {
		if _, ok := tAtom.Pkgs[dep]; !ok {
			d.l.Warn("Dependency cannot be found in atom", "dep", dep, "pkg", p)
			return false
		}
		if tAtom.Pkgs[dep].Dirty || tAtom.Pkgs[dep].Failed {
			return false
		}
	}
	for dep := range p.Depends {
		if _, ok := tAtom.Pkgs[dep]; !ok {
			d.l.Warn("Dependency cannot be found in atom", "dep", dep, "pkg", p)
			return false
		}
		if tAtom.Pkgs[dep].Dirty || tAtom.Pkgs[dep].Failed {
			return false
		}
	}
	// If we get this far, all hostdeps, makedeps, deps are clean.
	return true
}

// ImmediatelyDispatchable returns a map of tuples -> packages that can be
// hypothetically dispatched right now.
// *Assumes graph is freshly Cleaned*. If not, will return packages that may
// have been made clean without graph knowing.
func (d *DispatchFinder) ImmediatelyDispatchable() map[types.SpecTuple][]*types.Package {
	dispatchable := make(map[types.SpecTuple][]*types.Package)
	d.AtomMu.Lock()
	defer d.AtomMu.Unlock()
	for spec, atom := range d.atoms {
		// We need the host atom as well to be able to do this
		if _, ok := d.atoms[types.SpecTuple{spec.Host, spec.Host}]; !ok {
			d.l.Warn("Unable to find dispatchable due to lack of host atom", "spec", spec)
			continue
		}
		dispatchable[spec] = make([]*types.Package, 0)
		for _, pkg := range atom.Pkgs {
			if !pkg.Failed && pkg.Dirty && d.IsDispatchable(spec, pkg) {
				dispatchable[spec] = append(dispatchable[spec], pkg)
			}
		}
	}
	return dispatchable
}
