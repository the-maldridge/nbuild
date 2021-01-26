package srcpkgs

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/go-hclog"
)

// NewTree returns a new blank tree with the logger configured
func NewTree(l hclog.Logger) *PkgTree {
	x := PkgTree{
		l:       l.Named("srcpkg"),
		Pkgs:    make(map[string]*Package),
		Virtual: make(map[string]string),
		seen:    make(map[string]struct{}),
	}
	return &x
}

// Import tries to walk a package tree and import the whole thing.
func (t *PkgTree) Import(b string) error {
	paths, _ := filepath.Glob(filepath.Join(b, "*"))
	pkgCount := 0

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
		if _, ok := t.Pkgs[pkgname]; ok {
			// Already loaded this one, continue on
			t.l.Debug("Already loaded package", "package", pkgname)
			continue
		}
		t.l.Debug("Loading Package", "package", pkgname)

		if _, err := t.LoadPackage(pkgname); err != nil {
			t.l.Warn("Error loading package", "error", err)
			continue
		}

		pkgCount++
	}
	t.l.Debug("Loaded packages", "count", pkgCount)
	return nil
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

// LoadPackage tries to get a package and insert it into the tree.
func (t *PkgTree) LoadPackage(name string) (*Package, error) {
	pp, ok := t.Pkgs[name]
	if ok {
		t.l.Trace("Already loaded package", "package", name)
		return pp, nil
	}

	if t.pkgExists(name) {
		return t.loadFromDisk(name)
	}

	if strings.HasPrefix(name, "virtual?") {
		name = t.Virtual[strings.ReplaceAll(name, "virtual?", "")]
		return t.LoadPackage(name)
	}

	if strings.ContainsAny(name, "<>=") {
		n, err := t.getpkgdepname(name)
		if err != nil {
			t.l.Trace("Error getpkgdepname", "error", err)
			return nil, err
		}
		return t.LoadPackage(n)
	}

	// last resort
	n, err := t.getpkgname(name)
	if err != nil {
		t.l.Trace("Error getpkgname", "error", err)
		return nil, err
	}
	return t.LoadPackage(n)
}

func (t *PkgTree) loadFromDisk(name string) (*Package, error) {
	p := Package{}
	dump, err := exec.Command("./xbps-src", "show-pkg-var-dump", name).Output()
	t.l.Trace("exec error", "error", err)
	if err != nil {
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
			if parts[1] == "" {
				continue
			}
			key = parts[0]
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
	rv, _ := strconv.Atoi(tokens["revision"])
	p.Revision = rv

	hostmakedepends, ok := tokens["hostmakedepends"]
	if ok {
		for _, hdp := range strings.Fields(hostmakedepends) {
			if hdp == name {
				continue
			}
			hdpp, err := t.LoadPackage(hdp)
			if err != nil {
				continue
			}
			p.HostDepends = append(p.HostDepends, hdpp)
		}
	}

	makedepends, ok := tokens["makedepends"]
	if ok {
		for _, mdp := range strings.Fields(makedepends) {
			if mdp == name {
				continue
			}
			mdpp, err := t.LoadPackage(mdp)
			if err != nil {
				continue
			}
			p.MakeDepends = append(p.MakeDepends, mdpp)
		}
	}

	depends, ok := tokens["depends"]
	if ok {
		for _, dp := range strings.Fields(depends) {
			if dp == name {
				continue
			}
			dpp, err := t.LoadPackage(dp)
			if err != nil {
				continue
			}
			p.Depends = append(p.Depends, dpp)
		}
	}

	t.l.Debug("Loaded Package", "package", p.Name)
	t.l.Trace("Loaded Package", "data", p)
	t.Pkgs[name] = &p
	return &p, nil
}

func (t *PkgTree) getpkgname(s string) (string, error) {
	dump, err := exec.Command("xbps-uhelper", "getpkgname", s).Output()
	if err != nil {
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
