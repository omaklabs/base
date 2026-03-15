package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalStoragePutAndGet(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	content := []byte("hello, world!")

	info, err := ls.Put(ctx, "test.txt", bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	if info.Key != "test.txt" {
		t.Errorf("info.Key = %q, want %q", info.Key, "test.txt")
	}
	if info.Size != int64(len(content)) {
		t.Errorf("info.Size = %d, want %d", info.Size, len(content))
	}
	if info.ContentType != "text/plain" {
		t.Errorf("info.ContentType = %q, want %q", info.ContentType, "text/plain")
	}

	// Get the file back.
	rc, getInfo, err := ls.Get(ctx, "test.txt")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content = %q, want %q", string(got), string(content))
	}
	if getInfo.Key != "test.txt" {
		t.Errorf("getInfo.Key = %q, want %q", getInfo.Key, "test.txt")
	}
	if getInfo.Size != int64(len(content)) {
		t.Errorf("getInfo.Size = %d, want %d", getInfo.Size, len(content))
	}
}

func TestLocalStorageDelete(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	content := []byte("to be deleted")

	_, err = ls.Put(ctx, "delete-me.txt", bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Verify it exists.
	exists, err := ls.Exists(ctx, "delete-me.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Fatal("expected file to exist before delete")
	}

	// Delete.
	if err := ls.Delete(ctx, "delete-me.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify it no longer exists.
	exists, err = ls.Exists(ctx, "delete-me.txt")
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Error("expected file to not exist after delete")
	}
}

func TestLocalStorageExists(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()

	// Should not exist yet.
	exists, err := ls.Exists(ctx, "nonexistent.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if exists {
		t.Error("expected nonexistent.txt to not exist")
	}

	// Create a file.
	_, err = ls.Put(ctx, "exists.txt", bytes.NewReader([]byte("data")), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Should exist now.
	exists, err = ls.Exists(ctx, "exists.txt")
	if err != nil {
		t.Fatalf("Exists: %v", err)
	}
	if !exists {
		t.Error("expected exists.txt to exist")
	}
}

func TestLocalStorageURL(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	url := ls.URL("avatars/123.jpg")
	want := "/uploads/avatars/123.jpg"
	if url != want {
		t.Errorf("URL = %q, want %q", url, want)
	}
}

func TestLocalStorageCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	content := []byte("nested file content")

	// Put a file in a deeply nested path.
	_, err = ls.Put(ctx, "a/b/c/deep.txt", bytes.NewReader(content), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Verify the file exists on disk.
	fullPath := filepath.Join(dir, "a", "b", "c", "deep.txt")
	data, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(data, content) {
		t.Errorf("file content = %q, want %q", string(data), string(content))
	}
}

func TestLocalStorageImplementsStorage(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	// This is a compile-time check that *LocalStorage satisfies the Storage interface.
	var _ Storage = ls
}

func TestLocalStoragePutDetectsContentType(t *testing.T) {
	dir := t.TempDir()
	ls, err := NewLocalStorage(dir, "/uploads")
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()

	// Put with empty content type — should be auto-detected.
	content := []byte("<html><body>Hello</body></html>")
	info, err := ls.Put(ctx, "page.html", bytes.NewReader(content), "")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	if !strings.Contains(info.ContentType, "text/html") {
		t.Errorf("expected content type containing 'text/html', got %q", info.ContentType)
	}

	// Verify content was written correctly.
	rc, _, err := ls.Get(ctx, "page.html")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch after content-type detection put")
	}
}
