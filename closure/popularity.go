package closure

import (
	"sort"

	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
)

type ScoredNode struct {
	id    int64
	score int64
}

func Score(g graph.Directed) ([]ScoredNode, error) {

	nodes, err := topo.Sort(g)
	if err != nil {
		return nil, err
	}

	// Make a topological copy of g with dense node IDs.
	indexOf := make(map[int64]int, len(nodes))
	for i, n := range nodes {
		indexOf[n.ID()] = i
	}
	nodesLinkedFrom := make([][]int, len(nodes))
	for i, n := range nodes {
		id := n.ID()
		to := g.From(id)
		for to.Next() {
			v := to.Node()
			nodesLinkedFrom[i] = append(nodesLinkedFrom[i], indexOf[v.ID()])
		}
	}

	pop := make([]int64, len(nodes))
	for i := range nodes {
		pop[i] = 1
	}

	for v := range nodes {
		for _, u := range nodesLinkedFrom[v] {
			pop[u] += pop[v]
		}
	}

	scored := make([]ScoredNode, len(nodes))
	for i, n := range nodes {
		scored[i] = ScoredNode{n.ID(), pop[i]}
	}
	// sort descending ...
	// nodes is unstably sorted, so we explicitly rely on node id in case of popularity peers
	// this results in emanent stable sorting, so we don't need sort.SliceStable (which is less efficient)
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score || (scored[i].score == scored[j].score && scored[i].id > scored[j].id)
	})
	return scored, nil
}
