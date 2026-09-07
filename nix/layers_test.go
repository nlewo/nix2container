package nix

import (
	"archive/tar"
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
			Digest:  "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			DiffIDs: "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
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
			Digest:  "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			DiffIDs: "sha256:cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e",
			Size:    3072,
			Paths: types.Paths{
				types.Path{
					Path: "../data/layer1/file1",
				},
			},
			MediaType: "application/vnd.oci.image.layer.v1.tar",
			LayerPath: tmpDir + "/cc45bd46eca903b0900ebb997dffd5778904dca9ec02e7375dd1e653dfb61e2e.tar",
		},
	}
	assert.Equal(t, expected, layer)
}

// OrMode must OR bits into the mode set by Mode: {Mode:"0444",
// OrMode:"0311"} yields 0755 on a regular file.
func TestPermsOrMode(t *testing.T) {
	paths := []string{"../data/layer1/file1"}
	perms := []types.PermPath{
		{Path: "../data/layer1/file1", Regex: ".*", Mode: "0444", OrMode: "0311"},
	}
	layers, err := NewLayers(paths, 1, nil, nil, "", perms, v1.History{})
	if err != nil {
		t.Fatal(err)
	}
	r := TarPaths(layers[0].Paths)
	defer r.Close() // nolint: errcheck
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if hdr.Name == "../data/layer1/file1" && hdr.Typeflag == tar.TypeReg {
			assert.Equal(t, int64(0o755), hdr.Mode&0o777)
			return
		}
	}
	t.Fatal("file1 not found in layer tar")
}
