package types

import (
	"os"
	"encoding/json"
	"io/ioutil"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

type Image struct {
	ImageConfig v1.ImageConfig `json:"image-config"`
	Layers []Layer `json:"layers"`
}

type Layers struct {
}

type Layer struct {
	Digest string `json:"digest"`
	Paths []string `json:"paths,omitempty"`
	TarPath string `json:"tar-path,omitempty"`
}

func NewLayersFromFile(filename string) ([]Layer, error) {
	var layers []Layer
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, &layers)
	if err != nil {
		return nil, err
	}
	return layers, nil
}

func NewImageFromFile(filename string) (image Image, err error) {
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
