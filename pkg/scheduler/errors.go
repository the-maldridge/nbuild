package scheduler

// ErrNoCapacity is returned when a provider refuses to dispatch a
// build due to capacity exhaustion.
type ErrNoCapacity struct{}

func (e ErrNoCapacity) Error() string {
	return "insufficient capacity"
}

// ErrUnknownCapacityProvider is returned when a capacity factory is
// requested that has not been registered.
type ErrUnknownCapacityProvider struct {
	attempted string
}

// NewErrUnknownCapacityProvider returns a new error specialized to
// the attempted provider.
func NewErrUnknownCapacityProvider(s string) ErrUnknownCapacityProvider {
	return ErrUnknownCapacityProvider{s}
}

func (e ErrUnknownCapacityProvider) Error() string {
	return "no factory with name " + e.attempted + " exists"
}
