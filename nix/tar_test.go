package nix

import (
	"archive/tar"
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
