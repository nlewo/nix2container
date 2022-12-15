package closure

import (
	"reflect"
	"testing"
)

// The graph looks like:
//
// --- A - B - C - D - F
//
//	\   \   \
//	 \   \   \- E - F
//	  \   \
//	   \   \- E - F
//	    \
//	     \- G
func TestPopularities(t *testing.T) {
	storepaths := []Storepath{
		{
			Path:       "A",
			References: []string{"A", "B", "G"},
		},
		{
			Path:       "B",
			References: []string{"B", "C", "E"},
		},
		{
			Path:       "C",
			References: []string{"C", "D", "E"},
		},
		{
			Path:       "D",
			References: []string{"D", "F"},
		},
		{
			Path:       "E",
			References: []string{"E", "F"},
		},
		{
			Path:       "F",
			References: []string{"F"},
		},
		{
			Path:       "G",
			References: []string{"G"},
		},
	}
	popularity, err := SortedPathsByPopularity(storepaths)
	if err != nil {
		panic(err)
	}
	expected := []string{"F", "E", "D", "C", "G", "B", "A"}
	if !reflect.DeepEqual(popularity, expected) {
		t.Fatalf("Popularity should be '%#v' (while it is %#v)", expected, popularity)
	}
}
