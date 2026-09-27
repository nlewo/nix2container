package nix

import (
	"archive/tar"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"
)

// tarEntries returns the names of the entries of the tar built from
// paths, sorted.
func tarEntries(t *testing.T, paths types.Paths) []string {
	t.Helper()
	r := TarPaths(paths)
	defer r.Close() //nolint:errcheck
	tr := tar.NewReader(r)
	var names []string
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		names = append(names, hdr.Name)
	}
	sort.Strings(names)
	return names
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// SkipCopyTo is matched against the destination path, so a rewrite that
// moves a tree out of the matched prefix keeps it: this is what
// includeStorePaths = false relies on, since copyToRoot is rewritten to
// the image root while the closure stays under the store directory.
func TestSkipCopyToMatchesTheDestination(t *testing.T) {
	root := t.TempDir()
	store := filepath.Join(root, "nix", "store")
	writeFile(t, filepath.Join(store, "aaa-kept", "bin", "hello"))
	writeFile(t, filepath.Join(store, "bbb-dropped", "lib", "libc.so"))

	paths := types.Paths{
		{
			Path: filepath.Join(store, "aaa-kept"),
			Options: &types.PathOptions{
				Rewrite:    types.Rewrite{Regex: "^" + filepath.Join(store, "aaa-kept"), Repl: ""},
				SkipCopyTo: "^" + store,
			},
		},
		{
			Path:    filepath.Join(store, "bbb-dropped"),
			Options: &types.PathOptions{SkipCopyTo: "^" + store},
		},
	}

	names := tarEntries(t, paths)
	assert.Contains(t, names, "/bin/hello")
	for _, n := range names {
		assert.NotContains(t, n, "bbb-dropped")
	}
}

// A directory whose destination matches takes its subtree with it, and
// the walk does not descend into it. The unreadable directory would
// make the walk fail if it were entered.
func TestSkipCopyToPrunesTheWalk(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep", "file"))
	locked := filepath.Join(root, "dropped", "locked")
	writeFile(t, filepath.Join(locked, "file"))
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o755) //nolint:errcheck
	if os.Geteuid() == 0 {
		t.Skip("root reads unreadable directories")
	}

	paths := types.Paths{{
		Path:    root,
		Options: &types.PathOptions{SkipCopyTo: filepath.Join(root, "dropped") + "$"},
	}}

	names := tarEntries(t, paths)
	for _, n := range names {
		assert.NotContains(t, n, "dropped")
	}
	assert.Contains(t, names, filepath.Join(root, "keep", "file"))
}

// An invalid regex has to fail the build rather than be ignored.
func TestSkipCopyToInvalidRegex(t *testing.T) {
	paths := types.Paths{{
		Path:    t.TempDir(),
		Options: &types.PathOptions{SkipCopyTo: "("},
	}}
	_, _, err := TarPathsSum(paths)
	assert.ErrorContains(t, err, "invalid skip-copy-to regex")
}

// An empty SkipCopyTo means no filter. Compiling "" would give a regex
// that matches everything and silently empty the layer.
func TestSkipCopyToEmptyIsNoFilter(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "keep", "file"))

	paths := types.Paths{{Path: root, Options: &types.PathOptions{SkipCopyTo: ""}}}
	assert.Contains(t, tarEntries(t, paths), filepath.Join(root, "keep", "file"))
}
