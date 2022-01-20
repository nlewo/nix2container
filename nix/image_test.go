package nix

import (
	"testing"
	"reflect"
	"github.com/nlewo/nix2container/types"
)

func TestNewImageFromDir(t *testing.T) {
	image, err := NewImageFromDir("../data/image-directory")
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := types.Image{
		Layers: []types.Layer{
			types.Layer{
				Digest: "sha256:688e187d6c79c46e8261890f0010fd5d178b8faa178959b0b46b2635aa1eeff3",
				TarPath: "../data/image-directory/688e187d6c79c46e8261890f0010fd5d178b8faa178959b0b46b2635aa1eeff3",
			},
			types.Layer{
				Digest: "sha256:788e187d6c79c46e8261890f0010fd5d178b8faa178959b0b46b2635aa1eeff4",
				TarPath: "../data/image-directory/788e187d6c79c46e8261890f0010fd5d178b8faa178959b0b46b2635aa1eeff4",
			},
		},
	}
	if !reflect.DeepEqual(image.Layers, expected.Layers) {
		t.Fatalf("Response should be '%#v' (while it is %#v)", expected.Layers, image.Layers)
	}
}
