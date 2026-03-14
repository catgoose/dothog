// setup:feature:avatar
package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewPhotoStore(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "photos")
	ps, err := NewPhotoStore(dir)
	require.NoError(t, err)
	require.NotNil(t, ps)
	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestPhotoPath(t *testing.T) {
	ps := &PhotoStore{dir: "/tmp/photos"}

	t.Run("normal ID", func(t *testing.T) {
		got := ps.PhotoPath("abc-123-def")
		require.Equal(t, filepath.Join("/tmp/photos", "ab", "abc-123-def.jpg"), got)
	})

	t.Run("short ID", func(t *testing.T) {
		got := ps.PhotoPath("ab")
		require.Equal(t, filepath.Join("/tmp/photos", "ab", "ab.jpg"), got)
	})

	t.Run("single char ID", func(t *testing.T) {
		got := ps.PhotoPath("x")
		require.Equal(t, filepath.Join("/tmp/photos", "x", "x.jpg"), got)
	})
}

func TestSaveAndHasPhoto(t *testing.T) {
	ps, err := NewPhotoStore(t.TempDir())
	require.NoError(t, err)

	azureID := "abc-123-def-456"
	require.False(t, ps.HasPhoto(azureID))

	data := []byte("fake jpeg data")
	require.NoError(t, ps.Save(azureID, data))
	require.True(t, ps.HasPhoto(azureID))

	// Verify file contents
	got, err := os.ReadFile(ps.PhotoPath(azureID))
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestOpen(t *testing.T) {
	ps, err := NewPhotoStore(t.TempDir())
	require.NoError(t, err)

	azureID := "open-test-id"
	data := []byte("photo bytes")
	require.NoError(t, ps.Save(azureID, data))

	rc, err := ps.Open(azureID)
	require.NoError(t, err)
	defer rc.Close()

	buf := make([]byte, len(data))
	n, err := rc.Read(buf)
	require.NoError(t, err)
	require.Equal(t, len(data), n)
	require.Equal(t, data, buf)
}

func TestOpenNotFound(t *testing.T) {
	ps, err := NewPhotoStore(t.TempDir())
	require.NoError(t, err)

	_, err = ps.Open("nonexistent")
	require.Error(t, err)
}

func TestSaveOverwrite(t *testing.T) {
	ps, err := NewPhotoStore(t.TempDir())
	require.NoError(t, err)

	azureID := "overwrite-test"
	require.NoError(t, ps.Save(azureID, []byte("old")))
	require.NoError(t, ps.Save(azureID, []byte("new")))

	got, err := os.ReadFile(ps.PhotoPath(azureID))
	require.NoError(t, err)
	require.Equal(t, []byte("new"), got)
}
