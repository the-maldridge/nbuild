package source

import (
	"sync"

	git "github.com/go-git/go-git/v5"
	"github.com/hashicorp/go-hclog"
)

// A RepoMngr manages the git side of a git repository.
type RepoMngr struct {
	l    hclog.Logger
	Path string
	Mu   *sync.Mutex
	repo *git.Repository
}
