package nix

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"
)

func TestGraph(t *testing.T) {
	g := initGraph()
	err := addFileToGraph(g, "/nix", nil, nil)
	assert.Equal(t, nil, err)
	assert.Contains(t, g.contents, "")
	assert.Contains(t, g.contents[""].contents, "nix")

	g = initGraph()
	err = addFileToGraph(g, "/nix/store/hash1", nil, nil)
	assert.Equal(t, nil, err)
	assert.Contains(t, g.contents, "")
	assert.Contains(t, g.contents[""].contents, "nix")
	assert.Contains(t, g.contents[""].contents["nix"].contents, "store")
	assert.Contains(t, g.contents[""].contents["nix"].contents["store"].contents, "hash1")
	err = addFileToGraph(g, "/nix/store/hash2", nil, nil)
	assert.Equal(t, nil, err)
	assert.Contains(t, g.contents, "")
	assert.Contains(t, g.contents[""].contents, "nix")
	assert.Contains(t, g.contents[""].contents["nix"].contents["store"].contents, "hash1")
	assert.Contains(t, g.contents[""].contents["nix"].contents["store"].contents, "hash2")
}

func TestGraphSanitized(t *testing.T) {
	g := initGraph()
	_ = addFileToGraph(g, "/nix/store/hash1", nil, nil)
	_ = addFileToGraph(g, "/Nix~nix~case~hack~1/store/hash1", nil, nil)
	_ = addFileToGraph(g, "/Nix~nix~case~hack~0/store/hash1", nil, nil)
	_ = addFileToGraph(g, "/nix/store/Path~nix~case~hack~1", nil, nil)
	err := sanitizeGraph(g)
	assert.Equal(t, nil, err)
	assert.Contains(t, g.contents[""].contents["nix"].contents["store"].contents, "hash1")
	assert.Contains(t, g.contents[""].contents["Nix"].contents["store"].contents, "hash1")
	assert.Contains(t, g.contents[""].contents["Nix~nix~case~hack~0"].contents["store"].contents, "hash1")
	assert.Contains(t, g.contents[""].contents["nix"].contents["store"].contents, "Path")
}

func TestGraphSanitizeCollision(t *testing.T) {
	g := initGraph()
	_ = addFileToGraph(g, "/nix/store/path/hash1", nil, nil)
	_ = addFileToGraph(g, "/nix/store/path~nix~case~hack~1/hash1", nil, nil)
	err := sanitizeGraph(g)
	assert.ErrorContains(t, err, "collision")
}

func TestAddFileToGraphOverride(t *testing.T) {
	g := initGraph()
	err := addFileToGraph(g, "/nix/store/file1", nil, &types.PathOptions{
		Perms: []types.Perm{
			{
				Regex: "*",
				Uid:   1,
			},
		},
	})
	assert.Equal(t, nil, err)
	err = addFileToGraph(g, "/nix/store/file1", nil, &types.PathOptions{
		Perms: []types.Perm{
			{
				Regex: "*",
				Uid:   2,
			},
		},
	})
	assert.Error(t, err)
}

func TestWalkGraph(t *testing.T) {
	g := initGraph()
	paths := make([]string, 5)
	var idx int
	pidx := &idx
	err := addFileToGraph(g, "/nix/store/hash2", nil, nil)
	assert.Equal(t, nil, err)
	err = addFileToGraph(g, "/nix/store/hash1", nil, nil)
	assert.Equal(t, nil, err)
	err = sanitizeGraph(g)
	assert.Equal(t, nil, err)
	err = walkGraph(g, func(srcPath, dstPath string, info *os.FileInfo, options *types.PathOptions) error {
		paths[*pidx] = dstPath
		*pidx = *pidx + 1
		return nil
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, "/", paths[0])
	assert.Equal(t, "/nix", paths[1])
	assert.Equal(t, "/nix/store", paths[2])
	assert.Equal(t, "/nix/store/hash1", paths[3])
	assert.Equal(t, "/nix/store/hash2", paths[4])
}

func TestWalkGraphOnDirectory(t *testing.T) {
	graph := initGraph()
	err := filepath.Walk("../data/graph-directory",
		func(path string, info os.FileInfo, err error) error {
			return addFileToGraph(graph, path, &info, nil)
		},
	)
	assert.Equal(t, nil, err)
	dstPaths := make([]string, 10)
	srcPaths := make([]string, 10)
	missingDirectories := make([]string, 10)
	var idx int
	pidx := &idx
	err = sanitizeGraph(graph)
	assert.Equal(t, nil, err)
	err = walkGraph(graph, func(srcPath, dstPath string, info *os.FileInfo, options *types.PathOptions) error {
		dstPaths[*pidx] = dstPath
		srcPaths[*pidx] = srcPath
		if info == nil {
			missingDirectories[*pidx] = dstPath
		}
		*pidx = *pidx + 1
		return nil
	})
	assert.Equal(t, nil, err)
	assert.Equal(t, "..", dstPaths[0])
	assert.Equal(t, "../data", dstPaths[1])
	assert.Equal(t, "../data/graph-directory", dstPaths[2])
	assert.Equal(t, "../data/graph-directory/path1", dstPaths[3])
	assert.Equal(t, "../data/graph-directory/path1/path11", dstPaths[4])
	assert.Equal(t, "../data/graph-directory/path1/path11/file111", dstPaths[5])
	assert.Equal(t, "../data/graph-directory/path2", dstPaths[6])
	assert.Equal(t, "../data/graph-directory/path2/file21", dstPaths[7])

	assert.Equal(t, "", srcPaths[0])
	assert.Equal(t, "", srcPaths[1])
	assert.Equal(t, "../data/graph-directory", srcPaths[2])
	assert.Equal(t, "../data/graph-directory/path1", srcPaths[3])
	assert.Equal(t, "../data/graph-directory/path1/path11", srcPaths[4])
	assert.Equal(t, "../data/graph-directory/path1/path11/file111", srcPaths[5])

	assert.Equal(t, "..", missingDirectories[0])
	assert.Equal(t, "../data", missingDirectories[1])
	assert.Equal(t, "", missingDirectories[2])
}
