package nix

import (
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
	expectedDigest := "sha256:25561a5f9e53bfd69476c622953dee27868291dd67c478b499da6a1962893bb4"
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
