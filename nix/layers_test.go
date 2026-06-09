package nix

import (
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
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
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", perms, v1.History{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		{
			Digest:  "sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d",
			DiffIDs: "sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d",
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
	layer, err := NewLayers(paths, 1, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected := []types.Layer{
		{
			Digest:  "sha256:6c0a3ae4c7d4a4e950ca40e0a0c38d779c3d20f77f775cb61d9e43ebaab5583a",
			DiffIDs: "sha256:6c0a3ae4c7d4a4e950ca40e0a0c38d779c3d20f77f775cb61d9e43ebaab5583a",
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
	layer, err = NewLayersNonReproducible(paths, 1, tmpDir, []types.Layer{}, []types.RewritePath{}, "", []types.PermPath{}, v1.History{})
	if err != nil {
		t.Fatalf("%v", err)
	}
	expected = []types.Layer{
		{
			Digest:  "sha256:6c0a3ae4c7d4a4e950ca40e0a0c38d779c3d20f77f775cb61d9e43ebaab5583a",
			DiffIDs: "sha256:6c0a3ae4c7d4a4e950ca40e0a0c38d779c3d20f77f775cb61d9e43ebaab5583a",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			LayerPath: tmpDir + "/6c0a3ae4c7d4a4e950ca40e0a0c38d779c3d20f77f775cb61d9e43ebaab5583a.tar",
		},
	}
	assert.Equal(t, expected, layer)
}
