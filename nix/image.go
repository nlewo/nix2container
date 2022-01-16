package nix

import (
	"io"
	"bytes"
	"errors"
	"encoding/json"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/nlewo/nix2container/types"
	godigest "github.com/opencontainers/go-digest"
)

// GetConfigBlob returns the config blog of an image.
func GetConfigBlob(image types.Image) ([]byte, error) {
	imageV1, err := getV1Image(image)
	if err != nil {
		return nil, err
	}
	configBlob, err := json.Marshal(imageV1)
	if err != nil {
		return nil, err
	}
	return configBlob, nil
}

// GetConfigDigest returns the digest of the config blog of an image.
func GetConfigDigest(image types.Image) (d godigest.Digest, err error) {
	configBlob, err := GetConfigBlob(image)
	if err != nil {
		return d, err
	}
	d = godigest.FromBytes(configBlob)
	return
}

// GetBlob gets the layer corresponding to the provided digest.
func GetBlob(image types.Image, digest godigest.Digest) (io.ReadCloser, error) {
	for _, layer := range(image.Layers) {
		if layer.Digest == digest.String() {
			rc, _, err := LayerGetBlob(layer)
			return rc, err
		}
	}
	configDigest, err := GetConfigDigest(image)
	if err != nil {
		return nil, err
	}
	if digest == configDigest {
		configBlob, err := GetConfigBlob(image)
		if err != nil {
			return nil, err
		}
		rc := nopCloser{bytes.NewReader(configBlob)}
		return rc, nil
	}
	return nil, errors.New("No blob with specified digest found in image")
}

func getV1Image(image types.Image) (imageV1 v1.Image, err error) {
	imageV1.OS = "linux"
	imageV1.Architecture = "amd64"
	imageV1.Config = image.ImageConfig

	for _, layer := range(image.Layers) {
		digest, err := godigest.Parse(layer.Digest)
		if err != nil {
			return imageV1, err
		}
		imageV1.RootFS.DiffIDs = append(
			imageV1.RootFS.DiffIDs,
			digest)
	}
	return
}

type nopCloser struct{
	io.Reader
}
func (nopCloser) Close() error { return nil }
