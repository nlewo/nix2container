package nix

import (
	"path/filepath"
	"sort"
	"strings"
)

type Node struct {
	inTar    bool
	children map[string]*Node
}

// addPathInGraph adds a file and all missing file parents file
// in the graph.
// It also marks nodes that are in the tar.
func addPathInGraph(root *Node, path string) {
	var current *Node
	var exist bool
	components := strings.Split(filepath.Clean(path), "/")
	if components[0] == "" {
		components[0] = "/"
	}
	node := root
	for _, component := range components {
		current, exist = node.children[component]
		if !exist {
			current = &Node{
				children: make(map[string]*Node),
			}
			node.children[component] = current
		}
		node = current
	}
	// This node exists in the tar
	current.inTar = true
}

func collectPathsNotInTar(root *Node, base string) []string {
	collected := []string{}
	for name, child := range root.children {
		path := filepath.Join(base, name)
		if !child.inTar {
			collected = append(collected, path)
		}
		collected = append(collected, collectPathsNotInTar(child, path)...)
	}
	return collected
}

// pathsNotInTar returns all paths that have not been explicitly
// created. If the paths list only contains "/nix/store/hash", this
// function returns ["/nix", "/nix/store"].  Note we don't want to
// just create "/nix" and "/nix/store" directories because we want to be
// "nix" specific (think of guix for instance).
func pathsNotInTar(paths []string) []string {
	root := Node{
		children: make(map[string]*Node),
	}
	for _, path := range paths {
		addPathInGraph(&root, path)
	}
	collected := collectPathsNotInTar(&root, "")
	sort.Strings(collected)
	return collected
}
