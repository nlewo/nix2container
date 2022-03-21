package nix

import (
	"testing"
	"reflect"
)

func TestPathsNotInTar(t *testing.T) {
	paths := []string{
		"../data/file1",
		"../data/file2",
		"/nix/store/1",
		"/nix/store/1/1",
		"/nix/store/1/1/1",
		"/nix/store/1/1/2",
		"/nix/store/1/2",
		"/nix/store/1/2/1",
		"/tmp/1/2",
		"/nix/store/1/2/2",
		"/nix/store/3/4/5/6",
		"/nix/store/3/4",
		"/nix/store/2",
		"/nix/store/2/1",
		"/nix/store/2/1/2",
		"/nix/store/2/1/1",
	}
	expectedRoots := []string{
		"..", "../data", "/", "/nix", "/nix/store",
		"/nix/store/3", "/nix/store/3/4/5", "/tmp", "/tmp/1",
	}
	roots := pathsNotInTar(paths)
	if !reflect.DeepEqual(roots, expectedRoots) {
		t.Fatalf("%#v while should be %#v", roots, expectedRoots)
	}
}
