package nix

import (
	"testing"

	"github.com/nlewo/nix2container/types"
	"github.com/stretchr/testify/assert"
)

func TestUnhackNixCaseHack(t *testing.T) {
	assert.Equal(t, "filename", unhackNixCaseHack("filename"))
	assert.Equal(t, "filename", unhackNixCaseHack("filename~nix~case~hack~1"))
	assert.Equal(t, "filename", unhackNixCaseHack("filename~nix~case~hack~42"))
	assert.Equal(t, "filename~nix~case~hack~0", unhackNixCaseHack("filename~nix~case~hack~0"))
	assert.Equal(t, "filename~nix~case~hack~foo", unhackNixCaseHack("filename~nix~case~hack~foo"))
	assert.Equal(t, "filename~nix~case~hack~0-foobar", unhackNixCaseHack("filename~nix~case~hack~0-foobar"))
}

func TestSplit(t *testing.T) {
	assert.Equal(t, splitPath("/nix"), []string{"", "nix"})
	assert.Equal(t, splitPath("/nix/store"), []string{"", "nix", "store"})
	assert.Equal(t, splitPath("/"), []string{""})
	assert.Equal(t, splitPath("relative"), []string{"relative"})
	assert.Equal(t, splitPath("relative/file"), []string{"relative", "file"})
	assert.Equal(t, splitPath("relative/file/"), []string{"relative", "file"})
}

func TestFilePathToTarPath(t *testing.T) {
	pathOptions := types.PathOptions{
		Rewrite: types.Rewrite{
			Regex: "^/nix/store/x896lxz471i4rgicjxygfh37a0appv7l-nix-database",
			Repl:  ""},
		Perms: []types.Perm(nil),
	}
	path := "/nix/store/x896lxz471i4rgicjxygfh37a0appv7l-nix-database"
	tarPath := filePathToTarPath(path, &pathOptions)
	assert.Equal(t, tarPath, "")

	assert.Equal(t, filePathToTarPath("/", nil), "/")
}
