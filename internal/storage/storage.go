package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo contains metadata about a stored file.
type FileInfo struct {
	Key          string    // unique file identifier / path
	Size         int64     // file size in bytes
	ContentType  string    // MIME type
	LastModified time.Time
}

// Storage defines the interface for file storage backends.
type Storage interface {
	// Put stores a file. Key is the storage path (e.g., "uploads/avatar/123.jpg").
	// Returns the FileInfo of the stored file.
	Put(ctx context.Context, key string, reader io.Reader, contentType string) (FileInfo, error)

	// Get retrieves a file by key. Caller must close the returned ReadCloser.
	Get(ctx context.Context, key string) (io.ReadCloser, FileInfo, error)

	// Delete removes a file by key.
	Delete(ctx context.Context, key string) error

	// Exists checks if a file exists.
	Exists(ctx context.Context, key string) (bool, error)

	// URL returns a URL to access the file. For local storage, returns a path.
	// For S3, could return a pre-signed URL.
	URL(key string) string
}
