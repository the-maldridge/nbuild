package local

import (
	"github.com/the-maldridge/nbuild/pkg/scheduler"

	"github.com/hashicorp/go-hclog"
)

// Local is a capacity provider that builds one build at a time locally.
type Local struct {
	l       hclog.Logger
	ongoing *scheduler.Build
	path    string
}
