package local

import (
	"github.com/the-maldridge/nbuild/pkg/scheduler"

	"github.com/hashicorp/go-hclog"
)

// LocalCapacityProvider is a capacity provider that builds one build at a time locally.
type LocalCapacityProvider struct {
	l       hclog.Logger
	ongoing *scheduler.Build
	path    string
	slots   int
}
