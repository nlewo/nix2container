package nix

import (
	"archive/tar"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
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

// Invalid orMode values must fail loudly at tar time, not be silently
// misparsed: Sscanf-style laxity here corrupts header modes (a negative
// mode even flips archive/tar to GNU base-256 encoding).
func TestTarPermsOrModeInvalid(t *testing.T) {
	for _, orMode := range []string{"0o311", "-0200", "40755", "8"} {
		t.Run(orMode, func(t *testing.T) {
			paths := types.Paths{{
				Path: "../data/layer1/file1",
				Options: &types.PathOptions{
					Perms: []types.Perm{{Regex: ".*", OrMode: orMode}},
				},
			}}
			r := TarPaths(paths)
			defer r.Close() // nolint: errcheck
			tr := tar.NewReader(r)
			var err error
			for err == nil {
				_, err = tr.Next()
			}
			assert.ErrorContains(t, err, "invalid orMode")
		})
	}
}

// makeTree writes a tree shaped like a store path and returns its root.
//
// The shape is taken from rustc-1.97.1 (3253 files): a median of a few
// KiB, around 80% of the files below copyBufferSize, and a tail of large
// ones. A uniform distribution would understate how many files a layer
// carries per byte, which is what this package pays for. The tail stops
// at 2 MiB so the tree stays around 150 MB at 5000 files; a real store
// path reaches a hundred times that in a single binary.
//
// The bytes are random, not a repeated byte: identical contents would
// make any compression measured on this tree meaningless. The seed is
// fixed, so the tree is the same on every run.
func makeTree(t testing.TB, dirs, filesPerDir int) string {
	t.Helper()
	root := t.TempDir()
	rng := rand.New(rand.NewSource(1))
	// One pool of random bytes, sliced at a random offset per file, so
	// the files differ without paying for fresh randomness each time.
	pool := make([]byte, 8<<20)
	if _, err := rng.Read(pool); err != nil {
		t.Fatal(err)
	}
	for d := 0; d < dirs; d++ {
		for f := 0; f < filesPerDir; f++ {
			// Nested, because a store path is not a flat directory.
			dir := filepath.Join(root, fmt.Sprintf("dir%03d", d), fmt.Sprintf("sub%d", f%4))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			var size int
			switch r := rng.Float64(); {
			case r < 0.745:
				size = 100 + rng.Intn(8<<10)
			case r < 0.945:
				size = 8<<10 + rng.Intn(56<<10)
			case r < 0.995:
				size = 64<<10 + rng.Intn(448<<10)
			default:
				size = 512<<10 + rng.Intn(3<<20)
			}
			off := rng.Intn(len(pool) - size)
			name := filepath.Join(dir, fmt.Sprintf("file%03d", f))
			if err := os.WriteFile(name, pool[off:off+size], 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.Symlink("dir000/sub0/file000", filepath.Join(root, "link")); err != nil {
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := TarPathsSum(paths); err != nil {
			b.Fatal(err)
		}
	}
}

// TarPathsSum never writes the stream out, so this covers the file path:
// the buffered writer and the flush. Point TMPDIR at a real filesystem to
// see the buffering, since write syscalls to tmpfs are nearly free.
func BenchmarkTarPathsWrite(b *testing.B) {
	paths := types.Paths{{Path: makeTree(b, 50, 100)}}
	dir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Every iteration writes the same digest, and so the same file.
		if _, _, _, err := TarPathsWrite(paths, dir); err != nil {
			b.Fatal(err)
		}
	}
}

// The copy must stage through the one buffer per stream, not through a
// new one per file. This is easy to lose: io.CopyBuffer silently drops
// the buffer it is given when the source implements io.WriterTo or the
// destination implements io.ReaderFrom, and the tar bytes come out the
// same either way, so nothing else here would notice.
func TestTarPathsSumReusesTheCopyBuffer(t *testing.T) {
	const files = 5 * 40
	paths := types.Paths{{Path: makeTree(t, 5, 40)}}

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	if _, _, err := TarPathsSum(paths); err != nil {
		t.Fatal(err)
	}
	runtime.ReadMemStats(&after)
	allocated := after.TotalAlloc - before.TotalAlloc

	// A buffer per file would be at least files*copyBufferSize, which
	// measures around 6.7 MiB here against around 0.4 MiB. The bound
	// sits between the two, far enough from both to not be flaky.
	if max := uint64(2 << 20); allocated > max {
		t.Errorf("%d files allocated %d bytes, more than %d: the copy is no longer using the shared buffer (a buffer per file would be %d)",
			files, allocated, max, files*copyBufferSize)
	}
}
