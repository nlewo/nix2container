package nix

import (
	"reflect"
	"testing"

	"github.com/nlewo/nix2container/types"
)

func TestNewLayers(t *testing.T) {
	paths := []string{
		"../data/layer1/file1",
	}
	layer, err := NewLayers(paths, []types.Layer{}, []types.RewritePath{}, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		types.Layer{
			Digest: "sha256:38856f8cd2e336497b6257e891ad860ea77e24193a726125445823618aa16cce",
			DiffIDs: "sha256:38856f8cd2e336497b6257e891ad860ea77e24193a726125445823618aa16cce",
			Size: 1536,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
		},
	}
	if !reflect.DeepEqual(layer, expected) {
		t.Fatalf("Layers should be '%#v' (while it is %#v)", expected, layer)
	}

	tmpDir := t.TempDir()
	layer, err = NewLayersNonReproducible(paths, tmpDir, []types.Layer{}, []types.RewritePath{}, "")
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected = []types.Layer{
		types.Layer{
			Digest: "sha256:38856f8cd2e336497b6257e891ad860ea77e24193a726125445823618aa16cce",
			DiffIDs: "sha256:38856f8cd2e336497b6257e891ad860ea77e24193a726125445823618aa16cce",
			Size: 1536,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			LayerPath: tmpDir + "/layer.tar",
		},
	}
	if !reflect.DeepEqual(layer, expected) {
		t.Fatalf("Layers should be '%#v' (while it is %#v)", expected, layer)
	}
}
