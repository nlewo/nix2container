package types

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const ImageVersion = 1

// HealthConfig holds the Docker/Schema2 healthcheck configuration.
type HealthConfig struct {
	Test          []string      `json:"Test,omitempty"`
	Interval      time.Duration `json:"Interval,omitempty"`
	Timeout       time.Duration `json:"Timeout,omitempty"`
	StartPeriod   time.Duration `json:"StartPeriod,omitempty"`
	StartInterval time.Duration `json:"StartInterval,omitempty"`
	Retries       int           `json:"Retries,omitempty"`
}

type duration time.Duration

func (d *duration) UnmarshalJSON(b []byte) error {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		*d = duration(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		dur, err := time.ParseDuration(s)
		if err != nil {
			return err
		}
		*d = duration(dur)
		return nil
	}
	return fmt.Errorf("invalid duration: %s", string(b))
}

func (h *HealthConfig) UnmarshalJSON(b []byte) error {
	var raw struct {
		Test          []string `json:"Test,omitempty"`
		Interval      duration `json:"Interval,omitempty"`
		Timeout       duration `json:"Timeout,omitempty"`
		StartPeriod   duration `json:"StartPeriod,omitempty"`
		StartInterval duration `json:"StartInterval,omitempty"`
		Retries       int      `json:"Retries,omitempty"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	*h = HealthConfig{
		Test:          raw.Test,
		Interval:      time.Duration(raw.Interval),
		Timeout:       time.Duration(raw.Timeout),
		StartPeriod:   time.Duration(raw.StartPeriod),
		StartInterval: time.Duration(raw.StartInterval),
		Retries:       raw.Retries,
	}
	return nil
}

// ImageConfig embeds the standard OCI v1.ImageConfig and adds Healthcheck.
type ImageConfig struct {
	v1.ImageConfig
	Healthcheck *HealthConfig `json:"Healthcheck,omitempty"`
}

// Image represent the JSON image file produced by nix2container. This
// JSON file can then be used by the Skopeo Nix transport to actually
// build the container image.
type Image struct {
	Version     int         `json:"version"`
	ImageConfig ImageConfig `json:"image-config"`
	Layers      []Layer     `json:"layers"`
	Arch        string      `json:"arch"`
	Created     *time.Time  `json:"created"`
}

// MarshalJSON normalizes nil Layers to an empty slice so consumers of
// image.json never see `"layers": null`: a reader that indexes or
// iterates the layers (e.g. `jq '.layers[]'`, or a Python script doing
// `json["layers"]`) then works without a null guard. A nil Layers
// slice is reachable for a config-only image (no base image and no
// layer entries).
func (i Image) MarshalJSON() ([]byte, error) {
	type imageAlias Image // strip the method set to avoid recursion
	if i.Layers == nil {
		i.Layers = []Layer{}
	}
	return json.Marshal(imageAlias(i))
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
