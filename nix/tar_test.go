package nix

import (
	"testing"
	"github.com/nlewo/nix2container/types"
)

func TestTar(t *testing.T) {
	path := types.Path{
		Path: "../data/tar-directory",
	}
	digest, err := TarPathsSum(types.Paths{path})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := "sha256:a0a389b8df6fec3293a0b26714f77d6aa252d2304de516daa683b4a55053dc5a"
	if digest.String() != expected {
		t.Fatalf("Digest is %s while it should be %s", digest.String(), expected)
	}
}
