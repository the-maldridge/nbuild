package config

import (
	"github.com/the-maldridge/nbuild/pkg/types"
)

// Config represents the complete application configuration that
// nbuild supports.
type Config struct {
	Specs []types.SpecTuple
	RepoDataURLs map[string]map[string]string
}
