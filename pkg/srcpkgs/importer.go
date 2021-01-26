package srcpkgs

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
)

// NewTree returns a new blank tree with the logger configured
func NewTree(l hclog.Logger) *PkgTree {
	x := PkgTree{
		l:            l.Named("srcpkg"),
		SrcPkgsMutex: new(sync.Mutex),
		SrcPkgs:      make(map[string]*SrcPkg),
		Pkgs:         make(map[string]*Package),
		Virtual:      make(map[string]string),
		seen:         make(map[string]struct{}),
		bad:          make(map[string]string),
	}
	return &x
}

// Import tries to read every srcpkg from disk and then converge the
// graph.
func (t *PkgTree) Import(b string, parallelism int) error {
	paths, _ := filepath.Glob(filepath.Join(b, "*"))
	pkgCount := 0

	loadCh := make(chan string, 200)
	wg := new(sync.WaitGroup)

	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			id := i
			for {
				p, more := <-loadCh
				if !more {
					t.l.Debug("Importer shutting down", "ID", id)
					wg.Done()
					return
				}
				t.l.Debug("Loading Package", "package", p)
				spkg, err := t.loadFromDisk(p)
				if err != nil {
					t.l.Warn("Error loading package", "package", p, "error", err)
					continue
				}
				t.SrcPkgsMutex.Lock()
				t.SrcPkgs[p] = spkg

				pkg := Package{}
				pkg.Name = p
				pkg.Version = spkg.Version
				pkg.Revision = spkg.Revision
				t.Pkgs[p] = &pkg

				pkgCount++
				t.SrcPkgsMutex.Unlock()
			}
		}()
	}

	for _, p := range paths {
		pinfo, err := os.Lstat(p)
		if err != nil {
			t.l.Warn("Error with path", "error", err, "path", p)
			continue
		}

		if !pinfo.IsDir() {
			// We only care about the directories
			continue
		}
		t.l.Trace("PathInfo", "info", pinfo)
		pkgname := filepath.Base(p)
		if !t.pkgExists(pkgname) {
			continue
		}
		loadCh <- pkgname
	}
	close(loadCh)
	wg.Wait()
	t.l.Debug("Loaded packages", "count", pkgCount)
	t.l.Debug("Bad pkgs", "pkgs", t.bad)
	return nil
}

// ResolveGraph converst SrcPkgs to Pkgs and hooks up the dependency
// lists.  It is responsible for constructing the graph that
// ultimately is used to drive package builds.
func (t *PkgTree) ResolveGraph() {
	for p := range t.Pkgs {
		t.l.Debug("Resolving package", "pkg", p)
		sp := t.SrcPkgs[p]

		hd := make([]*Package, len(sp.HostDepends))
		i := 0
		for d := range sp.HostDepends {
			t.l.Trace("Package hostmakedepends", "pkg", p, "hostmakedepends", d)
			hd[i] = t.Pkgs[d]
			i++
		}
		t.Pkgs[p].HostDepends = hd

		md := make([]*Package, len(sp.MakeDepends))
		i = 0
		for d := range sp.MakeDepends {
			t.l.Trace("Package makedepends", "pkg", p, "makedepends", d)
			md[i] = t.Pkgs[d]
			i++
		}
		t.Pkgs[p].MakeDepends = md

		rd := make([]*Package, len(sp.Depends))
		i = 0
		for d := range sp.Depends {
			t.l.Trace("Package depends", "pkg", p, "depends", d)
			rd[i] = t.Pkgs[d]
			i++
		}
		t.Pkgs[p].Depends = rd
	}
}

// LoadVirtual loads the virtual packages map from the location 'loc'
func (t *PkgTree) LoadVirtual(loc string) error {
	f, err := os.Open(loc)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		l := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(l, "#") || l == "" {
			continue
		}
		flds := strings.Fields(l)
		t.Virtual[flds[0]] = flds[1]
	}
	return nil
}

// ResolvePackage tries to return a soure package that is referenced
// by any of the means that are valid in xbps-src
func (t *PkgTree) ResolvePackage(name string) (*SrcPkg, error) {
	pp, ok := t.SrcPkgs[name]
	if ok {
		t.l.Trace("Already loaded package", "package", name)
		return pp, nil
	}

	if strings.HasPrefix(name, "virtual?") {
		name = t.Virtual[strings.ReplaceAll(name, "virtual?", "")]
		return t.ResolvePackage(name)
	}

	if strings.ContainsAny(name, "<>=") {
		n, err := t.getpkgdepname(name)
		if err != nil {
			t.l.Trace("Error getpkgdepname", "error", err)
			return nil, err
		}
		return t.ResolvePackage(n)
	}

	// last resort
	n, err := t.getpkgname(name)
	if err != nil {
		t.l.Trace("Error getpkgname", "error", err)
		return nil, err
	}
	return t.ResolvePackage(n)
}

func (t *PkgTree) loadFromDisk(name string) (*SrcPkg, error) {
	p := SrcPkg{}
	dump, err := exec.Command("./xbps-src", "dbulk-dump", name).Output()
	t.l.Trace("exec error", "error", err)
	var exitError *exec.ExitError
	if err != nil && errors.As(err, &exitError) {
		stderr := string(exitError.Stderr)
		t.bad[name] = stderr
		return nil, err
	} else if err != nil {
		return nil, err
	}

	r := bytes.NewReader(dump)
	s := bufio.NewScanner(r)

	var key string
	tokens := make(map[string]string)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}

		// Line contains a colon, so must be a key
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			key = parts[0]
			if parts[1] == "" {
				continue
			}
			tokens[key] = parts[1]
		} else {
			// Does not contain a colon, is continuation of last key
			tokens[key] += (" " + line)
		}

		t.l.Trace("Parsing Package", "package", name, "line", line)
	}
	t.l.Trace("Parsed Tokens", "tokens", tokens)

	p.Name = strings.TrimSpace(tokens["pkgname"])
	p.Version = tokens["version"]
	rev, err := strconv.Atoi(tokens["revision"])
	if err != nil {
		rev = 0
	}
	p.Revision = rev

	hmd := strings.Fields(tokens["hostmakedepends"])
	p.HostDepends = make(map[string]struct{}, len(hmd))
	for _, h := range hmd {
		p.HostDepends[h] = struct{}{}
	}

	md := strings.Fields(tokens["makedepends"])
	p.MakeDepends = make(map[string]struct{}, len(md))
	for _, m := range md {
		p.MakeDepends[m] = struct{}{}
	}

	d := strings.Fields(tokens["depends"])
	p.Depends = make(map[string]struct{}, len(d))
	for _, rd := range d {
		p.Depends[rd] = struct{}{}
	}

	t.l.Trace("Loaded Package", "data", p)
	t.SrcPkgsMutex.Lock()
	t.SrcPkgs[name] = &p
	t.SrcPkgsMutex.Unlock()
	return &p, nil
}

func (t *PkgTree) getpkgname(s string) (string, error) {
	dump, err := exec.Command("xbps-uhelper", "getpkgname", s).Output()
	if err != nil {
		t.l.Trace("Failed to call command", "command", "xbps-uhelper getpkgname "+s)
		return "", err
	}
	return string(dump), nil
}

func (t *PkgTree) getpkgdepname(s string) (string, error) {
	dump, err := exec.Command("xbps-uhelper", "getpkgdepname", s).Output()
	if err != nil {
		return "", err
	}
	return string(dump), nil
}

func (t *PkgTree) pkgExists(name string) bool {
	_, err := os.Lstat(filepath.Join("srcpkgs", name, "template"))
	return !os.IsNotExist(err)
}