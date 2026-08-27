package nix

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"regexp"
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

func TestNixCaseHackBase(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		ok       bool
	}{
		{name: "filename~nix~case~hack~1", expected: "filename", ok: true},
		{name: "filename~nix~case~hack~42", expected: "filename", ok: true},
		{name: "filename~nix~case~hack~notes"},
		{name: "filename~nix~case~hack~0"},
		{name: "filename~nix~case~hack~01"},
		{name: "filename~nix~"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			base, ok := nixCaseHackBase(test.name)
			assert.Equal(t, test.ok, ok)
			assert.Equal(t, test.expected, base)
		})
	}
}

func pathWithRootRewrite(root string) types.Path {
	return types.Path{
		Path: root,
		Options: &types.PathOptions{Rewrite: types.Rewrite{
			Regex: "^" + regexp.QuoteMeta(root),
			Repl:  "",
		}},
	}
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("terminfo"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeTerminfoTree(t *testing.T, caseHacked bool) types.Paths {
	t.Helper()
	if caseHacked {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "share", "terminfo", "L", "linux"))
		writeFile(t, filepath.Join(root, "share", "terminfo", "l~nix~case~hack~1", "linux"))
		return types.Paths{pathWithRootRewrite(root)}
	}

	var paths types.Paths
	for _, directory := range []string{"L", "l"} {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "linux"))
		paths = append(paths, types.Path{
			Path: root,
			Options: &types.PathOptions{Rewrite: types.Rewrite{
				Regex: "^" + regexp.QuoteMeta(root),
				Repl:  filepath.Join("/share/terminfo", directory),
			}},
		})
	}
	return paths
}

func tarNames(t *testing.T, paths types.Paths) []string {
	t.Helper()
	reader := TarPaths(paths)
	defer reader.Close() // nolint: errcheck

	var names []string
	tr := tar.NewReader(reader)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return names
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, hdr.Name)
	}
}

func TestTarPathsPreservesLiteralNixCaseHackNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"FOO",
		"foo~nix~case~hack~2",
		"foo~nix~case~hack~notes",
		"standalone~nix~case~hack~1",
		"same",
		"same~nix~case~hack~1",
	} {
		writeFile(t, filepath.Join(root, name))
	}

	names := tarNames(t, types.Paths{pathWithRootRewrite(root)})
	assert.Contains(t, names, "/FOO")
	assert.Contains(t, names, "/foo~nix~case~hack~2")
	assert.Contains(t, names, "/foo~nix~case~hack~notes")
	assert.Contains(t, names, "/standalone~nix~case~hack~1")
	assert.Contains(t, names, "/same")
	assert.Contains(t, names, "/same~nix~case~hack~1")
}

func TestTarPathsCanonicalizesSequentialNixCaseHackNames(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{
		"BAR",
		"Bar~nix~case~hack~1",
		"bar~nix~case~hack~2",
		"QUX~nix~case~hack~1",
		"qux~nix~case~hack~1~nix~case~hack~1",
	} {
		writeFile(t, filepath.Join(root, name))
	}

	names := tarNames(t, types.Paths{pathWithRootRewrite(root)})
	assert.Contains(t, names, "/BAR")
	assert.Contains(t, names, "/Bar")
	assert.Contains(t, names, "/bar")
	assert.NotContains(t, names, "/Bar~nix~case~hack~1")
	assert.NotContains(t, names, "/bar~nix~case~hack~2")
	assert.Contains(t, names, "/QUX~nix~case~hack~1")
	assert.Contains(t, names, "/qux~nix~case~hack~1")
	assert.NotContains(t, names, "/qux~nix~case~hack~1~nix~case~hack~1")
}

func TestTarPathsCanonicalizesNixCaseHackSuffixAcrossBuilds(t *testing.T) {
	tests := []struct {
		name           string
		buildCaseHack  bool
		streamCaseHack bool
	}{
		{name: "linux-native"},
		{name: "darwin-native", buildCaseHack: true, streamCaseHack: true},
		{name: "darwin-host-linux-builder", streamCaseHack: true},
		{name: "darwin-store-linux-builder", buildCaseHack: true, streamCaseHack: true},
		{name: "darwin-build-linux-stream", buildCaseHack: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buildDigest, buildSize, err := TarPathsSum(writeTerminfoTree(t, test.buildCaseHack))
			if err != nil {
				t.Fatal(err)
			}
			streamDigest, streamSize, err := TarPathsSum(writeTerminfoTree(t, test.streamCaseHack))
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, buildDigest, streamDigest)
			assert.Equal(t, buildSize, streamSize)
		})
	}
}
