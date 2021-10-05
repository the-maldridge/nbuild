package storage

import (
	"errors"

	"github.com/hashicorp/go-hclog"
)

var (
	log hclog.Logger

	initcallbacks []func()

	factories map[string]Factory
)

// A Factory creates a store instance that can be served by the web
// package.
type Factory func(hclog.Logger) (Storage, error)

func init() {
	factories = make(map[string]Factory)
	log = hclog.L()
}

// SetLogger injects a logger into this package to allow setting up a
// logger tree.
func SetLogger(l hclog.Logger) {
	log = l
}

// RegisterFactory registers a factory to the list of available state stores
// that can be used.
func RegisterFactory(s string, f Factory) {
	if _, exists := factories[s]; exists {
		log.Warn("Store name collision", "store", s)
		return
	}
	factories[s] = f
	log.Info("Registered store", "store", s)
}

// RegisterCallback provides a mechanism for early registration of a
// function to be called during initialization.  This allows the
// actual factories to be registered later once config parsing has
// happened, logging is configured, and other early-init tasks are
// complete.
func RegisterCallback(f func()) {
	initcallbacks = append(initcallbacks, f)
}

// DoCallbacks is used to invoke all callbacks and perform phase one
// setup which will register the handlers to the map of factories.
func DoCallbacks() {
	for _, cb := range initcallbacks {
		cb()
	}
}

// Initialize attempts to initialize the given store and returns
// either a ready to use store or an error.
func Initialize(s string) (Storage, error) {
	f, ok := factories[s]
	if !ok {
		log.Error("Non-existant factory requested", "factory", s)
		return nil, errors.New("no factory exists with that name")
	}
	return f(log)
}
