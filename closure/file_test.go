package closure

import (
	"testing"
)

func TestReadClosureGraphFile(t *testing.T) {
	nodes, err := ReadClosureGraphFile("../data/closure-graph.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 5 {
		t.Fatalf("The graph should contain %d nodes (actual %d)", 9, len(nodes))
	}
}
