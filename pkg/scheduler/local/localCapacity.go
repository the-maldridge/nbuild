package local

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/scheduler"
	"github.com/the-maldridge/nbuild/pkg/source"
)

func NewLocalCapacityProvider(l hclog.Logger, path string) *LocalCapacityProvider {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	x := LocalCapacityProvider{
		l:       l.Named("capacityProvider"),
		path:    absPath,
		ongoing: nil,
	}
	return &x
}

// Wrapper function for pkgCmd.Run()
func (c *LocalCapacityProvider) pkgRun(cmd *exec.Cmd) {
	output, err := cmd.CombinedOutput()
	c.ongoing = nil
	if err != nil {
		c.l.Warn("Error building pkg", "err", err)
	}
	c.l.Trace("Building package output", "output", string(output))
}

// Builds a package.
func (c *LocalCapacityProvider) DispatchBuild(b scheduler.Build) error {
	if c.ongoing != nil {
		return new(scheduler.NoCapacityError)
	}
	c.ongoing = &b

	// Git checkout
	repo := source.New(c.l)
	repo.SetBasepath(c.path)
	err := repo.Bootstrap()
	if err != nil {
		return err
	}
	_, err = repo.Checkout(b.Rev)
	if err != nil {
		return err
	}

	os.Chdir(c.path)
	c.l.Info("Binary-bootstrapping", "path", c.path, "spec", b.Spec)
	bootstrapCmd := exec.Command("./xbps-src", "binary-bootstrap", b.Spec.Host)
	bootstrapCmd.Dir = c.path
	err = bootstrapCmd.Run()
	if err != nil {
		c.l.Warn("Error running binary-bootstrap", "err", err)
		return err
	}

	c.l.Debug("Building package", "build", b, "path", c.path)
	args := []string{"pkg", b.Pkg}
	if !b.Spec.Native() {
		args = append(args, "-a", b.Spec.Target)
	}
	pkgCmd := exec.Command("./xbps-src", args...)
	pkgCmd.Dir = c.path
	go c.pkgRun(pkgCmd)
	os.Chdir("..")

	return nil
}

// Lists the ongoing build, if there is one.
func (c *LocalCapacityProvider) ListBuilds() ([]scheduler.Build, error) {
	if c.ongoing == nil {
		return nil, nil
	} else {
		return []scheduler.Build{*c.ongoing}, nil
	}
}
