package nix

import (
	"reflect"
	"testing"

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
				Digest: "sha256:59bf1c3509f33515622619af21ed55bbe26d24913cedbca106468a5fb37a50c3",
				DiffIDs:"sha256:8d3ac3489996423f53d6087c81180006263b79f206d3fdec9e66f0e27ceb8759",
				MediaType:"application/vnd.oci.image.layer.v1.tar+gzip",
				LayerPath:"../data/image-directory/59bf1c3509f33515622619af21ed55bbe26d24913cedbca106468a5fb37a50c3",
			},
		},
	}
	if !reflect.DeepEqual(image.Layers, expected.Layers) {
		t.Fatalf("Layers should be '%#v' (while they are %#v)", expected.Layers, image.Layers)
	}
}
