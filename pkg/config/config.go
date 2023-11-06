package config

import (
	"encoding/json"
	"os"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// NewConfig returns a config object with default structures
// initialized.  The config can be loaded from other sources to
// override the defaults.
func NewConfig() *Config {
	return &Config{
		Specs: []types.SpecTuple{{"x86_64", "x86_64"}},
		RepoDataURLs: map[string]map[string]string{
			"x86_64": {
				"main":    "http://repo-fastly.voidlinux.org/current/x86_64-repodata",
				"nonfree": "http://repo-fastly.voidlinux.org/current/nonfree/x86_64-repodata",
				"local":   "file://void-packages/hostdir/binpkgs/x86_64-repodata",
			},
		},
		CapacityProvider: "local",
		BuildSlots: map[string]int{
			"x86_64:x86_64": 1,
		},
		RepoPath: "my-repo",
	}
}

// LoadFromFile does as the name suggests, and loads the config from a
// file
func (c *Config) LoadFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	return dec.Decode(c)
}
