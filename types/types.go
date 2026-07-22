package types

import (
	"encoding/json"
	"io"
	"os"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const ImageVersion = 1

// Image represent the JSON image file produced by nix2container. This
// JSON file can then be used by the Skopeo Nix transport to actually
// build the container image.
type Image struct {
	Version     int            `json:"version"`
	ImageConfig v1.ImageConfig `json:"image-config"`
	Layers      []Layer        `json:"layers"`
	Arch        string         `json:"arch"`
	Created     *time.Time     `json:"created"`
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
	// Octal permission bits to OR into the existing mode (e.g. "0200"
	// to add u+w). Applied after Mode, so {Mode:"0444", OrMode:"0200"}
	// yields 0644. Lets a single perms entry express "make writable"
	// over a tree whose files are a mix of 0444/0555 (nix-store
	// canonicalization) without flattening the execute bit.
	OrMode string `json:"orMode,omitempty"`
	Uid    int    `json:"uid"`
	Gid    int    `json:"gid"`
	Uname  string `json:"uname"`
	Gname  string `json:"gname"`
}

type PermPath struct {
	Path  string `json:"path"`
	Regex string `json:"regex"`
	// Octal representation of file permissions
	Mode string `json:"mode"`
	// See Perm.OrMode.
	OrMode string `json:"orMode,omitempty"`
	Uid    int    `json:"uid"`
	Gid    int    `json:"gid"`
	Uname  string `json:"uname"`
	Gname  string `json:"gname"`
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
	History   v1.History
}

func NewLayersFromFile(filename string) ([]Layer, error) {
	var layers []Layer
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close() // nolint: errcheck
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(content, &layers)
	if err != nil {
		return nil, err
	}
	return layers, nil
}
