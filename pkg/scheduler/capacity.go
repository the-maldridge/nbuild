package scheduler

import (
	"github.com/hashicorp/go-hclog"
)

var (
	log hclog.Logger

	initcallbacks []func()

	factories map[string]CapacityFactory
)

// A CapacityFactory is a constructor of a capacity plugin.  It takes
// a single logger which should be used to write out early init
// issues, and provide more information.  Additional configuration
// values should be derived from the config package.
type CapacityFactory func(l hclog.Logger) (CapacityProvider, error)

func init() {
	factories = make(map[string]CapacityFactory)
	log = hclog.L()
}

// SetLogger injects a logger into this package to allow setting up a
// logger tree.
func SetLogger(l hclog.Logger) {
	log = l.Named("capacity")
}

// RegisterInitCallback allows a sub pkg to defer initialization until
// after certain very early init has happened such as loading config
// files and configuring loggers.
func RegisterInitCallback(f func()) {
	initcallbacks = append(initcallbacks, f)
}

// DoCallbacks is used to invoke all callbacks and perform phase one
// setup which will register the handlers to the map of factories.
func DoCallbacks() {
	for _, cb := range initcallbacks {
		cb()
	}
}

// RegisterCapacityFactory blindly stores the factory at the given
// name.  This is relatively safe since all the factories are enabled
// at build time.  Were we to support additional factories externally,
// we would want to perform some checking here to determine if the
// factory name has already been used.
func RegisterCapacityFactory(name string, f CapacityFactory) {
	factories[name] = f
	log.Info("Registered capacity provider", "provider", name)
}

// ConstructCapacityProvider attempts to initialize the requested
// capacity provider using the provided logger.
func ConstructCapacityProvider(name string) (CapacityProvider, error) {
	f, ok := factories[name]
	if !ok {
		log.Warn("Tried to initialize with bogus provider name", "name", name)
		return nil, NewErrUnknownCapacityProvider(name)
	}
	return f(log)
}
