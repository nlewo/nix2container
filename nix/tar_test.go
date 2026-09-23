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
	expectedDigest := "sha256:1ea63d00b937dc24c711265b80444cc9e7e63751fb7f349b160be61d31381983"
	assert.Equal(t, expectedDigest, digest.String())

	expectedSize := int64(4096)
	assert.Equal(t, expectedSize, size)
	if size != expectedSize {
		t.Errorf("Size is %d while it should be %d", size, expectedSize)
	}
}
