package nomad

import (
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/nomad/api"

	"github.com/the-maldridge/nbuild/pkg/scheduler"
	"github.com/the-maldridge/nbuild/pkg/types"
)

type nomadProvider struct {
	l hclog.Logger
	c *api.Client

	slots map[string]int
}

func init() {
	scheduler.RegisterInitCallback(cb)
}

func cb() {
	scheduler.RegisterCapacityFactory("nomad", New)
}

// New returns a wrapper around a nomad client that implements the
// scheduler's CapacityProvider interface.
func New(l hclog.Logger) (scheduler.CapacityProvider, error) {
	c, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, err
	}

	x := &nomadProvider{
		l:     l.Named("nomad"),
		c:     c,
		slots: make(map[string]int),
	}
	return x, nil
}

func (n *nomadProvider) DispatchBuild(b scheduler.Build) error {
	r, err := n.runningBuilds()
	if err != nil {
		return err
	}
	if r[b.Spec]+1 > n.slots[b.Spec.String()] {
		return new(scheduler.ErrNoCapacity)
	}

	res, _, err := n.c.Jobs().Dispatch("xbps-src", n.mergeCallbacks(b.ToMap()), nil, nil)
	if err != nil {
		n.l.Warn("Nomad error", "error", err)
		return err
	}
	n.l.Debug("Dispatched job", "spec", b.Spec, "pkg", b.Pkg, "eval", res.EvalID, "jid", res.DispatchedJobID)
	return nil
}

func (n *nomadProvider) ListBuilds() ([]scheduler.Build, error) {
	qopts := &api.QueryOptions{
		Prefix: "xbps-src/dispatch-",
	}
	jobs, _, err := n.c.Jobs().List(qopts)
	if err != nil {
		return nil, err
	}
	running := []string{}
	for _, job := range jobs {
		if job.Type != "batch" || (job.Status != "running" && job.Status != "pending") {
			continue
		}
		running = append(running, job.ID)
		n.l.Trace("Searched Jobs", "job", job)
	}
	builds := make([]scheduler.Build, len(running))
	for i, job := range running {
		job, _, err := n.c.Jobs().Info(job, nil)
		if err != nil {
			continue
		}
		builds[i].Spec = types.NewSpecTuple(job.Meta["host_arch"], job.Meta["target_arch"])
		builds[i].Pkg = job.Meta["package"]
		builds[i].Rev = job.Meta["revision"]
		n.l.Trace("Found running Build", "build", builds[i])
	}
	return builds, nil
}

func (n *nomadProvider) SetSlots(s map[string]int) {
	n.slots = s
}

func (n *nomadProvider) runningBuilds() (map[types.SpecTuple]int, error) {
	cap := make(map[types.SpecTuple]int)

	builds, err := n.ListBuilds()
	if err != nil {
		return nil, new(scheduler.ErrNoCapacity)
	}

	for _, b := range builds {
		cap[b.Spec]++
	}
	return cap, nil
}

func (n *nomadProvider) mergeCallbacks(m map[string]string) map[string]string {
	m["callback_done"] = "http://localhost"
	m["callback_fail"] = "http://localhost"
	return m
}
