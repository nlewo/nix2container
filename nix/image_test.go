package nix

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	digest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"

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

// Legacy image configurations may carry Cmd and Entrypoint as plain
// strings. Only the rootfs section is needed, so they must not fail.
func TestNewImageFromDirStringCmd(t *testing.T) {
	dir := t.TempDir()
	config := []byte(`{
  "architecture": "amd64",
  "os": "linux",
  "config": {"Cmd": "/bin/sh -c true", "Entrypoint": "/bin/sh"},
  "rootfs": {"type": "layers", "diff_ids": ["sha256:8d3ac3489996423f53d6087c81180006263b79f206d3fdec9e66f0e27ceb8759"]}
}`)
	configDigest := digest.FromBytes(config)
	if err := os.WriteFile(filepath.Join(dir, configDigest.Encoded()), config, 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := []byte(`{
  "schemaVersion": 2,
  "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
  "config": {"mediaType": "application/vnd.docker.container.image.v1+json", "size": ` + fmt.Sprint(len(config)) + `, "digest": "` + configDigest.String() + `"},
  "layers": [{"mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip", "size": 1, "digest": "sha256:59bf1c3509f33515622619af21ed55bbe26d24913cedbca106468a5fb37a50c3"}]
}`)
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o644); err != nil {
		t.Fatal(err)
	}

	image, err := NewImageFromDir(dir)
	if err != nil {
		t.Fatalf("%v", err)
	}
	assert.Len(t, image.Layers, 1)
	assert.Equal(t, "sha256:8d3ac3489996423f53d6087c81180006263b79f206d3fdec9e66f0e27ceb8759", image.Layers[0].DiffIDs)
}

func TestGetV1Image(t *testing.T) {
	image := types.Image{
		Layers: []types.Layer{
			{
				Digest:    "sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d",
				DiffIDs:   "sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d",
				Size:      10,
				MediaType: "application/vnd.oci.image.layer.v1.tar",
				History: v1.History{
					CreatedBy: "nix2container",
				},
			},
		},
	}

	v1Image, err := getV1Image(image)
	expected := v1.Image{
		Platform: v1.Platform{OS: "linux"},
		RootFS: v1.RootFS{
			DiffIDs: []digest.Digest{
				"sha256:adf74a52f9e1bcd7dab77193455fa06743b979cf5955148010e5becedba4f72d"},
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
