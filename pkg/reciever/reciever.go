package reciever

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/go-hclog"
)

// NewReciever returns a reciever instance.
func NewReciever(l hclog.Logger) *Reciever {
	x := Reciever{
		l:         l.Named("reciever"),
		repoMutex: new(sync.Mutex),
	}

	return &x
}

// getArch gets the architecture from a filename.
func getArch(f string) string {
	noExt := strings.Split(strings.Split(path.Base(f), ".xbps")[0], ".")
	return noExt[len(noExt)-1]
}

// SetPath sets the path of the XBPS repository.
func (r *Reciever) SetPath(p string) {
	// If this fails, something is dreadfully wrong.
	// see scheduler/local.SetPath()
	r.path, _ = filepath.Abs(p)
}

// registerFile registers an XBPS package file into the index.
func (r *Reciever) registerFile(fPath string) error {
	architecture := getArch(fPath)
	cmd := exec.Command("xbps-rindex", "-a", fPath)
	cmd.Env = append(os.Environ(),
		"XBPS_TARGET_ARCH="+architecture,
	)
	r.repoMutex.Lock()
	defer r.repoMutex.Unlock()
	if err := cmd.Run(); err != nil {
		r.l.Warn("Unable to register package into index", "path", fPath, "arch", architecture, "err", err)
		return err
	}
	r.l.Trace("Added package into index", "path", fPath, "arch", architecture)
	return nil
}

// handleFile copies out a XBPS package file from HTTP out to a on-disk file.
func (r *Reciever) handleFile(fname string, repo string, data io.ReadCloser) error {
	// Do not check error, as it is a reader from HTTP so we don't care too much
	// if it dosen't close properly.
	defer data.Close()

	arch := getArch(fname)
	fPath := filepath.Join(r.path, arch, repo, fname)
	err := os.MkdirAll(path.Dir(fPath), 0755)
	if err != nil {
		r.l.Warn("Error creating directory", "path", path.Dir(fPath), "err", err)
		return err
	}
	out, err := os.Create(fPath)
	if err != nil {
		r.l.Warn("Error creating/opening file", "path", fPath, "err", err)
		return err
	}

	if _, err = io.Copy(out, data); err != nil {
		r.l.Warn("Error copying data into file", "path", fPath, "err", err)
		// If something went wrong copying, the error closing out is likely to
		// be the same.
		_ = out.Close()
		return err
	}
	if err = out.Close(); err != nil {
		r.l.Warn("Error closing out file", "path", fPath, "err", err)
		return err
	}
	r.l.Trace("Wrote file from HTTP", "path", fPath)

	if err = r.registerFile(fPath); err != nil {
		return err
	}
	return nil
}

// HTTPEntry provides the chi mountpoint for the reciever into the routing tree.
func (r *Reciever) HTTPEntry() chi.Router {
	rout := chi.NewRouter()
	rout.Put("/file", r.httpFile)
	return rout
}

// httpFile handles a file recieved via HTTP.
func (r *Reciever) httpFile(w http.ResponseWriter, req *http.Request) {
	err := r.handleFile(req.URL.Query().Get("fname"), req.URL.Query().Get("repo"), req.Body)
	if err != nil {
		r.httpJSONError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// jsonError returns a error as JSON.
func (r *Reciever) httpJSONError(w http.ResponseWriter, err error) {
	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusInternalServerError)
	out := struct {
		Error string
	}{
		Error: err.Error(),
	}
	w.Header().Set("Content-Type", "application/json")
	err = enc.Encode(out)
	if err != nil {
		r.l.Warn("Error encoding JSON error response")
	}
}
