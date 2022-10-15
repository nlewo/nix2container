package nix

import (
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"
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
			Digest:  "sha256:6123adfc04c22915c112368b802af161b921fbf7ef1c5f7283191ee552b46e27",
			DiffIDs: "sha256:6123adfc04c22915c112368b802af161b921fbf7ef1c5f7283191ee552b46e27",
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
	assert.Equal(t, expected, layer)
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
			Digest:  "sha256:f2c0df36c223df52ef1ccc9d5979b39fb03fecae111f908fc9c2bdd50d477acd",
			DiffIDs: "sha256:f2c0df36c223df52ef1ccc9d5979b39fb03fecae111f908fc9c2bdd50d477acd",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
		},
	}
	assert.Equal(t, expected, layer)

	tmpDir := t.TempDir()
	layer, err = NewLayersNonReproducible(paths, 1, tmpDir, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected = []types.Layer{
		{
			Digest:  "sha256:f2c0df36c223df52ef1ccc9d5979b39fb03fecae111f908fc9c2bdd50d477acd",
			DiffIDs: "sha256:f2c0df36c223df52ef1ccc9d5979b39fb03fecae111f908fc9c2bdd50d477acd",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			LayerPath: tmpDir + "/f2c0df36c223df52ef1ccc9d5979b39fb03fecae111f908fc9c2bdd50d477acd.tar",
		},
	}
	assert.Equal(t, expected, layer)
}
