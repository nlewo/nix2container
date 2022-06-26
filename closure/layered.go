package closure

import (
	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// SortedPathsByPopularity sorts storepaths by path popularity. It uses the algorithm described in
// https://grahamc.com/blog/nix-and-layered-docker-images
func SortedPathsByPopularity(storepaths []Storepath) ([]string, error) {
	g := simple.NewDirectedGraph()
	paths := make(map[string]int64)
	for _, p := range storepaths {
		var u, v graph.Node
		if id, ok := paths[p.Path]; ok {
			u = g.Node(id)
		} else {
			u = g.NewNode()
			g.AddNode(u)
			paths[p.Path] = u.ID()
		}
		for _, r := range p.References {
			if id, ok := paths[r]; ok {
				v = g.Node(id)
			} else {
				v = g.NewNode()
				g.AddNode(v)
				paths[r] = v.ID()
			}
			if u == v {
				continue
			}
			g.SetEdge(g.NewEdge(u, v))
		}
	}
	scored, err := Score(g)
	if err != nil {
		return []string{}, err
	}

	out := make([]string, len(scored))
	for i, s := range scored {
		for p, id := range paths {
			if id == s.id {
				logrus.Debugf("Score: %d (%s)", s.score, p)
				out[i] = p
			}
		}
	}
	return out, nil
}
