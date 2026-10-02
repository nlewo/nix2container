package closure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSplit(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "split.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSplitFromFileKeepsOrder(t *testing.T) {
	g := []Storepath{{Path: "/nix/store/a"}, {Path: "/nix/store/b"}, {Path: "/nix/store/c"}}
	split, err := SplitFromFile(g, writeSplit(t, `[["/nix/store/c"],["/nix/store/a","/nix/store/b"]]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(split) != 2 || split[0][0] != "/nix/store/c" || len(split[1]) != 2 {
		t.Fatalf("unexpected split: %v", split)
	}
}

func TestSplitFromFileRejectsIncompleteOrForeignSplits(t *testing.T) {
	g := []Storepath{{Path: "/nix/store/a"}, {Path: "/nix/store/b"}}
	for _, tc := range []struct{ body, want string }{
		{`[["/nix/store/a"]]`, "in no layer"},
		{`[["/nix/store/a","/nix/store/b","/nix/store/x"]]`, "not in the closure graph"},
		{`[["/nix/store/a"],["/nix/store/a","/nix/store/b"]]`, "listed twice"},
		{`[["/nix/store/a","/nix/store/b"],[]]`, "is empty"},
		{`{"a": 1}`, "cannot unmarshal"},
	} {
		_, err := SplitFromFile(g, writeSplit(t, tc.body))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("%s: want error containing %q, got %v", tc.body, tc.want, err)
		}
	}
}
