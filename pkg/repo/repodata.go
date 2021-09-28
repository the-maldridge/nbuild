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

	packages map[string]*types.Package
}

// NewIndexService creates an IndexService
func NewIndexService(l hclog.Logger) *IndexService {
	is := IndexService{
		l:        l.Named("IndexService"),
		packages: make(map[string]*types.Package),
	}
	return &is
}

// LoadIndex retrieves the index via http.
func (is *IndexService) LoadIndex(path string) error {
	var indexBytes []byte
	var err error

	switch {
	case strings.HasPrefix(path, "http"):
		indexBytes, err = is.fetchHTTP(path)
	case strings.HasPrefix(path, "file"):
		indexBytes, err = is.fetchFile(path)
	default:
		err = errors.New("Unknown repodata scheme")
		is.l.Error("Repodata scheme must be either file or http(s)")
	}
	if err != nil {
		return err
	}

	if err := is.parseRepoData(indexBytes); err != nil {
		return err
	}

	return nil
}

// PkgCount is a quick check of how many packages this index knows
// about.
func (is *IndexService) PkgCount() int {
	return len(is.packages)
}

// GetPackage returns a single package from the index.
func (is *IndexService) GetPackage(name string) (*types.Package, error) {
	pkg, ok := is.packages[name]
	if !ok {
		return nil, errors.New("NoSuchPackage")
	}
	return pkg, nil
}

func (is *IndexService) fetchHTTP(path string) ([]byte, error) {
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

func (is *IndexService) fetchFile(path string) ([]byte, error) {
	return os.ReadFile(strings.TrimPrefix(path, "file://"))
}

// Heavily inspired and simplified from the generalized reader in
// Duncaen's go-xbps project.
func (is *IndexService) parseRepoData(indexBytes []byte) error {
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
		if err := dec.Decode(&is.packages); err != nil {
			return err
		}
		return nil
	}
	return nil
}
