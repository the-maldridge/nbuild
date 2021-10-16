package graph

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/types"
)

// NewAPIClient creates a new API client.
func NewAPIClient(l hclog.Logger) *APIClient {
	x := APIClient{
		l:       l.Named("client"),
		hClient: &http.Client{Timeout: 30 * time.Second},
	}
	return &x
}

// General function to recieve a response
func (c *APIClient) do(endpoint string, method string) (string, bool) {
	if c.Url == "" {
		c.l.Warn("Url not set for API", "endpoint", endpoint)
		return "", false
	}

	var resp *http.Response
	var err error
	fullUrl := "http://" + c.Url + "/api/graph" + endpoint
	switch method {
	case "GET":
		resp, err = c.hClient.Get(fullUrl)
	case "POST":
		resp, err = c.hClient.Post(fullUrl, "application/json", bytes.NewBuffer([]byte("{}")))
	default:
		c.l.Warn("Unknown method", "method", method, "endpoint", endpoint)
	}
	if err != nil {
		c.l.Warn("Unable to recieve from API", "endpoint", endpoint, "method", method, "err", err)
		return "", false
	}
	defer resp.Body.Close()

	// Don't worry about anything other than just reading to a string.
	// No API results are massive.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		c.l.Warn("Unable to read response from API", "endpoint", endpoint, "method", method, "err", err)
		return "", false
	}

	return string(body), true
}

// Clean target via API
func (c *APIClient) Clean(target string) bool {
	jsonText, ok := c.do("/clean/"+target, "POST")
	if jsonText != "" {
		var errText map[string]string
		json.Unmarshal([]byte(jsonText), &errText)
		c.l.Warn("Error cleaning", "target", target, "err", errText["Error"])
		return false
	}
	return ok
}

// Syncto via API
func (c *APIClient) SyncTo(rev string) bool {
	_, ok := c.do("/syncto/"+rev, "POST")
	return ok
}

// Struct to store dispatchable
type Dispatchable struct {
	Pkgs map[types.SpecTuple][]string
	Rev  string
}

type rawDispatchable struct {
	Pkgs     map[string][]string
	Revision string
}

// Get dispatchable via API
func (c *APIClient) GetDispatchable() (*Dispatchable, bool) {
	jsonResult, ok := c.do("/dispatchable", "GET")
	if !ok {
		return nil, false
	}
	data := rawDispatchable{}
	err := json.Unmarshal([]byte(jsonResult), &data)
	if err != nil {
		c.l.Warn("Error unmarshalling dispatchable", "err", err)
		return nil, false
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
	return &result, true
}
