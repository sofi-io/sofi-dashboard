package storage

import (
	"context"
	"io"
)

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key    string
	Size   int64
	SHA256 string
}

// Storage defines the contract for document storage.
type Storage interface {
	Put(
		ctx context.Context,
		key string,
		reader io.Reader,
	) (ObjectInfo, error)

	Get(
		ctx context.Context,
		key string,
	) (io.ReadCloser, error)

	Delete(
		ctx context.Context,
		key string,
	) error

	Exists(
		ctx context.Context,
		key string,
	) (bool, error)
}
