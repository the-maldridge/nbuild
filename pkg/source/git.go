package source

import (
	"path/filepath"
	"strconv"
	"sync"

	git "github.com/go-git/go-git/v5"
	gitPlumbing "github.com/go-git/go-git/v5/plumbing"
	"github.com/hashicorp/go-hclog"
)

// New creates a new instance of RepoMngr
func New(l hclog.Logger) *RepoMngr {
	x := RepoMngr{
		l:  l.Named("git"),
		Mu: new(sync.Mutex),
	}
	return &x
}

// SetBasepath sets up the path for the repo to be written to.
func (r *RepoMngr) SetBasepath(p string) {
	r.Path = p
}

// Create a git repository at Path from URL
func (r *RepoMngr) Bootstrap() error {
	var err error
	if r.Path == "" {
		r.l.Warn("Error in repo manager, path must be set to bootstrap")
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.l.Info("Opening repository", "path", r.Path)
	r.repo, err = git.PlainOpen(r.Path)
	if err != nil {
		r.l.Warn("Error opening repository", "path", r.Path)
		return err
	}
	return nil
}

// Get the current HEAD hash
func (r *RepoMngr) At() (string, error) {
	var err error
	head, err := r.repo.Head()
	if err != nil {
		r.l.Trace("Error getting HEAD")
		return "", err
	}
	return head.Hash().String(), nil
}

// Checkout a particular revision
func (r *RepoMngr) Checkout(commit string) ([]string, error) {
	if r.repo == nil {
		r.l.Warn("Error in repo manager, repo must be bootstrapped to checkout")
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()

	// Find the old commit
	oldHead, err := r.repo.Head()
	if err != nil {
		r.l.Warn("Error getting old HEAD", "err", err, "path", r.Path)
		return nil, err
	}
	oldCommit, err := r.repo.CommitObject(oldHead.Hash())
	if err != nil {
		r.l.Warn("Error getting old CommitObject", "err", err, "path", r.Path)
		return nil, err
	}
	r.l.Info("Attempting to checkout in git repository", "path", r.Path,
		"old", oldHead.Hash().String(), "new", commit)

	// Check we are not doing nothing
	if oldHead.Hash().String() == commit {
		r.l.Trace("Nothing changed in checkout")
		return make([]string, 0), nil
	}

	// Checkout the new commit
	worktree, err := r.repo.Worktree()
	if err != nil {
		r.l.Warn("Error getting worktree", "err", err, "path", r.Path)
		return nil, err
	}
	newHash := gitPlumbing.NewHash(commit)
	err = worktree.Checkout(&git.CheckoutOptions{Hash: newHash, Force: true})

	// Diff the two commits
	newCommit, err := r.repo.CommitObject(newHash)
	if err != nil {
		r.l.Warn("Error getting new CommitObject", "err", err, "path", r.Path)
		return nil, err
	}
	diff, err := newCommit.Patch(oldCommit)
	if err != nil {
		r.l.Warn("Error getting patch", "err", err, "path", r.Path)
		return nil, err
	}
	diffFileStats := diff.Stats()
	r.l.Debug("Files were changed in checkout", "count", strconv.Itoa(len(diffFileStats)))
	changedFiles := make([]string, len(diffFileStats))
	for i := 0; i < len(diffFileStats); i++ {
		r.l.Trace("File was changed in checkout", "path", diffFileStats[i].Name)
		changedFiles[i] = filepath.Join(r.Path, diffFileStats[i].Name)
	}

	return changedFiles, nil
}

// Fetch origin
func (r *RepoMngr) Fetch() error {
	if r.repo == nil {
		r.l.Warn("Error in repo manager, repo must be bootstrapped to fetch")
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.l.Debug("Fetching origin for git repository", "path", r.Path)
	err := r.repo.Fetch(&git.FetchOptions{RemoteName: "origin"})
	if err != nil {
		r.l.Trace("Error fetching")
		return err
	}
	return nil
}
