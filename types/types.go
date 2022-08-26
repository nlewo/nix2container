package types

import (
	"encoding/json"
	"io/ioutil"
	"os"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

type Image struct {
	ImageConfig v1.ImageConfig `json:"image-config"`
	Layers      []Layer        `json:"layers"`
	Arch        string         `json:"arch"`
}

type Rewrite struct {
	Regex string `json:"regex"`
	Repl  string `json:"repl"`
}

// RewritePath describes how to replace the Regex in Path by the
// replacement Repl.
//
// This allows to rewrite storepaths during the tar operation. This is
// mainly used to move storepaths from the /nix/store to / in the
// image.
type RewritePath struct {
	Path  string `json:"path"`
	Regex string `json:"regex"`
	Repl  string `json:"repl"`
}

type Perm struct {
	Regex string `json:"regex"`
	// Octal representation of file permissions
	Mode string `json:"mode"`
}

type PermPath struct {
	Path  string `json:"path"`
	Regex string `json:"regex"`
	// Octal representation of file permissions
	Mode string `json:"mode"`
}

type PathOptions struct {
	Rewrite Rewrite `json:"rewrite,omitempty"`
	Perms   []Perm  `json:"perms,omitempty"`
}

type Path struct {
	Path    string       `json:"path"`
	Options *PathOptions `json:"options,omitempty"`
}

type Paths []Path

type Layer struct {
	Digest  string `json:"digest"`
	Size    int64  `json:"size"`
	DiffIDs string `json:"diff_ids"`
	Paths   Paths  `json:"paths,omitempty"`
	// OCI mediatype
	// https://github.com/opencontainers/image-spec/blob/8b9d41f48198a7d6d0a5c1a12dc2d1f7f47fc97f/specs-go/v1/mediatype.go
	MediaType string `json:"mediatype"`
	LayerPath string `json:"layer-path,omitempty"`
}

func NewLayersFromFile(filename string) ([]Layer, error) {
	var layers []Layer
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
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
