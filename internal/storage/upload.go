package storage

import (
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// MaxUploadSize is the default max upload size (10 MB).
const MaxUploadSize = 10 << 20

// Upload represents a parsed file upload from a multipart form.
type Upload struct {
	Filename    string
	Size        int64
	ContentType string
	Reader      io.Reader
}

// ParseUpload extracts a file upload from a multipart form request.
// fieldName is the form field name (e.g., "avatar").
// Returns the Upload or an error if no file was uploaded or parsing failed.
func ParseUpload(r *http.Request, fieldName string) (*Upload, error) {
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		return nil, fmt.Errorf("parsing multipart form: %w", err)
	}

	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, fmt.Errorf("getting form file %q: %w", fieldName, err)
	}

	// Read the first 512 bytes to detect content type.
	buf := make([]byte, 512)
	n, err := io.ReadAtLeast(file, buf, 1)
	if err != nil && err != io.ErrUnexpectedEOF {
		file.Close()
		return nil, fmt.Errorf("reading file for content detection: %w", err)
	}
	contentType := http.DetectContentType(buf[:n])

	// Create a reader that prepends the sniffed bytes back, then reads the
	// rest of the file. We do not close file here — the caller is responsible
	// for draining the reader, and the multipart form cleanup will close it.
	combined := io.MultiReader(
		readerFromBytes(buf[:n]),
		file,
	)

	return &Upload{
		Filename:    header.Filename,
		Size:        header.Size,
		ContentType: contentType,
		Reader:      combined,
	}, nil
}

// GenerateKey creates a unique storage key for an upload.
// Uses the pattern: prefix/uuid.ext (e.g., "avatars/550e8400-e29b.jpg").
func GenerateKey(prefix, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	id := generateUUID()

	if prefix == "" {
		return id + ext
	}
	return strings.TrimRight(prefix, "/") + "/" + id + ext
}

// generateUUID produces a version 4 UUID using crypto/rand.
func generateUUID() string {
	var buf [16]byte
	_, _ = rand.Read(buf[:])
	buf[6] = (buf[6] & 0x0f) | 0x40 // version 4
	buf[8] = (buf[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
