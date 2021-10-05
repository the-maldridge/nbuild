package storage

// Storage is an interfaces for a generic blobstore.
type Storage interface {
	Get([]byte) ([]byte, error)
	Put([]byte, []byte) error
	Del([]byte) error

	Close() error
}
