package nix

import (
	"fmt"
	"github.com/nlewo/nix2container/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

// On case insensitive FS (adfs on MacOS for instance), Nix adds a
// suffix to avoid filename collisions.
// See https://github.com/NixOS/nix/blob/ba9e69cdcd8022f37e344f2c86e60ee2b9da493f/src/libutil/archive.cc#L90
var useNixCaseHack string

type fileNode struct {
	// The file name on the FS
	srcPath  string
	info     *os.FileInfo
	options  *types.PathOptions
	contents map[string]*fileNode
}

func initGraph() *fileNode {
	root := &fileNode{
		contents: make(map[string]*fileNode),
	}
	return root
}

// addFileToGraph adds a file to the graph. A node of the graph
// represent a file. When addding a file, all parent directories of
// this file are added in the graph.
//
// The info and options are added to the node representing the file
// itself, ie. the leaf node.
//
// Note the graph describes the file tree of the tar stream, not the
// file tree read on the FS. This means transformations are done during
// the graph construction.
func addFileToGraph(root *fileNode, path string, info *os.FileInfo, options *types.PathOptions) error {

	dstPath := path
	if useNixCaseHack != "" {
		dstPath = removeNixCaseHackSuffix(dstPath)
	}

	dstPath = filePathToTarPath(dstPath, options)
	// A regex in the options could make the path becoming the
	// empty string. In this case, we don't want to create
	// anything in the graph.
	if dstPath == "" {
		return nil
	}

	parts := splitPath(dstPath)
	current := root
	for _, part := range parts {
		if node, exists := current.contents[part]; exists {
			current = node
		} else {
			current.contents[part] = &fileNode{
				contents: make(map[string]*fileNode),
			}
			current = current.contents[part]
		}
	}

	if current.info != nil {
		if (*current.info).Mode() != (*info).Mode() {
			return fmt.Errorf("The file '%s' already exists in the graph with mode '%v' from '%s' while it is added again with mode '%v' by '%s'",
				dstPath, (*current.info).Mode(), current.srcPath, (*info).Mode(), path)
		}
		// .Size() is only meaningful for regular files
		// See https://pkg.go.dev/io/fs#FileInfo
		if (*current.info).Mode().IsRegular() && (*current.info).Size() != (*info).Size() {
			return fmt.Errorf("The file '%s' already exists in the graph with size '%d' from '%s' while it is added again with size '%d' by '%s'",
				dstPath, (*current.info).Size(), current.srcPath, (*info).Size(), path)
		}
	}
	current.info = info

	if current.options != nil && !reflect.DeepEqual(current.options.Perms, options.Perms) {
		return fmt.Errorf("The file '%s' already exists in the tar with perms %#v but is overridden with perms %#v",
			dstPath, current.options.Perms, options.Perms)
	}
	current.options = options

	current.srcPath = path
	return nil
}

// If info is nil, dstPath is then a directory: this directory has
// been added to the graph but has not been walk by
// filepath.Walk. This for instance occurs when /nix/store/storepath1
// is added: /nix/store is not walk by the filepath.Walk function.
type walkFunc func(srcPath, dstPath string, info *os.FileInfo, options *types.PathOptions) error

func walkGraph(root *fileNode, walkFn walkFunc) error {
	return walkGraphFn("", root, walkFn)
}

func walkGraphFn(base string, root *fileNode, walkFn walkFunc) error {
	keys := make([]string, len(root.contents))
	i := 0
	for k := range root.contents {
		keys[i] = k
		i++
	}
	// Each subdirectory is sorted to avoid depending on the
	// source file name order: we instead want to order file based
	// on the name they have in the tar stream.
	sort.Strings(keys)

	for _, k := range keys {
		dstPath := filepath.Join(base, k)
		if k == "" {
			dstPath = filepath.Join("/", k)
		}
		if err := walkFn(root.contents[k].srcPath, dstPath, root.contents[k].info, root.contents[k].options); err != nil {
			return err
		}
		if err := walkGraphFn(dstPath, root.contents[k], walkFn); err != nil {
			return err
		}
	}
	return nil
}
