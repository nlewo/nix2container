package nix

import (
	"io"
	"os"
	"io/ioutil"
	"bytes"
	"errors"
	"encoding/json"
	"github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/nlewo/nix2container/types"
	"github.com/sirupsen/logrus"
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

func NewImageFromFile(filename string) (image types.Image, err error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return image, err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return image, err
	}
	err = json.Unmarshal(content, &image)
	if err != nil {
		return image, err
	}
	return image, nil
}

// NewImageFromDir builds an Image based on an directory populated by
// the Skopeo dir transport. The directory needs to be a absolute
// path since tarball filepaths are referenced in the image Layers.
func NewImageFromDir(directory string) (image types.Image, err error) {
	manifestFile, err := os.Open(directory + "/manifest.json")
	defer manifestFile.Close()
	if err != nil {
		return image, err
	}
	content, err := ioutil.ReadAll(manifestFile)
	if err != nil {
		return image, err
	}
	var manifest v1.Manifest
	err = json.Unmarshal(content, &manifest)
	if err != nil {
		return image, err
	}
	
	// TODO: we should also load the configuration of the image.

	for _, l := range manifest.Layers {
		layerFilename := directory + "/" + l.Digest.Encoded()
		logrus.Infof("Adding tar file '%s' as image layer", layerFilename)
		image.Layers = append(image.Layers, types.Layer{
			TarPath: layerFilename,
			Digest: l.Digest.String(),
		})
	}
	return image, nil
}



type nopCloser struct{
	io.Reader
}
func (nopCloser) Close() error { return nil }
