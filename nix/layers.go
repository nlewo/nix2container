package nix

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"reflect"

	"github.com/nlewo/nix2container/types"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

func getPaths(storePaths []string, parents []types.Layer, rewrites []types.RewritePath, exclude string) types.Paths {
	var paths types.Paths
	for _, p := range storePaths {
		path := types.Path{
			Path: p,
		}
		for _, rewrite := range rewrites {
			if p == rewrite.Path {
				path.Options = &types.PathOptions{
					Rewrite: types.Rewrite{
						Regex: rewrite.Regex,
						Repl:  rewrite.Repl,
					},
				}
			}
		}
		if p == exclude || isPathInLayers(parents, path) {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func NewLayers(storePaths []string, parents []types.Layer, rewrites []types.RewritePath, exclude string) (layers []types.Layer, err error) {
	paths := getPaths(storePaths, parents, rewrites, exclude)
	d, err := TarPathsSum(paths)
	if err != nil {
		return layers, err
	}
	layers = []types.Layer{
		types.Layer{
			Digest:    d.String(),
			DiffIDs:    d.String(),
			Paths:     paths,
			MediaType: v1.MediaTypeImageLayer,
		},
	}
	return layers, nil
}

func NewLayersNonReproducible(storePaths []string, tarDirectory string, parents []types.Layer, rewrites []types.RewritePath, exclude string) (layers []types.Layer, err error) {
	paths := getPaths(storePaths, parents, rewrites, exclude)

	layerPath := tarDirectory + "/layer.tar"
	d, err := TarPathsWrite(paths, layerPath)
	if err != nil {
		return layers, err
	}
	layers = []types.Layer{
		types.Layer{
			Digest:    d.String(),
			DiffIDs:    d.String(),
			Paths:     paths,
			// TODO: we should use v1.MediaTypeImageLayerGzip instead
			MediaType: v1.MediaTypeImageLayer,
			LayerPath: layerPath,
		},
	}
	return layers, nil
}

func isPathInLayers(layers []types.Layer, path types.Path) bool {
	for _, layer := range layers {
		for _, p := range layer.Paths {
			if reflect.DeepEqual(p, path) {
				return true
			}
		}
	}
	return false
}
