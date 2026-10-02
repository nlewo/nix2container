package closure

import (
	"encoding/json"
	"fmt"
	"os"
)

// SplitFromFile reads a layer split, a JSON list of store path lists in
// layer order, and checks it against the closure graph: every path of
// the graph must appear exactly once, and no other path may appear.
// The split can come from another tool, for instance the store_layers
// that nixpkgs' streamLayeredImage computes for the same closure.
func SplitFromFile(storepaths []Storepath, filename string) ([][]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var split [][]string
	if err := json.Unmarshal(data, &split); err != nil {
		return nil, fmt.Errorf("%s: %w", filename, err)
	}
	want := make(map[string]bool, len(storepaths))
	for _, sp := range storepaths {
		want[sp.Path] = true
	}
	seen := make(map[string]bool, len(storepaths))
	for i, group := range split {
		if len(group) == 0 {
			return nil, fmt.Errorf("%s: layer %d is empty", filename, i)
		}
		for _, p := range group {
			if !want[p] {
				return nil, fmt.Errorf("%s: layer %d lists %s, which is not in the closure graph", filename, i, p)
			}
			if seen[p] {
				return nil, fmt.Errorf("%s: %s is listed twice", filename, p)
			}
			seen[p] = true
		}
	}
	for _, sp := range storepaths {
		if !seen[sp.Path] {
			return nil, fmt.Errorf("%s: closure path %s is in no layer", filename, sp.Path)
		}
	}
	return split, nil
}
