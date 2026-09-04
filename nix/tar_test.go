package nix

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/assert"
)

func TestTar(t *testing.T) {
	path := types.Path{
		Path: "../data/tar-directory",
	}
	digest, size, err := TarPathsSum(types.Paths{path})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expectedDigest := "sha256:1ea63d00b937dc24c711265b80444cc9e7e63751fb7f349b160be61d31381983"
	assert.Equal(t, expectedDigest, digest.String())

	expectedSize := int64(4096)
	assert.Equal(t, expectedSize, size)
	if size != expectedSize {
		t.Errorf("Size is %d while it should be %d", size, expectedSize)
	}
}

// makeTree writes a tree with directories, small and large files and a
// symlink, and returns its root.
func makeTree(t testing.TB, dirs, filesPerDir int) string {
	t.Helper()
	root := t.TempDir()
	for d := 0; d < dirs; d++ {
		dir := filepath.Join(root, fmt.Sprintf("dir%03d", d))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		for f := 0; f < filesPerDir; f++ {
			// Sizes from a few bytes to more than the 32 KiB copy buffer.
			data := bytes.Repeat([]byte{byte('a' + f%26)}, 1+(f*7919)%70000)
			if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("file%03d", f)), data, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.Symlink("dir000/file000", filepath.Join(root, "link")); err != nil {
		t.Fatal(err)
	}
	return root
}

// The file written by TarPathsWrite must hold the same bytes as the
// stream that TarPathsSum hashes.
func TestTarPathsWriteMatchesSum(t *testing.T) {
	paths := types.Paths{{Path: makeTree(t, 5, 20)}}
	sum, size, err := TarPathsSum(paths)
	if err != nil {
		t.Fatal(err)
	}
	path, written, writtenSize, err := TarPathsWrite(paths, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, sum, written)
	assert.Equal(t, size, writtenSize)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, size, int64(len(data)))
	assert.Equal(t, sum, digest.FromBytes(data))
}

func BenchmarkTarPathsSum(b *testing.B) {
	paths := types.Paths{{Path: makeTree(b, 50, 100)}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, _, err := TarPathsSum(paths); err != nil {
			b.Fatal(err)
		}
	}
}
