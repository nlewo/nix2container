package nix

import (
	"testing"

	"github.com/nlewo/nix2container/types"
)

func TestTar(t *testing.T) {
	path := types.Path{
		Path: "../data/tar-directory",
	}
	digest, size, err := TarPathsSum(types.Paths{path})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expectedDigest := "sha256:077af73ad0fb226436e92a272318b777b6976b85c3a05d86183274818dd634f8"
	if digest.String() != expectedDigest {
		t.Errorf("Digest is %s while it should be %s", digest.String(), expectedDigest)
	}
	expectedSize := int64(4096)
	if size != expectedSize {
		t.Errorf("Size is %d while it should be %d", size, expectedSize)
	}
}
