package nix

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"reflect"

	"github.com/nlewo/nix2container/types"
	"github.com/sirupsen/logrus"
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
		if p == exclude {
			logrus.Infof("Excluding path %s from layer", p)
			continue
		}
		if isPathInLayers(parents, path) {
			logrus.Infof("Excluding path %s because already present in a parent layer", p)
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func NewLayers(storePaths []string, parents []types.Layer, rewrites []types.RewritePath, exclude string) (layers []types.Layer, err error) {
	paths := getPaths(storePaths, parents, rewrites, exclude)
	d, s, err := TarPathsSum(paths)
	logrus.Infof("Adding %d paths to layer (size:%d digest:%s)", len(paths), s, d.String())
	if err != nil {
		return layers, err
	}
	layers = []types.Layer{
		types.Layer{
			Digest:    d.String(),
			DiffIDs:    d.String(),
			Size: s,
			Paths:     paths,
			MediaType: v1.MediaTypeImageLayer,
		},
	}
	return layers, nil
}

func NewLayersNonReproducible(storePaths []string, tarDirectory string, parents []types.Layer, rewrites []types.RewritePath, exclude string) (layers []types.Layer, err error) {
	paths := getPaths(storePaths, parents, rewrites, exclude)

	layerPath := tarDirectory + "/layer.tar"
	d, s, err := TarPathsWrite(paths, layerPath)
	logrus.Infof("Adding %d paths to layer (size:%d digest:%s)", len(paths), s, d.String())
	if err != nil {
		return layers, err
	}
	layers = []types.Layer{
		types.Layer{
			Digest:    d.String(),
			DiffIDs:    d.String(),
			Size: s,
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
