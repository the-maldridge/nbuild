package graph

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// NewAPIClient creates a new API client.
func NewAPIClient(l hclog.Logger, url string) (*APIClient, error) {
	x := APIClient{
		l:       l.Named("client"),
		hClient: &http.Client{Timeout: 30 * time.Second},
		url: url,
	}
	if x.url == "" {
		x.l.Warn("URL must not be empty!")
		return nil, errors.New("url must be set")
	}
	return &x, nil
}

// General function to recieve a response
func (c *APIClient) do(endpoint string, method string) (string, error) {
	var resp *http.Response
	var err error
	fullURL := c.url + endpoint
	switch method {
	case "GET":
		resp, err = c.hClient.Get(fullURL)
	case "POST":
		resp, err = c.hClient.Post(fullURL, "application/json", bytes.NewBuffer([]byte("{}")))
	default:
		c.l.Warn("Unknown method", "method", method, "endpoint", endpoint)
	}
	if err != nil {
		c.l.Warn("Unable to recieve from API", "endpoint", endpoint, "method", method, "err", err)
		return "", err
	}
	defer resp.Body.Close()

	// Don't worry about anything other than just reading to a string.
	// No API results are massive.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.l.Warn("Unable to read response from API", "endpoint", endpoint, "method", method, "err", err)
		return "", err
	}

	return string(body), nil
}

// Clean target via API
func (c *APIClient) Clean(target string) error {
	jsonText, err := c.do("/clean/"+target, "POST")
	if err != nil {
		return err
	}
	if jsonText != "" {
		var errText map[string]string
		json.Unmarshal([]byte(jsonText), &errText)
		c.l.Warn("Error cleaning", "target", target, "err", errText["Error"])
		return errors.New("error cleaning")
	}
	return nil
}

// SyncTo requests a remote graph server to syncronize to the provided
// git hash.
func (c *APIClient) SyncTo(rev string) error {
	_, err := c.do("/syncto/"+rev, "POST")
	return err
}

// Dispatchable represents a list of builds that can be dispatched in
// parallel right now.
type Dispatchable struct {
	Pkgs map[types.SpecTuple][]string
	Rev  string
}

type rawDispatchable struct {
	Pkgs     map[string][]string
	Revision string
}

// GetDispatchable queries a remote graph server to determine what
// packages can be dispatched.
func (c *APIClient) GetDispatchable() (*Dispatchable, error) {
	jsonResult, err := c.do("/dispatchable", "GET")
	if err != nil {
		return nil, err
	}
	data := rawDispatchable{}
	err = json.Unmarshal([]byte(jsonResult), &data)
	if err != nil {
		c.l.Warn("Error unmarshalling dispatchable", "err", err)
		return nil, err
	}

	origPkgs := data.Pkgs
	pkgs := make(map[types.SpecTuple][]string)
	for tuple, list := range origPkgs {
		specTuple := types.SpecTuple{strings.Split(tuple, ":")[0], strings.Split(tuple, ":")[1]}
		pkgs[specTuple] = list
	}

	result := Dispatchable{
		Pkgs: pkgs,
		Rev:  data.Revision,
	}
	return &result, nil
}
