package nix

import (
	"github.com/nlewo/nix2container/types"
	"testing"
)

func TestTar(t *testing.T) {
	path := types.Path{
		Path: "../data/tar-directory",
	}
	digest, size, err := TarPathsSum(types.Paths{path})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expectedDigest := "sha256:efccbbe35209d59cfeebd8e73785258d3679fa258f72a7dfbc2eec65695fd5c8"
	if digest.String() != expectedDigest {
		t.Fatalf("Digest is %s while it should be %s", digest.String(), expectedDigest)
	}
	expectedSize := int64(3584)
	if size != expectedSize {
		t.Fatalf("Size is %d while it should be %d", size, expectedSize)
	}
}
