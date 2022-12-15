package cmd

import (
	"encoding/json"
	"os"

	"github.com/nlewo/nix2container/types"
)

func readPermsFile(filename string) (permPaths []types.PermPath, err error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return permPaths, err
	}
	err = json.Unmarshal(content, &permPaths)
	if err != nil {
		return permPaths, err
	}
	return
}

func readRewritesFile(filename string) (rewritePaths []types.RewritePath, err error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return rewritePaths, err
	}
	err = json.Unmarshal(content, &rewritePaths)
	if err != nil {
		return rewritePaths, err
	}
	return
}
