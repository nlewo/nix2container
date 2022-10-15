package nix

import (
	"fmt"
	"github.com/nlewo/nix2container/types"
	"os"
	"path/filepath"
	"reflect"
	"sort"
)

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
	pathInTar := filePathToTarPath(path, options)
	// A regex in the options could make the path becoming the
	// empty string. In this case, we don't want to create
	// anything in the graph.
	if pathInTar == "" {
		return nil
	}

	parts := splitPath(pathInTar)
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

	if current.srcPath != "" && current.srcPath != path {
		return fmt.Errorf("The file '%s' already exists in the tar with source path %s but is added again with the source path %s",
			pathInTar, current.srcPath, path)
	}
	current.srcPath = path

	if current.options != nil && !reflect.DeepEqual(current.options, options) {
		return fmt.Errorf("The file '%s' already exists in the tar with options %#v but is overriden with options %#v",
			pathInTar, current.options, options)
	}
	current.options = options

	current.info = info
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
