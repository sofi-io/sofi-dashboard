package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type FilesystemStorage struct {
	root string
}

func NewFilesystemStorage(root string) (*FilesystemStorage, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("storage root is required")
	}

	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve storage root: %w", err)
	}

	if err := os.MkdirAll(absoluteRoot, 0750); err != nil {
		return nil, fmt.Errorf("create storage root: %w", err)
	}

	return &FilesystemStorage{
		root: absoluteRoot,
	}, nil
}

func (s *FilesystemStorage) resolve(key string) (string, error) {
	if s == nil || s.root == "" {
		return "", fmt.Errorf("storage is not initialized")
	}

	if strings.TrimSpace(key) == "" {
		return "", fmt.Errorf("storage key is required")
	}

	normalized := filepath.FromSlash(key)

	if !filepath.IsLocal(normalized) || normalized == "." {
		return "", fmt.Errorf("invalid storage key: %s", key)
	}

	fullPath := filepath.Join(s.root, normalized)

	rel, err := filepath.Rel(s.root, fullPath)
	if err != nil {
		return "", fmt.Errorf("resolve storage key: %w", err)
	}

	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("storage key escapes root: %s", key)
	}

	return fullPath, nil
}

func (s *FilesystemStorage) Put(
	ctx context.Context,
	key string,
	reader io.Reader,
) (ObjectInfo, error) {
	if reader == nil {
		return ObjectInfo{}, fmt.Errorf("reader is required")
	}

	if err := ctx.Err(); err != nil {
		return ObjectInfo{}, err
	}

	target, err := s.resolve(key)
	if err != nil {
		return ObjectInfo{}, err
	}

	if err := os.MkdirAll(filepath.Dir(target), 0750); err != nil {
		return ObjectInfo{}, fmt.Errorf("create object directory: %w", err)
	}

	temp, err := os.CreateTemp(filepath.Dir(target), ".upload-*")
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("create temporary object: %w", err)
	}

	tempName := temp.Name()
	defer os.Remove(tempName)

	hash := sha256.New()

	written, err := io.Copy(
		io.MultiWriter(temp, hash),
		&contextReader{
			ctx: ctx,
			r:   reader,
		},
	)

	if err != nil {
		temp.Close()
		return ObjectInfo{}, fmt.Errorf("write object: %w", err)
	}

	if err := ctx.Err(); err != nil {
		temp.Close()
		return ObjectInfo{}, err
	}

	if err := temp.Sync(); err != nil {
		temp.Close()
		return ObjectInfo{}, fmt.Errorf("sync object: %w", err)
	}

	if err := temp.Close(); err != nil {
		return ObjectInfo{}, fmt.Errorf("close object: %w", err)
	}

	if err := os.Rename(tempName, target); err != nil {
		return ObjectInfo{}, fmt.Errorf("commit object: %w", err)
	}

	return ObjectInfo{
		Key:    key,
		Size:   written,
		SHA256: hex.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (s *FilesystemStorage) Get(
	ctx context.Context,
	key string,
) (io.ReadCloser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	target, err := s.resolve(key)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(target)
	if err != nil {
		return nil, fmt.Errorf("open object: %w", err)
	}

	return file, nil
}

func (s *FilesystemStorage) Delete(
	ctx context.Context,
	key string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	target, err := s.resolve(key)
	if err != nil {
		return err
	}

	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete object: %w", err)
	}

	return nil
}

func (s *FilesystemStorage) Exists(
	ctx context.Context,
	key string,
) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}

	target, err := s.resolve(key)
	if err != nil {
		return false, err
	}

	info, err := os.Stat(target)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("stat object: %w", err)
	}

	return info.Mode().IsRegular(), nil
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}

	return r.r.Read(p)
}
