package nix

import (
	"reflect"
	"testing"

	"github.com/nlewo/nix2container/types"
)

func TestPerms(t *testing.T) {
	paths := []string{
		"../data/layer1/file1",
	}
	perms := []types.PermPath{
		types.PermPath{
			Path: "../data/layer1/file1",
			Regex: ".*file1",
			Mode: "0641",
		},
	}
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", perms)
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		types.Layer{
			Digest: "sha256:1d93274b1a59eed0a471f601a7f87ae58e2860566ed20313043b1efd983e8baa",
			DiffIDs: "sha256:1d93274b1a59eed0a471f601a7f87ae58e2860566ed20313043b1efd983e8baa",
			Size: 1536,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
					Options: &types.PathOptions{
						Perms: []types.Perm{
							types.Perm{
								Regex: ".*file1",
								Mode: "0641",
							},
						},
					},
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
		},
	}
	if !reflect.DeepEqual(layer, expected) {
		t.Fatalf("Layers should be '%#v' (while it is %#v)", expected, layer)
	}
}

func TestNewLayers(t *testing.T) {
	paths := []string{
		"../data/layer1/file1",
	}
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{})
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
	layer, err = NewLayersNonReproducible(paths, 1, tmpDir, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{})
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
			LayerPath: tmpDir + "/38856f8cd2e336497b6257e891ad860ea77e24193a726125445823618aa16cce.tar",
		},
	}
	if !reflect.DeepEqual(layer, expected) {
		t.Fatalf("Layers should be '%#v' (while it is %#v)", expected, layer)
	}
}
