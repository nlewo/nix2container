package closure

import (
	"testing"
	"reflect"
)

func TestFindRoots(t *testing.T) {
	nodes, err := ReadClosureGraphFile("../data/closure-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	roots := findRoots(nodes)
	if len(roots) != 2 {
		t.Fatalf("The graph should contain %d root nodes (actual %d)", 2, len(roots))
	}

}

func TestBuildGraph(t *testing.T) {
	nodes, err := ReadClosureGraphFile("../data/closure-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	graph := buildGraph(nodes)
	if len(graph) != 2 {
		t.Fatalf("The graph should contain %d root nodes (actual %d)", 2, len(graph))
	}
}

// A - B - C - D - F
//  \   \   \
//   \   \   \- E - F
//    \   \
//     \   \- E - F
//      \
//       \- G
func TestPopularities(t *testing.T) {
	storepaths := []Storepath{
		Storepath{
			Path: "A",
			References: []string{"A", "B", "G"},
		},
		Storepath{
			Path: "B",
			References: []string{"B", "C", "E"},
		},
		Storepath{
			Path: "C",
			References: []string{"C", "D", "E"},
		},
		Storepath{
			Path: "D",
			References: []string{"D", "F"},
		},
		Storepath{
			Path: "E",
			References: []string{"E", "F"},
		},
		Storepath{
			Path: "F",
			References: []string{"F"},
		},
		Storepath{
			Path: "G",
			References: []string{"G"},
		},
	}
	popularity := SortedPathsByPopularity(storepaths)
	expected := []string{"F", "E", "D", "C", "G", "B", "A"}
	if !reflect.DeepEqual(popularity, expected) {
		t.Fatalf("Popularity should be '%#v' (while it is %#v)", expected, popularity)
	}
}
