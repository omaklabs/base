package storage

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseUpload(t *testing.T) {
	// Create a multipart form with a file field.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("avatar", "photo.jpg")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}

	fileContent := []byte("fake jpeg content here, long enough to detect")
	if _, err := part.Write(fileContent); err != nil {
		t.Fatalf("Write: %v", err)
	}
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	upload, err := ParseUpload(req, "avatar")
	if err != nil {
		t.Fatalf("ParseUpload: %v", err)
	}

	if upload.Filename != "photo.jpg" {
		t.Errorf("Filename = %q, want %q", upload.Filename, "photo.jpg")
	}
	if upload.Size != int64(len(fileContent)) {
		t.Errorf("Size = %d, want %d", upload.Size, len(fileContent))
	}
	if upload.ContentType == "" {
		t.Error("ContentType should not be empty")
	}

	// Read all content from the upload reader.
	got, err := io.ReadAll(upload.Reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, fileContent) {
		t.Errorf("content = %q, want %q", string(got), string(fileContent))
	}
}

func TestParseUploadNoFile(t *testing.T) {
	// Create a multipart form WITHOUT a file.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("name", "test")
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	_, err := ParseUpload(req, "avatar")
	if err == nil {
		t.Fatal("expected error when no file is uploaded, got nil")
	}
}

func TestGenerateKey(t *testing.T) {
	key1 := GenerateKey("avatars", "photo.jpg")
	key2 := GenerateKey("avatars", "photo.jpg")

	// Keys should be different (unique UUID).
	if key1 == key2 {
		t.Errorf("expected different keys, both are %q", key1)
	}

	// Keys should have the format prefix/uuid.ext.
	if !strings.HasPrefix(key1, "avatars/") {
		t.Errorf("key %q should start with 'avatars/'", key1)
	}
	if !strings.HasSuffix(key1, ".jpg") {
		t.Errorf("key %q should end with '.jpg'", key1)
	}

	// The UUID part should be present between prefix and extension.
	parts := strings.Split(key1, "/")
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts separated by '/', got %d in %q", len(parts), key1)
	}
	uuidPart := strings.TrimSuffix(parts[1], ".jpg")
	if len(uuidPart) == 0 {
		t.Error("UUID part should not be empty")
	}
}

func TestGenerateKeyPreservesExtension(t *testing.T) {
	tests := []struct {
		filename string
		wantExt  string
	}{
		{"photo.jpg", ".jpg"},
		{"image.png", ".png"},
		{"document.pdf", ".pdf"},
		{"archive.tar.gz", ".gz"},
		{"noext", ""},
		{"Photo.JPG", ".jpg"}, // lowercased
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			key := GenerateKey("files", tt.filename)
			ext := filepath.Ext(key)
			if ext != tt.wantExt {
				t.Errorf("extension = %q, want %q (key=%q)", ext, tt.wantExt, key)
			}
		})
	}
}

func TestGenerateKeyEmptyPrefix(t *testing.T) {
	key := GenerateKey("", "photo.jpg")

	// Should not start with a "/".
	if strings.HasPrefix(key, "/") {
		t.Errorf("key %q should not start with '/'", key)
	}
	if !strings.HasSuffix(key, ".jpg") {
		t.Errorf("key %q should end with '.jpg'", key)
	}
}
