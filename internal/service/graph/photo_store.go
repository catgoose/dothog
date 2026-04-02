// setup:feature:avatar

package graph

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PhotoStore manages cached user profile photos on the filesystem.
// Photos are stored in a two-level directory structure to avoid flat
// directory performance issues: dir/ab/abcdef-1234.jpg
type PhotoStore struct {
	dir string
}

// NewPhotoStore creates a PhotoStore rooted at dir, creating the directory
// if it does not exist.
func NewPhotoStore(dir string) (*PhotoStore, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create photo dir: %w", err)
	}
	return &PhotoStore{dir: dir}, nil
}

// Save writes photo data for the given Azure ID. The write is atomic:
// data is written to a temp file then renamed into place.
func (ps *PhotoStore) Save(azureID string, data []byte) error {
	dest := ps.PhotoPath(azureID)
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create photo subdir: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dest), ".photo-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("write photo: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("rename photo: %w", err)
	}
	return nil
}

// HasPhoto reports whether a cached photo exists for the given Azure ID.
func (ps *PhotoStore) HasPhoto(azureID string) bool {
	_, err := os.Stat(ps.PhotoPath(azureID))
	return err == nil
}

// Open returns a ReadCloser for the cached photo. The caller must close it.
func (ps *PhotoStore) Open(azureID string) (io.ReadCloser, error) {
	return os.Open(ps.PhotoPath(azureID))
}

// PhotoPath returns the filesystem path for a given Azure ID's photo.
func (ps *PhotoStore) PhotoPath(azureID string) string {
	prefix := azureID
	if len(prefix) > 2 {
		prefix = prefix[:2]
	}
	return filepath.Join(ps.dir, prefix, azureID+".jpg")
}
