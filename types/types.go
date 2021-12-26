package types

import (
	"os"
	"encoding/json"
	"io/ioutil"
)

type Layer struct {
	Digest string `json:"digest"`
	Paths []string `json:"paths"`
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
