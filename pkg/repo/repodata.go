package repo

import (
	"archive/tar"
	"bytes"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/klauspost/compress/zstd"
	"howett.net/plist"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// IndexService is a wrapper around a lot of functions that
// interrogate repodata.
type IndexService struct {
	l hclog.Logger

	indicies map[string]*Index
}

// Index is an architecture specific index.
type Index struct {
	l hclog.Logger

	Arch      string
	Repodatas map[string]string
	repos     map[string]map[string]*types.Package
}

// NewIndexService creates an IndexService
func NewIndexService(l hclog.Logger) *IndexService {
	is := IndexService{
		l:        l.Named("IndexService"),
		indicies: make(map[string]*Index),
	}
	return &is
}

// LoadIndex retrieves the index via http.
func (is *IndexService) LoadIndex(arch, repo, path string) error {
	idx, ok := is.indicies[arch]
	if !ok {
		is.indicies[arch] = &Index{
			l:         is.l.Named(arch),
			Arch:      arch,
			Repodatas: map[string]string{repo: path},
			repos:     make(map[string]map[string]*types.Package),
		}
		idx = is.indicies[arch]
	}

	if _, ok := idx.Repodatas[repo]; !ok {
		idx.Repodatas[repo] = path
	}

	return idx.Load(repo, path)
}

// ReloadArch requests the specific arch to reload.
func (is *IndexService) ReloadArch(arch string) error {
	idx, ok := is.indicies[arch]
	if !ok {
		return errors.New("arch is unknown")
	}
	return idx.ReloadAll()
}

// GetPackage returns a single package from a single arch if it is
// known.
func (is *IndexService) GetPackage(arch, pkg string) (*types.Package, error) {
	idx, ok := is.indicies[arch]
	if !ok {
		return nil, errors.New("arch is unknown")
	}
	return idx.GetPackage(pkg)
}

// Load loads or reloads a single index from a file that is either on disk or remote.
func (i *Index) Load(repo, path string) error {
	var indexBytes []byte
	var err error

	switch {
	case strings.HasPrefix(path, "http"):
		indexBytes, err = i.fetchHTTP(path)
	case strings.HasPrefix(path, "file"):
		indexBytes, err = i.fetchFile(path)
	default:
		err = errors.New("unknown repodata scheme")
		i.l.Error("Repodata scheme must be either file or http(s)")
	}
	if err != nil {
		i.l.Warn("Error loading arch", "error", err)
		return err
	}

	return i.parseRepoData(repo, indexBytes)
}

// ReloadAll retrieves and re-loads all configured repodatas.
func (i *Index) ReloadAll() error {
	for repo, path := range i.Repodatas {
		i.Load(repo, path)
	}
	return nil
}

func (i *Index) fetchHTTP(path string) ([]byte, error) {
	resp, err := http.Get(path)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func (i *Index) fetchFile(path string) ([]byte, error) {
	return os.ReadFile(strings.TrimPrefix(path, "file://"))
}

// GetPackage returns a single package from the index.
func (i *Index) GetPackage(name string) (*types.Package, error) {
	for _, packages := range i.repos {
		pkg, ok := packages[name]
		if !ok {
			continue
		}
		return pkg, nil
	}
	return nil, errors.New("NoSuchPackage")
}

// Heavily inspired and simplified from the generalized reader in
// Duncaen's go-xbps project.
func (i *Index) parseRepoData(repo string, indexBytes []byte) error {
	i.l.Debug("Parsing repodata", "repo", repo)
	ibr := bytes.NewReader(indexBytes)

	// If we switch package formats again we'll need to add some
	// logic here to know what is being loaded.
	d, err := zstd.NewReader(ibr)
	if err != nil {
		return err
	}
	defer d.Close()

	tarchive := tar.NewReader(d)

	// Iterate throught the tar inside the zstd file and pick out
	// the index list.  This contains the package graph that we're
	// interested in.
	for {
		header, err := tarchive.Next()
		switch err {
		case nil:
		case io.EOF:
			return nil
		default:
			return err
		}

		if header.Name != "index.plist" {
			continue
		}

		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(tarchive); err != nil {
			return err
		}
		rs := bytes.NewReader(buf.Bytes())
		dec := plist.NewDecoder(rs)
		pkgs := make(map[string]*types.Package)
		if err := dec.Decode(pkgs); err != nil {
			return err
		}
		i.repos[repo] = pkgs
		return nil
	}
	return nil
}
