package types

import (
	"os"
	"encoding/json"
	"io/ioutil"
)

type Layers struct {
}

type Layer struct {
	Digest string `json:"digest"`
	Paths []string `json:"paths"`
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
