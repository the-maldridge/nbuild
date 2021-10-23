package scheduler

// Equal performs a deep check into a build to determine if it is
// equal to another value.
func (b Build) Equal(c Build) bool {
	return b.Spec.Host == c.Spec.Host &&
		b.Spec.Target == c.Spec.Target &&
		b.Pkg == c.Pkg &&
		b.Rev == c.Rev
}

// ToMap flattens out a build into a map of key/value pairs.
func (b Build) ToMap() map[string]string {
	return map[string]string{
		"host_arch":   b.Spec.Host,
		"target_arch": b.Spec.Target,
		"package":     b.Pkg,
		"revision":    b.Rev,
	}
}
