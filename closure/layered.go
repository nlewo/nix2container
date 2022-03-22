package closure

import (
	"sort"
)

type Node struct {
	Path string
	References []Node
	Popularities map[string]int
}

func buildGraph(storepaths []Storepath) (paths []Node) {
	roots := findRoots(storepaths)
	for _, r := range roots {
		paths = append(paths, buildGraphNode(r, storepaths))
	}
	return paths
}

func findRoots(storepaths []Storepath) (roots []Storepath) {
	for _, storepath := range storepaths {
		isRoot := true
		for _, storepath1 := range storepaths {
			if storepath.Path == storepath1.Path {
				continue
			}
			for _, ref := range storepath1.References {
				if storepath.Path == ref {
					isRoot = false
					break
				}
			}
		}
		if isRoot {
			roots = append(roots, storepath)
		}
	}
	return
}

func buildGraphNode(storepath Storepath, storepaths []Storepath) (node Node) {
	node.Path = storepath.Path
	node.Popularities = make(map[string]int)
	for _, r := range storepath.References {
		if r == node.Path {
			continue
		}
		for _, n := range storepaths {
			if n.Path == r {
				child := buildGraphNode(n, storepaths)
				node.References = append(
					node.References,
					child,
				)
				updatePopularities(node.Popularities, child.Popularities)
			}
		}
	}

	incrementPopularities(node.Popularities)

	_, exist := node.Popularities[node.Path]
	if exist {
		node.Popularities[node.Path] = node.Popularities[node.Path] + 1
	} else {
		node.Popularities[node.Path] = 1
	}
	return
}

func updatePopularities(node, child map[string]int) {
	for k, v := range child {
		_, exist := node[k]
		if exist {
			node[k] = node[k] + v

		} else {
			node[k] = v
		}
	}
}

func incrementPopularities(popularitites map[string]int) {
	for k, v := range popularitites {
		popularitites[k] = v + 1
	}
}

type Pair struct {
	Key   string
	Value int
}

type PairList []Pair

func (p PairList) Len() int           { return len(p) }
func (p PairList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p PairList) Less(i, j int) bool {
	if p[i].Value == p[j].Value {
		return p[i].Key < p[j].Key
	} else {
		return p[i].Value < p[j].Value
	}
}

// SortedPathByPopularity sorts storepaths by path popularity. It uses the algorithm described in
// https://grahamc.com/blog/nix-and-layered-docker-images
func SortedPathsByPopularity(storepaths []Storepath) []string {
	nodes := buildGraph(storepaths)
	popularities := make(map[string]int)
	for _, n := range nodes {
		updatePopularities(popularities, n.Popularities)
	}

	p := make(PairList, len(popularities))
	i := 0
	for k, v := range popularities {
		p[i] = Pair{k, v}
		i++
	}

	sort.Sort(p)

	var paths []string
	for i := len(p); i > 0; i-- {
		paths = append(paths, p[i-1].Key)
	}
	return paths
}
