package nix

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"reflect"

	"github.com/nlewo/nix2container/types"
	godigest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

// LayerOptions is what shapes the paths of a layer beyond the store
// paths themselves. A zero value gives a plain layer.
type LayerOptions struct {
	// Layers whose paths are left out of this one.
	Parents []types.Layer
	// Path rewrites, to move a store path's content elsewhere in the image.
	Rewrites []types.RewritePath
	// A store path to leave out, even when it is in the closure.
	Exclude string
	// Ownership and mode overrides.
	Perms []types.PermPath
}

func getPaths(storePaths []string, o LayerOptions) types.Paths {
	parents, rewrites, exclude, permPaths := o.Parents, o.Rewrites, o.Exclude, o.Perms
	var paths types.Paths
	for _, p := range storePaths {
		path := types.Path{
			Path: p,
		}
		var pathOptions types.PathOptions
		hasPathOptions := false
		var perms []types.Perm
		for _, perm := range permPaths {
			if p == perm.Path {
				hasPathOptions = true
				perms = append(perms, types.Perm{
					Regex:  perm.Regex,
					Mode:   perm.Mode,
					OrMode: perm.OrMode,
					Uid:    perm.Uid,
					Gid:    perm.Gid,
					Uname:  perm.Uname,
					Gname:  perm.Gname,
				})
			}
		}
		if perms != nil {
			pathOptions.Perms = perms
		}
		for _, rewrite := range rewrites {
			if p == rewrite.Path {
				hasPathOptions = true
				pathOptions.Rewrite = types.Rewrite{
					Regex: rewrite.Regex,
					Repl:  rewrite.Repl,
				}
			}
		}
		if hasPathOptions {
			path.Options = &pathOptions
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

// If tarDirectory is not an empty string, the tar layer is written to
// the disk. This is useful for layer containing non reproducible
// store paths.
func newLayers(groups []types.Paths, tarDirectory string, history v1.History) (layers []types.Layer, err error) {
	for _, layerPaths := range groups {
		layerPath := ""
		var digest godigest.Digest
		var size int64
		if tarDirectory == "" {
			digest, size, err = TarPathsSum(layerPaths)
		} else {
			layerPath, digest, size, err = TarPathsWrite(layerPaths, tarDirectory)
		}
		if err != nil {
			return layers, err
		}
		logrus.Infof("Adding %d paths to layer (size:%d digest:%s)", len(layerPaths), size, digest.String())
		layer := types.Layer{
			Digest:    digest.String(),
			DiffIDs:   digest.String(),
			Size:      size,
			Paths:     layerPaths,
			MediaType: v1.MediaTypeImageLayer,
			History:   history,
		}
		if tarDirectory != "" {
			// TODO: we should use v1.MediaTypeImageLayerGzip instead
			layer.MediaType = v1.MediaTypeImageLayer
			layer.LayerPath = layerPath
		}

		layers = append(layers, layer)
	}
	return layers, nil
}

// maxLayersGroups cuts paths into at most maxLayers groups: one path
// per group, in order, and every remaining path in the last group.
func maxLayersGroups(paths types.Paths, maxLayers int) (groups []types.Paths) {
	offset := 0
	for offset < len(paths) {
		max := offset + 1
		if offset == maxLayers-1 {
			max = len(paths)
		}
		groups = append(groups, paths[offset:max])
		offset = max
	}
	return groups
}

// splitGroups filters each group like getPaths does and drops the
// groups left empty, for instance when all their paths are already in
// a parent layer.
func splitGroups(split [][]string, o LayerOptions) (groups []types.Paths) {
	for _, storePaths := range split {
		paths := getPaths(storePaths, o)
		if len(paths) > 0 {
			groups = append(groups, paths)
		}
	}
	return groups
}

// NewLayersWithOptions builds the layers of storePaths, at most
// maxLayers of them, shaped by o.
func NewLayersWithOptions(storePaths []string, maxLayers int, o LayerOptions, history v1.History) ([]types.Layer, error) {
	return newLayers(maxLayersGroups(getPaths(storePaths, o), maxLayers), "", history)
}

// NewLayersNonReproducibleWithOptions is NewLayersWithOptions with each
// layer tar written to tarDirectory.
func NewLayersNonReproducibleWithOptions(storePaths []string, maxLayers int, tarDirectory string, o LayerOptions, history v1.History) ([]types.Layer, error) {
	return newLayers(maxLayersGroups(getPaths(storePaths, o), maxLayers), tarDirectory, history)
}

// NewLayersFromSplitWithOptions creates one layer per group of split,
// in order, shaped by o. The split is not checked against a closure:
// see closure.SplitFromFile.
func NewLayersFromSplitWithOptions(split [][]string, o LayerOptions, history v1.History) ([]types.Layer, error) {
	return newLayers(splitGroups(split, o), "", history)
}

// NewLayersNonReproducibleFromSplitWithOptions is
// NewLayersFromSplitWithOptions with each layer tar written to
// tarDirectory.
func NewLayersNonReproducibleFromSplitWithOptions(split [][]string, tarDirectory string, o LayerOptions, history v1.History) ([]types.Layer, error) {
	return newLayers(splitGroups(split, o), tarDirectory, history)
}

func NewLayers(storePaths []string, maxLayers int, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, history v1.History) ([]types.Layer, error) {
	return NewLayersWithOptions(storePaths, maxLayers, LayerOptions{Parents: parents, Rewrites: rewrites, Exclude: exclude, Perms: perms}, history)
}

func NewLayersNonReproducible(storePaths []string, maxLayers int, tarDirectory string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, history v1.History) (layers []types.Layer, err error) {
	return NewLayersNonReproducibleWithOptions(storePaths, maxLayers, tarDirectory, LayerOptions{Parents: parents, Rewrites: rewrites, Exclude: exclude, Perms: perms}, history)
}

// NewLayersFromSplit creates one layer per group of split, in order.
// The split is not checked against a closure: see closure.SplitFromFile.
func NewLayersFromSplit(split [][]string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, history v1.History) ([]types.Layer, error) {
	return NewLayersFromSplitWithOptions(split, LayerOptions{Parents: parents, Rewrites: rewrites, Exclude: exclude, Perms: perms}, history)
}

func NewLayersNonReproducibleFromSplit(split [][]string, tarDirectory string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, history v1.History) ([]types.Layer, error) {
	return NewLayersNonReproducibleFromSplitWithOptions(split, tarDirectory, LayerOptions{Parents: parents, Rewrites: rewrites, Exclude: exclude, Perms: perms}, history)
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
