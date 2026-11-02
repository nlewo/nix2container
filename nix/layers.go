package nix

import (
	_ "crypto/sha256"
	_ "crypto/sha512"
	"reflect"
	"runtime"
	"sync"

	"github.com/nlewo/nix2container/types"
	godigest "github.com/opencontainers/go-digest"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

func getPaths(storePaths []string, parents []types.Layer, rewrites []types.RewritePath, exclude string, permPaths []types.PermPath, ignoreDestination string) types.Paths {
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
		if ignoreDestination != "" {
			hasPathOptions = true
			pathOptions.IgnoreDestination = ignoreDestination
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
//
// Layers are tarred concurrently, by up to GOMAXPROCS workers, and
// returned in the order of groups.
func newLayers(groups []types.Paths, tarDirectory string, history v1.History) ([]types.Layer, error) {
	layers := make([]types.Layer, len(groups))
	errs := make([]error, len(groups))
	jobs := make(chan int)
	var wg sync.WaitGroup
	// No min builtin nor range over int: skopeo vendors this package
	// with an old go directive.
	workers := runtime.GOMAXPROCS(0)
	if len(groups) < workers {
		workers = len(groups)
	}
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				layers[i], errs[i] = newLayer(groups[i], tarDirectory, history)
			}
		}()
	}
	for i := range groups {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return layers, nil
}

func newLayer(layerPaths types.Paths, tarDirectory string, history v1.History) (types.Layer, error) {
	layerPath := ""
	var digest godigest.Digest
	var size int64
	var err error
	if tarDirectory == "" {
		digest, size, err = TarPathsSum(layerPaths)
	} else {
		layerPath, digest, size, err = TarPathsWrite(layerPaths, tarDirectory)
	}
	if err != nil {
		return types.Layer{}, err
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
	return layer, nil
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
func splitGroups(split [][]string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, ignoreDestination string) (groups []types.Paths) {
	for _, storePaths := range split {
		paths := getPaths(storePaths, parents, rewrites, exclude, perms, ignoreDestination)
		if len(paths) > 0 {
			groups = append(groups, paths)
		}
	}
	return groups
}

func NewLayers(storePaths []string, maxLayers int, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, ignoreDestination string, history v1.History) ([]types.Layer, error) {
	paths := getPaths(storePaths, parents, rewrites, exclude, perms, ignoreDestination)
	return newLayers(maxLayersGroups(paths, maxLayers), "", history)
}

func NewLayersNonReproducible(storePaths []string, maxLayers int, tarDirectory string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, ignoreDestination string, history v1.History) (layers []types.Layer, err error) {
	paths := getPaths(storePaths, parents, rewrites, exclude, perms, ignoreDestination)
	return newLayers(maxLayersGroups(paths, maxLayers), tarDirectory, history)
}

// NewLayersFromSplit creates one layer per group of split, in order.
// The split is not checked against a closure: see closure.SplitFromFile.
func NewLayersFromSplit(split [][]string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, ignoreDestination string, history v1.History) ([]types.Layer, error) {
	return newLayers(splitGroups(split, parents, rewrites, exclude, perms, ignoreDestination), "", history)
}

func NewLayersNonReproducibleFromSplit(split [][]string, tarDirectory string, parents []types.Layer, rewrites []types.RewritePath, exclude string, perms []types.PermPath, ignoreDestination string, history v1.History) ([]types.Layer, error) {
	return newLayers(splitGroups(split, parents, rewrites, exclude, perms, ignoreDestination), tarDirectory, history)
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
