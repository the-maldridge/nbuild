package reciever

import (
	"sync"

	"github.com/hashicorp/go-hclog"
)

// Reciever takes build package artifacts via HTTP and incorporates them into
// a XBPS repository.
type Reciever struct {
	l         hclog.Logger
	path      string
	repoMutex *sync.Mutex
}
