package types

// A Package is a buildable unit within the source packages
// collection.
type Package struct {
	Name        string
	Dirty       bool
	Failed      bool
	Version     string `plist:"pkgver"`
	HostDepends map[string]struct{}
	MakeDepends map[string]struct{}
	Depends     map[string]struct{}
}
