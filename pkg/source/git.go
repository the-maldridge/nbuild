package source

import (
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

// Create a git repository at Path from URL
func (r *RepoMngr) Bootstrap() error {
	var err error
	if r.Path == "" {
		r.l.Warn("Error in repo manager, path must be set to bootstrap")
	}
	if r.Url == "" {
		r.l.Warn("Error in repo manager, url must be set to bootstrap")
	}
	r.Mu.Lock()
	defer r.Mu.Unlock()
	r.l.Debug("Cloning repository", "path", r.Path, "url", r.Url)
	// Don't do a shallow clone (Depth: BIG)
	r.repo, err = git.PlainClone(r.Path, false,
		&git.CloneOptions{URL: r.Url, Depth: 99999999})
	if err != nil {
		r.l.Trace("Error running PlainClone")
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
		r.l.Trace("Error getting old HEAD")
		return nil, err
	}
	oldCommit, err := r.repo.CommitObject(oldHead.Hash())
	if err != nil {
		r.l.Trace("Error getting old CommitObject")
		return nil, err
	}
	r.l.Debug("Attempting to checkout in git repository", "path", r.Path,
		"old", oldHead.Hash().String(), "new", commit)

	// Check we are not doing nothing
	if oldHead.Hash().String() == commit {
		r.l.Trace("Nothing changed in checkout")
		return make([]string, 0), nil
	}

	// Checkout the new commit
	worktree, err := r.repo.Worktree()
	if err != nil {
		r.l.Trace("Error getting worktree")
		return nil, err
	}
	newHash := gitPlumbing.NewHash(commit)
	err = worktree.Checkout(&git.CheckoutOptions{Hash: newHash, Force: true})

	// Diff the two commits
	newCommit, err := r.repo.CommitObject(newHash)
	if err != nil {
		r.l.Trace("Error getting new CommitObject")
		return nil, err
	}
	diff, err := newCommit.Patch(oldCommit)
	if err != nil {
		r.l.Trace("Error getting patch")
		return nil, err
	}
	diffFileStats := diff.Stats()
	r.l.Debug("Files were changed in checkout", "count", strconv.Itoa(len(diffFileStats)))
	changedFiles := make([]string, len(diffFileStats))
	for i := 0; i < len(diffFileStats); i++ {
		r.l.Trace("File was changed in checkout", "path", diffFileStats[i].Name)
		changedFiles[i] = diffFileStats[i].Name
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
