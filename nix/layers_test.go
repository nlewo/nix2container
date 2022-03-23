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
		{
			Path:  "../data/layer1/file1",
			Regex: ".*file1",
			Mode:  "0641",
		},
	}
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", perms)
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		{
			Digest:  "sha256:7031b24697abf372b252fffb1432f685b364b742212df74787e2a2a8c8d4f66f",
			DiffIDs: "sha256:7031b24697abf372b252fffb1432f685b364b742212df74787e2a2a8c8d4f66f",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
					Options: &types.PathOptions{
						Perms: []types.Perm{
							{
								Regex: ".*file1",
								Mode:  "0641",
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
		{
			Digest:  "sha256:a97d8eab8c8b698b1c5aa10625b30b3b47baf102d1c429d567023a05ebe53480",
			DiffIDs: "sha256:a97d8eab8c8b698b1c5aa10625b30b3b47baf102d1c429d567023a05ebe53480",
			Size:    3072,
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
		{
			Digest:  "sha256:a97d8eab8c8b698b1c5aa10625b30b3b47baf102d1c429d567023a05ebe53480",
			DiffIDs: "sha256:a97d8eab8c8b698b1c5aa10625b30b3b47baf102d1c429d567023a05ebe53480",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			LayerPath: tmpDir + "/a97d8eab8c8b698b1c5aa10625b30b3b47baf102d1c429d567023a05ebe53480.tar",
		},
	}
	if !reflect.DeepEqual(layer, expected) {
		t.Fatalf("Layers should be '%#v' (while it is %#v)", expected, layer)
	}
}
