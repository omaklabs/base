package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// LocalStorage implements the Storage interface using the local filesystem.
type LocalStorage struct {
	basePath string // e.g., "./data/uploads"
	baseURL  string // e.g., "/uploads" (served via HTTP)
}

// NewLocalStorage creates a new LocalStorage rooted at basePath. The directory
// is created if it does not already exist. baseURL is the URL prefix used when
// generating file URLs.
func NewLocalStorage(basePath, baseURL string) (*LocalStorage, error) {
	if err := os.MkdirAll(basePath, 0o755); err != nil {
		return nil, fmt.Errorf("creating storage directory: %w", err)
	}
	return &LocalStorage{
		basePath: basePath,
		baseURL:  strings.TrimRight(baseURL, "/"),
	}, nil
}

// Put stores a file at the given key under basePath. Subdirectories are created
// as needed. If contentType is empty it is detected from the file content.
func (ls *LocalStorage) Put(_ context.Context, key string, reader io.Reader, contentType string) (FileInfo, error) {
	fullPath := filepath.Join(ls.basePath, key)

	// Ensure parent directories exist.
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileInfo{}, fmt.Errorf("creating directories for %q: %w", key, err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("creating file %q: %w", key, err)
	}
	defer f.Close()

	// If no content type is provided, sniff it from the first 512 bytes.
	var actualReader io.Reader = reader
	if contentType == "" {
		buf := make([]byte, 512)
		n, readErr := io.ReadAtLeast(reader, buf, 1)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return FileInfo{}, fmt.Errorf("reading for content detection: %w", readErr)
		}
		contentType = http.DetectContentType(buf[:n])
		// Prepend the sniffed bytes back so they are written to the file.
		actualReader = io.MultiReader(
			readerFromBytes(buf[:n]),
			reader,
		)
	}

	size, err := io.Copy(f, actualReader)
	if err != nil {
		return FileInfo{}, fmt.Errorf("writing file %q: %w", key, err)
	}

	stat, err := f.Stat()
	if err != nil {
		return FileInfo{}, fmt.Errorf("stat file %q: %w", key, err)
	}

	return FileInfo{
		Key:          key,
		Size:         size,
		ContentType:  contentType,
		LastModified: stat.ModTime(),
	}, nil
}

// Get retrieves a file by key. The caller must close the returned ReadCloser.
func (ls *LocalStorage) Get(_ context.Context, key string) (io.ReadCloser, FileInfo, error) {
	fullPath := filepath.Join(ls.basePath, key)

	f, err := os.Open(fullPath)
	if err != nil {
		return nil, FileInfo{}, fmt.Errorf("opening file %q: %w", key, err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, FileInfo{}, fmt.Errorf("stat file %q: %w", key, err)
	}

	// Detect content type by reading the first 512 bytes.
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	ct := http.DetectContentType(buf[:n])

	// Seek back to the beginning so the caller gets the full content.
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		f.Close()
		return nil, FileInfo{}, fmt.Errorf("seeking file %q: %w", key, err)
	}

	info := FileInfo{
		Key:          key,
		Size:         stat.Size(),
		ContentType:  ct,
		LastModified: stat.ModTime(),
	}

	return f, info, nil
}

// Delete removes a file by key.
func (ls *LocalStorage) Delete(_ context.Context, key string) error {
	fullPath := filepath.Join(ls.basePath, key)
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("deleting file %q: %w", key, err)
	}
	return nil
}

// Exists checks if a file exists at the given key.
func (ls *LocalStorage) Exists(_ context.Context, key string) (bool, error) {
	fullPath := filepath.Join(ls.basePath, key)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("checking file %q: %w", key, err)
}

// URL returns the URL for accessing the file. For local storage this is
// baseURL + "/" + key.
func (ls *LocalStorage) URL(key string) string {
	return ls.baseURL + "/" + key
}

// readerFromBytes returns an io.Reader over a byte slice.
func readerFromBytes(b []byte) io.Reader {
	return &bytesReader{data: b}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
