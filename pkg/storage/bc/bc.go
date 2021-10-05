package bc

import (
	"errors"
	"os"

	"git.mills.io/prologic/bitcask"
	"github.com/hashicorp/go-hclog"

	"github.com/the-maldridge/nbuild/pkg/storage"
)

// bcStore is the type that must satisfy web.Store
type bcStore struct {
	s *bitcask.Bitcask

	l hclog.Logger
}

func init() {
	storage.RegisterCallback(newFactory)
}

func newFactory() {
	storage.RegisterFactory("bitcask", newBCStore)
}

func newBCStore(l hclog.Logger) (storage.Storage, error) {
	x := new(bcStore)
	x.l = l.Named("bitcask")

	p := os.Getenv("NBUILD_BITCASK_PATH")
	if p == "" {
		l.Error("NBUILD_BITCASK_PATH must be set")
		return nil, errors.New("required variable unset")
	}

	opts := []bitcask.Option{
		bitcask.WithMaxKeySize(1024),
		bitcask.WithMaxValueSize(1024 * 1000 * 32), // 32MiB
		bitcask.WithSync(true),
	}
	b, err := bitcask.Open(p, opts...)
	if err != nil {
		l.Error("Error initializing bitcask", "error", err)
		return nil, err
	}
	x.s = b

	return x, nil
}

func (b *bcStore) Get(k []byte) ([]byte, error) {
	v, err := b.s.Get(k)
	switch err {
	case nil:
		return v, nil
	case bitcask.ErrKeyNotFound:
		return nil, nil
	default:
		return nil, err
	}
}

func (b *bcStore) Put(k, v []byte) error {
	return b.s.Put(k, v)
}

func (b *bcStore) Del(k []byte) error {
	return b.s.Delete(k)
}

func (b *bcStore) Close() error {
	return b.s.Close()
}
