package local

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/scheduler"
	"github.com/the-maldridge/nbuild/pkg/source"
)

func init() {
	scheduler.RegisterInitCallback(cb)
}

func cb() {
	scheduler.RegisterCapacityFactory("local", New)
}

// New returns a local capacity provider that operates on a single
// directory on the local host.  This provider is not really intended
// for production use and exists more to make testing the rest of the
// system easier.
func New(l hclog.Logger) (scheduler.CapacityProvider, error) {
	x := Local{
		l:       l.Named("capacityProvider"),
		path:    "local-checkout",
		ongoing: nil,
	}
	x.path, _ = filepath.Abs(x.path)
	return &x, nil
}

// SetSlots is used to setup the capacity for multi-process builders.
// This builder has a hard-coded capacity of one and so this does
// nothing.
func (c *Local) SetSlots(map[string]int) {}

// SetPath allows overriding the default path to the checkout, which
// is "local-capcity" in the current working directory.
func (c *Local) SetPath(p string) {
	// This can only error out if the underlying call to GetCwd
	// fails, which typically can only fail if the dir is gone,
	// and if you're removing directories on a running system then
	// you're in a seriously unsupported workflow already.
	c.path, _ = filepath.Abs(p)
}

// Wrapper function for pkgCmd.Run()
func (c *Local) pkgRun(cmd *exec.Cmd) {
	output, err := cmd.CombinedOutput()
	c.ongoing = nil
	if err != nil {
		c.l.Warn("Error building pkg", "err", err)
	}
	c.l.Trace("Building package output", "output", string(output))
}

// DispatchBuild attempts to spin off a build if no build is currently
// working.  This implicitly causes the capacity provider to have a
// capacity of 1 since the local git checkout can be at only one
// revision at a time.
func (c *Local) DispatchBuild(b scheduler.Build) error {
	if c.ongoing != nil {
		return new(scheduler.ErrNoCapacity)
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

// ListBuilds returns the currently in progress build, if one exists.
func (c *Local) ListBuilds() ([]scheduler.Build, error) {
	if c.ongoing == nil {
		return nil, nil
	}
	return []scheduler.Build{*c.ongoing}, nil
}
