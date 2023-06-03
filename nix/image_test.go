package nix

import (
	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"
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
			{
				Digest:    "sha256:59bf1c3509f33515622619af21ed55bbe26d24913cedbca106468a5fb37a50c3",
				DiffIDs:   "sha256:8d3ac3489996423f53d6087c81180006263b79f206d3fdec9e66f0e27ceb8759",
				MediaType: "application/vnd.oci.image.layer.v1.tar+gzip",
				LayerPath: "../data/image-directory/59bf1c3509f33515622619af21ed55bbe26d24913cedbca106468a5fb37a50c3",
			},
		},
	}
	if !reflect.DeepEqual(image.Layers, expected.Layers) {
		t.Fatalf("Layers should be '%#v' (while they are %#v)", expected.Layers, image.Layers)
	}
}

func TestGetV1Image(t *testing.T) {
	image := types.Image{
		Layers: []types.Layer{
			{
				Digest:    "sha256:6123adfc04c22915c112368b802af161b921fbf7ef1c5f7283191ee552b46e27",
				DiffIDs:   "sha256:6123adfc04c22915c112368b802af161b921fbf7ef1c5f7283191ee552b46e27",
				Size:      10,
				MediaType: "application/vnd.oci.image.layer.v1.tar",
			},
		},
	}

	v1Image, err := getV1Image(image)
	expected := v1.Image{
		OS: "linux",
		RootFS: v1.RootFS{
			DiffIDs: []digest.Digest{
				"sha256:6123adfc04c22915c112368b802af161b921fbf7ef1c5f7283191ee552b46e27"},
			Type: "layers",
		},
		History: []v1.History{
			{
				CreatedBy: "nix2container",
			},
		},
	}

	assert.Nil(t, err)
	assert.Equal(t, v1Image, expected)
}
