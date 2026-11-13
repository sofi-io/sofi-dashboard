package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"
)

func TestFilesystemStoragePutGet(t *testing.T) {
	ctx := context.Background()

	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	content := "SOFI document storage test"

	info, err := store.Put(
		ctx,
		"documents/2026/test.txt",
		strings.NewReader(content),
	)
	if err != nil {
		t.Fatal(err)
	}

	expectedHash := sha256.Sum256([]byte(content))

	if info.Size != int64(len(content)) {
		t.Fatalf("unexpected size: %d", info.Size)
	}

	if info.SHA256 != hex.EncodeToString(expectedHash[:]) {
		t.Fatalf("unexpected SHA256: %s", info.SHA256)
	}

	exists, err := store.Exists(ctx, info.Key)
	if err != nil {
		t.Fatal(err)
	}

	if !exists {
		t.Fatal("object should exist")
	}

	reader, err := store.Get(ctx, info.Key)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	got, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}

	if string(got) != content {
		t.Fatalf("unexpected content: %s", got)
	}
}

func TestFilesystemStorageDelete(t *testing.T) {
	ctx := context.Background()

	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Put(ctx, "test.txt", strings.NewReader("test"))
	if err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(ctx, "test.txt"); err != nil {
		t.Fatal(err)
	}

	exists, err := store.Exists(ctx, "test.txt")
	if err != nil {
		t.Fatal(err)
	}

	if exists {
		t.Fatal("object should not exist")
	}
}

func TestFilesystemStorageRejectsInvalidKeys(t *testing.T) {
	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	invalidKeys := []string{
		"",
		"../secret.txt",
		"../../etc/passwd",
		"/etc/passwd",
		".",
	}

	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			_, err := store.Put(
				context.Background(),
				key,
				strings.NewReader("test"),
			)

			if err == nil {
				t.Fatalf("expected invalid key error: %s", key)
			}
		})
	}
}

func TestFilesystemStorageCancelledContext(t *testing.T) {
	store, err := NewFilesystemStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = store.Put(ctx, "test.txt", strings.NewReader("test"))
	if err == nil {
		t.Fatal("expected context cancellation")
	}
}
