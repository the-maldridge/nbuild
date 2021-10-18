package scheduler

func (b *Build) Equal(c *Build) bool {
	return b.Spec.Host == c.Spec.Host &&
		b.Spec.Target == c.Spec.Target &&
		b.Pkg == c.Pkg &&
		b.Rev == c.Rev
}
