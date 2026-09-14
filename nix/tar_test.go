package nix

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nlewo/nix2container/types"
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

func TestRemoveNixCaseHackSuffix(t *testing.T) {
	ret := removeNixCaseHackSuffix("filename~nix~case~hack~1")
	expected := "filename"
	if ret != expected {
		t.Errorf("%s should be %s", ret, expected)
	}
	ret = removeNixCaseHackSuffix("/path~nix~case~hack~1/filename")
	expected = "/path/filename"
	if ret != expected {
		t.Errorf("%s should be %s", ret, expected)
	}
	ret = removeNixCaseHackSuffix("filename~nix~")
	expected = "filename~nix~"
	if ret != expected {
		t.Errorf("%s should be %s", ret, expected)
	}
}

// An excluded subtree and everything under it is left out of the tar;
// a sibling with the same prefix is not.
func TestTarPathsExcludes(t *testing.T) {
	root := t.TempDir()
	for _, d := range []string{"share/doc/a", "share/doc/b", "share/docs", "bin"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, d, "f"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	paths := types.Paths{{Path: root, Options: &types.PathOptions{Excludes: []string{"share/doc", "bin/f"}}}}
	r := TarPaths(paths)
	defer r.Close() // nolint: errcheck
	tr := tar.NewReader(r)
	var names []string
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, strings.TrimPrefix(hdr.Name, root))
	}
	joined := strings.Join(names, "\n")
	assert.NotContains(t, joined, "/share/doc/")
	assert.NotContains(t, joined, "/share/doc\n")
	assert.NotContains(t, joined, "/bin/f")
	assert.Contains(t, joined, "/share/docs/f")
	assert.Contains(t, joined, "/bin\n")
}
